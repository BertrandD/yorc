package ansible

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"github.com/hashicorp/consul/api"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const output_custom_wrapper = `
[[[printf ". $HOME/.janus/%s/%s/%s" .NodeName .Operation .BasePrimary]]]
[[[range $artName, $art := .Output -]]]
[[[printf "echo %s,$%s >> $HOME/out.csv" $artName $artName]]]
[[[printf "chmod 777 $HOME/out.csv" ]]]
[[[printf "echo $%s" $artName]]]
[[[end]]]
`

const ansible_playbook = `
- name: Executing script {{ script_to_run }}
  hosts: all
  tasks:
    - file: path="{{ ansible_env.HOME}}/.janus/[[[.NodeName]]]/[[[.Operation]]]" state=directory mode=0755
    [[[if .HaveOutput]]]
    [[[printf  "- copy: src=\"{{ wrapper_location }}\" dest=\"{{ ansible_env.HOME}}/.janus/wrapper.sh\" mode=0744" ]]]
    [[[end]]]
    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/.janus/[[[.NodeName]]]/[[[.Operation]]]" mode=0744
    [[[ range $artName, $art := .Artifacts -]]]
    [[[printf "- copy: src=\"%s/%s\" dest=\"{{ ansible_env.HOME}}/.janus/%s/%s/%s\"" $.OverlayPath $art $.NodeName $.Operation (path $art)]]]
    [[[end]]]
    [[[if not .HaveOutput]]]
    [[[printf "- shell: \"{{ ansible_env.HOME}}/.janus/%s/%s/%s\"" .NodeName .Operation .BasePrimary]]][[[else]]]
    [[[printf "- shell: \"/bin/bash -l {{ ansible_env.HOME}}/.janus/wrapper.sh\""]]][[[end]]]
      environment:
        [[[ range $key, $input := .Inputs -]]]
        [[[ if (len $input) gt 0]]][[[printf  "%s: %s" $key $input]]][[[else]]]
        [[[printf  "%s: \"\"" $key]]]
        [[[end]]]
        [[[end]]][[[ range $artName, $art := .Artifacts -]]]
        [[[printf "%s: \"{{ ansible_env.HOME}}/.janus/%v/%v/%s\"" $artName $.NodeName $.Operation $art]]]
        [[[end]]][[[ range $contextK, $contextV := .Context -]]]
        [[[printf "%s: %s" $contextK $contextV]]]
        [[[end]]][[[ range $hostVarIndex, $hostVarValue := .VarInputsNames -]]]
        [[[printf "%s: \"{{%s}}\"" $hostVarValue $hostVarValue]]]
        [[[end]]]
    [[[if .HaveOutput]]]
    [[[printf "- fetch: src={{ ansible_env.HOME}}/out.csv dest={{dest_folder}} flat=yes" ]]]
    [[[end]]]
`

const ansible_config = `[defaults]
host_key_checking=False
timeout=600
`

type execution struct {
	kv                       *api.KV
	DeploymentId             string
	NodeName                 string
	Operation                string
	NodeType                 string
	Inputs                   map[string]string
	PerInstanceInputs        map[string]map[string]string
	VarInputsNames           []string
	Primary                  string
	BasePrimary              string
	Dependencies             []string
	Hosts                    map[string]string
	OperationPath            string
	NodePath                 string
	NodeTypePath             string
	Artifacts                map[string]string
	OverlayPath              string
	Context                  map[string]string
	Output                   map[string]string
	HaveOutput               bool
	isRelationshipOperation  bool
	isRelationshipTargetNode bool
	relationshipType         string
	relationshipTargetName   string
}

func newExecution(kv *api.KV, deploymentId, nodeName, operation string) (*execution, error) {
	execution := &execution{kv: kv,
		DeploymentId:      deploymentId,
		NodeName:          nodeName,
		Operation:         operation,
		PerInstanceInputs: make(map[string]map[string]string),
		VarInputsNames:    make([]string, 0)}
	return execution, execution.resolveExecution()
}

