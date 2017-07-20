package kubernetes

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/operations"
	"novaforge.bull.com/starlings-janus/janus/prov/structs"
	"path"
	"strings"
	"time"
	"novaforge.bull.com/starlings-janus/janus/events"
)

// An EnvInput represent a TOSCA operation input
//
// This element is exported in order to be used by text.Template but should be consider as internal

type execution interface {
	execute(ctx context.Context) error
}

type executionScript struct {
	*executionCommon
}

type executionCommon struct {
	kv                  *api.KV
	cfg                 config.Configuration
	deploymentID        string
	taskID              string
	NodeName            string
	Operation           prov.Operation
	NodeType            string
	Description         string
	OperationRemotePath string
	OperationPath       string
	EnvInputs           []*structs.EnvInput
	VarInputsNames      []string
	Repositories        map[string]string
	NodePath            string
	NodeTypePath        string
	Artifacts           map[string]string
	OverlayPath         string
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) (execution, error) {
	execCommon := &executionCommon{kv: kv,
		cfg:            cfg,
		deploymentID:   deploymentID,
		NodeName:       nodeName,
		Operation:      operation,
		VarInputsNames: make([]string, 0),
		EnvInputs:      make([]*structs.EnvInput, 0),
		taskID:         taskID,
	}

	return execCommon, execCommon.resolveOperation()
}

