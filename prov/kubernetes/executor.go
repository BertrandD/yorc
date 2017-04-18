package kubernetes

import (
	"context"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"novaforge.bull.com/starlings-janus/janus/prov"
	"k8s.io/apimachinery/pkg/api/resource"
)

type defaultExecutor struct {
	clientset *kubernetes.Clientset
}

// NewExecutor returns an Executor
func NewExecutor() prov.DelegateExecutor {

	var clientset *kubernetes.Clientset
	conf, err := clientcmd.BuildConfigFromFlags("", "./kubconfig.yaml")
	if err != nil {
		panic(err)
	}
	clientset, err = kubernetes.NewForConfig(conf)

	return &defaultExecutor{clientset: clientset}
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(nodeType, "janus.nodes.KubernetesContainer") {
		return errors.Errorf("Unsupported node type '%s' for node '%s'", nodeType, nodeName)
	}

	op := strings.ToLower(delegateOperation)
	switch {
	case op == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName)
	case op == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}

	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {

	found, dockerImage, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "image")
	if err != nil {
		return err
	}
	if !found || dockerImage == ""{
		return errors.Errorf("Property image not found on node %s", nodeName)
	}

	found, namespace, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "namespace")

	if !found || namespace == "" {
		namespace = "default"
	}

	_, cpuShareStr, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "cpu_share")
	_, cpuLimitStr, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "cpu_limit")
	_, memShareStr, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "mem_share")
	_, memLimitStr, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "mem_limit")
	_, imagePullPolicy, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "imagePullPolicy")
	_, dockerRunCmd, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "docker_run_cmd")

	cpuLimit, err := resource.ParseQuantity(cpuLimitStr)
	cpuShare, _ := resource.ParseQuantity(cpuShareStr)
	memLimit, _ := resource.ParseQuantity(memLimitStr)
	memShare, _ := resource.ParseQuantity(memShareStr)

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   strings.ToLower(nodeName),
			Labels: map[string]string{"name": strings.ToLower(nodeName)},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  strings.ToLower(nodeName),
					Image: dockerImage,
					ImagePullPolicy: v1.PullPolicy(imagePullPolicy),
					Command: strings.Fields(dockerRunCmd),
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU: cpuShare,
							v1.ResourceMemory: memShare,
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU: cpuLimit,
							v1.ResourceMemory: memLimit,
						},
					},
				},
			},
		},
	}

	_, err = e.clientset.CoreV1().Pods(namespace).Create(&pod)
	return err
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
	found, namespace, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "namespace")
	if err != nil {
		return err
	}
	if !found || namespace == "" {
		namespace = "default"
	}

	err = e.clientset.CoreV1().Pods(namespace).Delete(strings.ToLower(nodeName), &metav1.DeleteOptions{})
	return err
}