func (e *execution) resolveArtifacts() error {
	log.Debugf("Resolving artifacts")
	artifacts := make(map[string]string)
	// First resolve node type artifacts then node template artifact if the is a conflict then node template will have the precedence
	// TODO deal with type inheritance
	// TODO deal with relationships
	paths := []string{path.Join(e.NodePath, "artifacts"), path.Join(e.NodeTypePath, "artifacts")}
	for _, apath := range paths {
		artsPaths, _, err := e.kv.Keys(apath+"/", "/", nil)
		if err != nil {
			return err
		}
		for _, artPath := range artsPaths {
			kvp, _, err := e.kv.Get(path.Join(artPath, "name"), nil)
			if err != nil {
				return err
			}
			if kvp == nil {
				return fmt.Errorf("Missing mandatory key in consul %q", path.Join(artPath, "name"))
			}
			artName := string(kvp.Value)
			kvp, _, err = e.kv.Get(path.Join(artPath, "file"), nil)
			if err != nil {
				return err
			}
			if kvp == nil {
				return fmt.Errorf("Missing mandatory key in consul %q", path.Join(artPath, "file"))
			}
			artifacts[artName] = string(kvp.Value)
		}
	}

	e.Artifacts = artifacts
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return nil
}

func (e *execution) resolveInputs() error {
	log.Debug("resolving inputs")
	resolver := deployments.NewResolver(e.kv, e.DeploymentId)
	inputs := make(map[string]string)
	inputKeys, _, err := e.kv.Keys(e.OperationPath+"/inputs/", "/", nil)
	if err != nil {
		return err
	}
	for _, input := range inputKeys {
		kvPair, _, err := e.kv.Get(input+"/name", nil)
		if err != nil {
			return err
		}
		if kvPair == nil {
			return fmt.Errorf("%s/name missing", input)
		}
		inputName := string(kvPair.Value)
		kvPair, _, err = e.kv.Get(input+"/expression", nil)
		if err != nil {
			return err
		}
		if kvPair == nil {
			return fmt.Errorf("%s/expression missing", input)
		}
		va := tosca.ValueAssignment{}
		yaml.Unmarshal(kvPair.Value, &va)
		instancesNames, err := deployments.GetNodeInstancesNames(e.kv, e.DeploymentId, e.NodeName)
		if err != nil {
			return err
		}
		var inputValue string
		if len(instancesNames) > 0 {
			for i, instanceName := range instancesNames {
				targetContext := false
				var instanceInputs map[string]string
				ok := false
				if instanceInputs, ok = e.PerInstanceInputs[instanceName]; !ok {
					instanceInputs = make(map[string]string)
					e.PerInstanceInputs[instanceName] = instanceInputs
				}
				if e.isRelationshipOperation {
					targetContext, inputValue, err = resolver.ResolveExpressionForRelationship(va.Expression, e.NodeName, e.relationshipTargetName, e.relationshipType, instanceName)
				} else {
					inputValue, err = resolver.ResolveExpressionForNode(va.Expression, e.NodeName, instanceName)
				}
				if err != nil {
					return err
				}
				if e.isRelationshipOperation && targetContext {
					inputs[replaceMinus(e.relationshipTargetName+"_"+instanceName+"_"+inputName)] = inputValue
				} else {
					inputs[replaceMinus(e.NodeName+"_"+instanceName+"_"+inputName)] = inputValue
				}
				instanceInputs[replaceMinus(inputName)] = inputValue
				if i == 0 {
					e.VarInputsNames = append(e.VarInputsNames, replaceMinus(inputName))
				}
			}

		} else {
			if e.isRelationshipOperation {
				_, inputValue, err = resolver.ResolveExpressionForRelationship(va.Expression, e.NodeName, e.relationshipTargetName, e.relationshipType, "")
			} else {
				inputValue, err = resolver.ResolveExpressionForNode(va.Expression, e.NodeName, "")
			}
			if err != nil {
				return err
			}
			inputs[replaceMinus(inputName)] = inputValue
		}
	}
	e.Inputs = inputs

	log.Debugf("Resolved inputs: %v", e.Inputs)
	return nil
}

func (e *execution) resolveHosts(nodeName string) error {
	// e.nodePath
	instancesPath := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/instances", nodeName)
	log.Debugf("Resolving hosts for node %q", nodeName)

	hosts := make(map[string]string)
	instances, err := deployments.GetNodeInstancesNames(e.kv, e.DeploymentId, nodeName)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		kvp, _, err := e.kv.Get(path.Join(instancesPath, instance, "capabilities/endpoint/attributes/ip_address"), nil)
		if err != nil {
			return err
		}
		if kvp != nil && len(kvp.Value) != 0 {
			hosts[instance] = string(kvp.Value)
		}
	}
	if len(hosts) == 0 {
		// So we have to traverse the HostedOn relationships...
		hostedOnNode, err := deployments.GetHostedOnNode(e.kv, e.DeploymentId, nodeName)
		if err != nil {
			return err
		}
		if hostedOnNode == "" {
			return fmt.Errorf("Can't find an Host with an ip_address in the HostedOn hierarchy for node %q in deployment %q", e.NodeName, e.DeploymentId)
		}
		return e.resolveHosts(hostedOnNode)
	}
	e.Hosts = hosts
	return nil
}