func (e *executionCommon) resolveOperation() error {
	e.NodePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/nodes", e.NodeName)
	var err error
	e.NodeType, err = deployments.GetNodeType(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	e.NodeTypePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/types", e.NodeType)
	operationNodeType := e.NodeType

	e.OperationPath = deployments.GetOperationPath(e.deploymentID, operationNodeType, e.Operation.Name)
	if err != nil {
		return err
	}

	return nil
}

func (e *executionCommon) execute(ctx context.Context) (err error) {
	switch strings.ToLower(e.Operation.Name) {
	case "tosca.interfaces.node.lifecycle.standard.delete",
		"tosca.interfaces.node.lifecycle.standard.configure":
		log.Printf("Voluntary bypassing operation %s", e.Operation.Name)
		return nil
	case "tosca.interfaces.node.lifecycle.standard.start":
		err = e.deployNode(ctx)
		if err != nil {
			return err
		}
		return e.checkNode(ctx)
	case "tosca.interfaces.node.lifecycle.standard.stop":
		return e.uninstallNode(ctx)
	default:
		return errors.Errorf("Unsupported operation %q", e.Operation.Name)
	}

}

func (e *executionCommon) parseEnvInputs() []v1.EnvVar {
	var data []v1.EnvVar

	for _, val := range e.EnvInputs {
		tmp := v1.EnvVar{Name: val.Name, Value: val.Value}
		data = append(data, tmp)
	}

	return data
}

func (e *executionCommon) deployNode(ctx context.Context) error {
	clientset := ctx.Value("clientset")
	generator := NewGenerator(e.kv, e.cfg)

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	namespace = strings.ToLower(namespace)
	err = generator.CreateNamespaceIfMissing(e.deploymentID, namespace, clientset.(*kubernetes.Clientset))
	if err != nil {
		return err
	}

	e.EnvInputs, e.VarInputsNames, err = operations.InputsResolver(e.kv, e.OperationPath, e.deploymentID, e.NodeName, e.taskID, e.Operation.Name)
	inputs := e.parseEnvInputs()

	deployment, service, err := generator.GenerateDeployment(e.deploymentID, e.NodeName, e.Operation.Name, e.NodeType, inputs)
	if err != nil {
		return err
	}

	_, err = (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Create(&deployment)

	if err != nil {
		return errors.Wrap(err, "Failed to create deployment")
	}

	if service.Name != "" {
		serv, err := (clientset.(*kubernetes.Clientset)).CoreV1().Services(namespace).Create(&service)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
		for _, val := range serv.Spec.Ports {
			log.Printf("%s : %s: %d:%d mapped to %d", serv.Name, val.Name, val.Port, val.TargetPort.IntVal, val.NodePort)
		}
	}

	return nil
}

func (e *executionCommon) checkNode(ctx context.Context) error {
	clientset := ctx.Value("clientset")

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	deploymentReady := false

	for !deploymentReady {
		deployment, err := (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(strings.ToLower(namespace)).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed fetch deployment")
		}
		log.Printf("Deployment %s : %d pod available of %d", e.NodeName, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
			deploymentReady = true
		} else {
			selector := ""
			for key, val := range deployment.Spec.Selector.MatchLabels {
				if selector != "" {
					selector += ","
				}
				selector += key + "=" + val
			}
			//log.Printf("selector: %s", selector)
			pods, _ := (clientset.(*kubernetes.Clientset)).CoreV1().Pods(namespace).List(
				metav1.ListOptions{
					LabelSelector: selector,
				})

			// We should always have only 1 pod (as the Replica is set to 1)
			for _, podItem := range pods.Items {

				//log.Printf("Check pod %s", podItem.Name)
				err := e.checkPod(ctx, podItem.Name)
				if err != nil {
					return err
				}

			}
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

func (e *executionCommon) checkPod(ctx context.Context, podName string) error {
	clientset := ctx.Value("clientset")

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	pod := v1.Pod{}
	status := v1.PodUnknown
	latestReason := ""

	for status != v1.PodRunning && latestReason != "ErrImagePull" && latestReason != "InvalidImageName" {
		pod, err := (clientset.(*kubernetes.Clientset)).CoreV1().Pods(strings.ToLower(namespace)).Get(podName, metav1.GetOptions{})

		if err != nil {
			return errors.Wrap(err, "Failed to fetch pod")
		}

		status = pod.Status.Phase

		if status == v1.PodPending && len(pod.Status.ContainerStatuses) > 0 {
			reason := pod.Status.ContainerStatuses[0].State.Waiting.Reason
			if reason != latestReason {
				latestReason = reason
				log.Printf(pod.Name + " : " + string(pod.Status.Phase) + "->" + reason)
				events.LogEngineMessage(e.kv, e.deploymentID, "Pod status : "+pod.Name+" : "+string(pod.Status.Phase)+" -> "+reason)
			}
		} else {
			log.Printf(pod.Name + " : " + string(pod.Status.Phase))
			events.LogEngineMessage(e.kv, e.deploymentID, "Pod status : "+pod.Name+" : "+string(pod.Status.Phase))
		}

		time.Sleep(2 * time.Second)
	}

	ready := true
	cond := v1.PodCondition{}
	for _, condition := range pod.Status.Conditions {
		if condition.Status == v1.ConditionFalse {
			ready = false
			cond = condition
		}
	}

	if !ready {
		reason := pod.Status.ContainerStatuses[0].State.Waiting.Reason
		message := pod.Status.ContainerStatuses[0].State.Waiting.Message

		if reason == "RunContainerError" {
			logs, err := (clientset.(*kubernetes.Clientset)).CoreV1().Pods(strings.ToLower(namespace)).GetLogs(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), &v1.PodLogOptions{}).Do().Raw()
			if err != nil {
				return errors.Wrap(err, "Failed to fetch pod logs")
			}
			podLogs := string(logs)
			return errors.Errorf("Pod failed to start reason : %s --- Message : %s --- Pod logs : %s", reason, message, podLogs)
		}

		return errors.Errorf("Pod failed to start reason : %s --- Message : %s -- condition : %s", reason, message, cond.Message)
	}

	return nil
}


func (e *executionCommon) uninstallNode(ctx context.Context) error {
	clientset := ctx.Value("clientset")

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	deployment, err := (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(strings.ToLower(namespace)).Get(strings.ToLower(e.cfg.ResourcesPrefix + e.NodeName), metav1.GetOptions{})

	replica := int32(0)
	deployment.Spec.Replicas = &replica
	_, err = (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(strings.ToLower(namespace)).Update(deployment)

	err = (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(strings.ToLower(namespace)).Delete(strings.ToLower(e.cfg.ResourcesPrefix + e.NodeName), &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to delete deployment")
	}

	err = (clientset.(*kubernetes.Clientset)).CoreV1().Services(strings.ToLower(namespace)).Delete(strings.ToLower(GeneratePodName(e.cfg.ResourcesPrefix+e.NodeName)), &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to delete service")
	}

	return nil
}

func getNamespace(kv *api.KV, deploymentID, nodeName string) (string, error) {
	found, namespace, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "namespace")
	if err != nil {
		return "", err
	}
	if !found || namespace == "" {
		return deployments.GetDeploymentTemplateName(kv, deploymentID)
	}
	return namespace, nil
}