func replaceMinus(str string) string {
	return strings.Replace(str, "-", "_", -1)
}

func (e *execution) resolveContext() error {
	execContext := make(map[string]string)

	//TODO: Need to be improved with the runtime (INSTANCE,INSTANCES)
	new_node := replaceMinus(e.NodeName)
	execContext["NODE"] = new_node
	names, err := deployments.GetNodeInstancesNames(e.kv, e.DeploymentId, e.NodeName)
	if err != nil {
		return err
	}
	for i := range names {
		var instanceInputs map[string]string
		ok := false
		if instanceInputs, ok = e.PerInstanceInputs[names[i]]; !ok {
			instanceInputs = make(map[string]string)
			e.PerInstanceInputs[names[i]] = instanceInputs
		}
		newName := new_node + "_" + replaceMinus(names[i])
		instanceInputs["INSTANCE"] = newName
		if i == 0 {
			e.VarInputsNames = append(e.VarInputsNames, "INSTANCE")
		}
		if e.isRelationshipOperation {
			instanceInputs["SOURCE_INSTANCE"] = newName
			if i == 0 {
				e.VarInputsNames = append(e.VarInputsNames, "SOURCE_INSTANCE")
			}
		}
		names[i] = newName
	}
	execContext["INSTANCES"] = strings.Join(names, ",")
	if host, err := deployments.GetHostedOnNode(e.kv, e.DeploymentId, e.NodeName); err != nil {
		return err
	} else if host != "" {
		execContext["HOST"] = host
	}

	if e.isRelationshipOperation {

		execContext["SOURCE_NODE"] = new_node
		execContext["SOURCE_INSTANCE"] = new_node
		execContext["SOURCE_INSTANCES"] = strings.Join(names, ",")

		execContext["TARGET_NODE"] = replaceMinus(e.relationshipTargetName)
		execContext["TARGET_INSTANCE"] = replaceMinus(e.relationshipTargetName)
		targetNames, err := deployments.GetNodeInstancesNames(e.kv, e.DeploymentId, e.relationshipTargetName)
		if err != nil {
			return err
		}
		for i := range targetNames {
			targetNames[i] = replaceMinus(e.relationshipTargetName + "_" + targetNames[i])
		}
		execContext["TARGET_INSTANCES"] = strings.Join(targetNames, ",")

	}

	e.Context = execContext

	return nil
}

func (e *execution) resolveOperationOutput() error {
	log.Debugf(e.OperationPath)
	log.Debugf(e.Operation)

	//We get all the output of the NodeType
	pathList, _, err := e.kv.Keys(e.NodeTypePath+"/output/", "", nil)

	if err != nil {
		return err
	}

	output := make(map[string]string)

	//For each type we compare if we are in the good lifecycle operation
	for _, path := range pathList {
		tmp := strings.Split(e.Operation, ".")
		if strings.Contains(path, tmp[len(tmp)-1]) {
			nodeOutPath := filepath.Join(e.NodePath, "attributes", strings.ToLower(filepath.Base(path)))
			e.HaveOutput = true
			output[filepath.Base(path)] = nodeOutPath
		}
	}

	log.Debugf("%v", output)
	e.Output = output
	return nil
}

// isTargetOperation returns true if the given operationName contains one of the following patterns (case doesn't matter):
//     pre_configure_target, post_configure_target, add_target, target_changed or remove_target
func isTargetOperation(operationName string) bool {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "pre_configure_target") || strings.Contains(op, "post_configure_target") || strings.Contains(op, "add_target") || strings.Contains(op, "target_changed") || strings.Contains(op, "remove_target") {
		return true
	}
	return false
}

func (e *execution) resolveExecution() error {
	log.Printf("Preparing execution of operation %q on node %q for deployment %q", e.Operation, e.NodeName, e.DeploymentId)
	ovPath, err := filepath.Abs(filepath.Join("work", "deployments", e.DeploymentId, "overlay"))
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath
	e.NodePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", e.NodeName)
	kvPair, _, err := e.kv.Get(e.NodePath+"/type", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("type for node %s in deployment %s is missing", e.NodeName, e.DeploymentId)
	}

	e.NodeType = string(kvPair.Value)
	e.NodeTypePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", e.NodeType)
	if strings.Contains(e.Operation, "Standard") {
		e.isRelationshipOperation = false
	} else {
		// In a relationship
		e.isRelationshipOperation = true
		opAndReq := strings.Split(e.Operation, "/")
		e.isRelationshipTargetNode = false
		if isTargetOperation(opAndReq[0]) {
			e.isRelationshipTargetNode = true
		}
		if len(opAndReq) == 2 {
			reqName := opAndReq[1]
			kvPair, _, err := e.kv.Get(path.Join(e.NodePath, "requirements", reqName, "relationship"), nil)
			if err != nil {
				return err
			}
			if kvPair == nil {
				return fmt.Errorf("Requirement %q for node %q in deployment %q is missing", reqName, e.NodeName, e.DeploymentId)
			}
			e.relationshipType = string(kvPair.Value)
			kvPair, _, err = e.kv.Get(path.Join(e.NodePath, "requirements", reqName, "node"), nil)
			if err != nil {
				return err
			}
			if kvPair == nil {
				return fmt.Errorf("Requirement %q for node %q in deployment %q is missing", reqName, e.NodeName, e.DeploymentId)
			}
			e.relationshipTargetName = string(kvPair.Value)
		} else {
			// Old way if requirement is not specified get the last one
			// TODO remove this part
			kvPair, _, err := e.kv.Keys(path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", e.NodeName, "requirements"), "", nil)
			if err != nil {
				return err
			}
			for _, key := range kvPair {
				if strings.HasSuffix(key, "relationship") && !strings.Contains(key, "/host/") {
					kvPair, _, err := e.kv.Get(path.Join(key), nil)
					if err != nil {
						return err
					}
					e.relationshipType = string(kvPair.Value)
				}
				if strings.HasSuffix(key, "node") && !strings.Contains(key, "/host/") {
					kvPair, _, err := e.kv.Get(path.Join(key), nil)
					if err != nil {
						return err
					}
					e.relationshipTargetName = string(kvPair.Value)
				}
			}
		}
	}

	//TODO deal with inheritance operation may be not in the direct node type
	if e.isRelationshipOperation {
		idx := strings.Index(e.Operation, "Configure.")
		var op string
		if idx >= 0 {
			op = e.Operation[idx:]
		} else {
			op = strings.TrimPrefix(e.Operation, "tosca.interfaces.node.lifecycle.")
			op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
		}
		e.OperationPath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", e.relationshipType) + "/interfaces/" + strings.Replace(op, ".", "/", -1)
	} else {
		idx := strings.Index(e.Operation, "Standard.")
		var op string
		if idx >= 0 {
			op = e.Operation[idx:]
		} else {
			op = strings.TrimPrefix(e.Operation, "tosca.interfaces.node.lifecycle.")
			op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
		}
		e.OperationPath = e.NodeTypePath + "/interfaces/" + strings.Replace(op, ".", "/", -1)
	}
	log.Debugf("Operation Path: %q", e.OperationPath)
	kvPair, _, err = e.kv.Get(e.OperationPath+"/implementation/primary", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("primary implementation missing for type %s in deployment %s is missing", e.NodeType, e.DeploymentId)
	}
	e.Primary = string(kvPair.Value)
	e.BasePrimary = path.Base(e.Primary)
	kvPair, _, err = e.kv.Get(e.OperationPath+"/implementation/dependencies", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("dependencies implementation missing for type %s in deployment %s is missing", e.NodeType, e.DeploymentId)
	}
	e.Dependencies = strings.Split(string(kvPair.Value), ",")

	if err = e.resolveInputs(); err != nil {
		return err
	}
	if err = e.resolveArtifacts(); err != nil {
		return err
	}
	if e.isRelationshipTargetNode {
		err = e.resolveHosts(e.relationshipTargetName)
	} else {
		err = e.resolveHosts(e.NodeName)
	}
	if err != nil {
		return err
	}
	if err = e.resolveOperationOutput(); err != nil {
		return err
	}

	return e.resolveContext()

}

func (e *execution) execute(ctx context.Context) error {

	ansibleRecipePath := filepath.Join("work", "deployments", e.DeploymentId, "ansible", e.NodeName, e.Operation)
	ansibleHostVarsPath := filepath.Join("work", "deployments", e.DeploymentId, "ansible", e.NodeName, e.Operation, "host_vars")
	if err := os.MkdirAll(ansibleRecipePath, 0775); err != nil {
		log.Printf("%+v", err)
		return err
	}
	if err := os.MkdirAll(ansibleHostVarsPath, 0775); err != nil {
		log.Printf("%+v", err)
		return err
	}
	var buffer bytes.Buffer
	buffer.WriteString("[all]\n")
	for instanceName, host := range e.Hosts {
		buffer.WriteString(host)
		// TODO should not be hard-coded
		buffer.WriteString(" ansible_ssh_user=cloud-user ansible_ssh_private_key_file=~/.ssh/janus.pem\n")

		var perInstanceInputsBuffer bytes.Buffer
		if instanceInputs, ok := e.PerInstanceInputs[instanceName]; ok {
			for inputName, inputValue := range instanceInputs {
				perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: %s\n", inputName, inputValue))
			}
		}
		if perInstanceInputsBuffer.Len() > 0 {
			if err := ioutil.WriteFile(filepath.Join(ansibleHostVarsPath, host+".yml"), perInstanceInputsBuffer.Bytes(), 0664); err != nil {
				log.Printf("Failed to write vars for host %q file: %v", host, err)
				return err
			}
		}
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "hosts"), buffer.Bytes(), 0664); err != nil {
		log.Print("Failed to write hosts file")
		return err
	}
	buffer.Reset()
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
		"abs":  filepath.Abs,
	}
	tmpl := template.New("execTemplate")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	if e.HaveOutput {
		wrap_template := template.New("execTemplate")
		wrap_template = wrap_template.Delims("[[[", "]]]")
		wrap_template, err := tmpl.Parse(output_custom_wrapper)
		if err != nil {
			return err
		}
		var buffer bytes.Buffer
		if err := wrap_template.Execute(&buffer, e); err != nil {
			log.Print("Failed to Generate wrapper template")
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "wrapper.sh"), buffer.Bytes(), 0664); err != nil {
			log.Print("Failed to write playbook file")
			return err
		}
	}
	tmpl, err := tmpl.Parse(ansible_playbook)
	if err := tmpl.Execute(&buffer, e); err != nil {
		log.Print("Failed to Generate ansible playbook template")
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		log.Print("Failed to write playbook file")
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "ansible.cfg"), []byte(ansible_config), 0664); err != nil {
		log.Print("Failed to write ansible.cfg file")
		return err
	}
	scriptPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}
	log.Printf("Ansible recipe for deployment with id %s: executing %q on remote host", e.DeploymentId, scriptPath)
	var cmd *exec.Cmd
	var wrapperPath string
	if e.HaveOutput {
		wrapperPath, _ = filepath.Abs(ansibleRecipePath)
		cmd = exec.CommandContext(ctx, "ansible-playbook", "-v", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s , wrapper_location=%s/wrapper.sh , dest_folder=%s", scriptPath, wrapperPath, wrapperPath))
	} else {
		cmd = exec.CommandContext(ctx, "ansible-playbook", "-v", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s", scriptPath))
	}
	cmd.Dir = ansibleRecipePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if e.HaveOutput {
		if err := cmd.Run(); err != nil {
			log.Print(err)
			return err
		}
		fi, err := os.Open(filepath.Join(wrapperPath, "out.csv"))
		if err != nil {
			panic(err)
		}
		r := csv.NewReader(fi)
		records, err := r.ReadAll()
		if err != nil {
			log.Fatal(err)
		}
		for _, line := range records {
			storeConsulKey(e.kv, e.Output[line[0]], line[1])
		}
		return nil

	} else {
		if err := cmd.Start(); err != nil {
			log.Print(err)
			return err
		}
	}

	return cmd.Wait()
}

func storeConsulKey(kv *api.KV, key, value string) {
	// PUT a new KV pair
	p := &api.KVPair{Key: key, Value: []byte(value)}
	if _, err := kv.Put(p, nil); err != nil {
		log.Panic(err)
	}
}
