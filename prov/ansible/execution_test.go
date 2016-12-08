package ansible

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"text/template"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func TestAnsibleParallel(t *testing.T) {
	t.Run("ansible", func(t *testing.T) {
		t.Run("templatesTest", templatesTest)
		t.Run("testExecution_OnNode", testExecution_OnNode)
		t.Run("testExecution_OnRelationshipSource", testExecution_OnRelationshipSource)
		t.Run("testExecution_OnRelationshipTarget", testExecution_OnRelationshipTarget)
	})
}

func templatesTest(t *testing.T) {
	t.Parallel()
	ec := &executionCommon{
		NodeName:            "Welcome",
		Operation:           "tosca.interfaces.node.lifecycle.standard.start",
		Artifacts:           map[string]string{"scripts": "my_scripts"},
		OverlayPath:         "/some/local/path",
		VarInputsNames:      []string{"INSTANCE", "PORT"},
		OperationRemotePath: ".janus/path/on/remote",
	}

	e := &executionScript{
		executionCommon: ec,
	}

	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}

	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err := tmpl.Parse(shell_ansible_playbook)
	require.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	require.Nil(t, err)
}

func testExecution_OnNode(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()
	deploymentId := "d1"
	nodeName := "NodeA"
	nodeTypeName := "janus.types.A"
	operation := "tosca.interfaces.node.lifecycle.standard.create"

	srv1.PopulateKV(map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "name"):                                                        []byte(nodeTypeName),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/name"):                   []byte("A1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/expression"):             []byte("get_property: [SELF, document_root]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/name"):                   []byte("A3"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/expression"):             []byte("get_property: [SELF, empty]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/name"):                   []byte("A2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/expression"):             []byte("get_attribute: [HOST, ip_address]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/implementation/primary"):           []byte("/tmp/create.sh"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/implementation/dependencies"):      []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/is_property_definition"): []byte("false"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "type"): []byte(nodeTypeName),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "properties/document_root"):    []byte("/var/www"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "properties/empty"):            []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "requirements/0/node"):         []byte("Compute"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/Compute/0/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/Compute/1/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/Compute/2/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/Compute/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/Compute/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/Compute/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeName, "0/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeName, "1/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeName, "2/status"): []byte("initial"),
	})

	t.Run("testExecution_ResolveInputsOnNode", func(t *testing.T) {
		testExecution_ResolveInputsOnNode(t, kv, deploymentId, nodeName, nodeTypeName, operation)
	})
	t.Run("testExecution_GenerateOnNode", func(t *testing.T) {
		testExecution_GenerateOnNode(t, kv, deploymentId, nodeName, operation)
	})

}

func testExecution_ResolveInputsOnNode(t *testing.T, kv *api.KV, deploymentId, nodeName, nodeTypeName, operation string) {
	execution := &executionCommon{kv: kv,
		DeploymentId:            deploymentId,
		NodeName:                nodeName,
		Operation:               operation,
		OperationPath:           path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", nodeTypeName, "interfaces/standard/create"),
		isRelationshipOperation: false,
		isPerInstanceOperation:  false,
		VarInputsNames:          make([]string, 0),
		EnvInputs:               make([]*EnvInput, 0)}

	err := execution.resolveInputs()
	require.Nil(t, err)
	require.Len(t, execution.EnvInputs, 9)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2":

			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "10.10.10.1", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "10.10.10.2", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "10.10.10.3", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A3":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 9)
}

func compareStringsIgnoreWhitespace(t *testing.T, expected, actual string) {
	re_leadclose_whtsp := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
	re_inside_whtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	expected = re_leadclose_whtsp.ReplaceAllString(expected, "")
	expected = re_inside_whtsp.ReplaceAllString(expected, " ")
	actual = re_leadclose_whtsp.ReplaceAllString(actual, "")
	actual = re_inside_whtsp.ReplaceAllString(actual, " ")
	require.Equal(t, expected, actual)
}

func testExecution_GenerateOnNode(t *testing.T, kv *api.KV, deploymentId, nodeName, operation string) {

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/create.sh"
      environment:
        NodeA_0_A1: /var/www
        NodeA_1_A1: /var/www
        NodeA_2_A1: /var/www
        NodeA_0_A2: 10.10.10.1
        NodeA_1_A2: 10.10.10.2
        NodeA_2_A2: 10.10.10.3
        NodeA_0_A3: ""
        NodeA_1_A3: ""
        NodeA_2_A3: ""
        HOST: Compute
        INSTANCES: NodeA_0,NodeA_1,NodeA_2
        NODE: NodeA
        A1: "{{A1}}"
        A2: "{{A2}}"
        A3: "{{A3}}"
        INSTANCE: "{{INSTANCE}}"


`
	execution, err := newExecution(kv, deploymentId, nodeName, operation)
	require.Nil(t, err)

	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemotePath = "tmp"
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}
	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err = tmpl.Parse(shell_ansible_playbook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}

func testExecution_OnRelationshipSource(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()
	deploymentId := "d1"
	nodeAName := "NodeA"
	relationshipTypeName := "janus.types.Rel"
	nodeBName := "NodeB"

	var operationTestCases = []string{
		"tosca.interfaces.node.lifecycle.Configure.pre_configure_source/1",
		"tosca.interfaces.node.lifecycle.Configure.pre_configure_source/connect/" + nodeBName,
	}

	srv1.PopulateKV(map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types/janus.types.A/name"):                                                                                  []byte("janus.types.A"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "name"):                                                                       []byte(relationshipTypeName),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A1/name"):                   []byte("A1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A1/expression"):             []byte("get_property: [SOURCE, document_root]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A2/name"):                   []byte("A2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A2/expression"):             []byte("get_attribute: [TARGET, ip_address]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/implementation/primary"):           []byte("/tmp/pre_configure_source.sh"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/implementation/dependencies"):      []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A1/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A2/is_property_definition"): []byte("false"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types/janus.types.B/name"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "properties/document_root"): []byte("/var/www"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "type"):                     []byte("janus.types.A"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/node"):         []byte("ComputeA"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/1/name"):         []byte("connect"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/1/node"):         []byte("NodeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/1/relationship"): []byte(relationshipTypeName),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "type"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/node"):         []byte("ComputeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/0/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/1/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/2/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/0/attributes/ip_address"): []byte("10.10.10.10"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/1/attributes/ip_address"): []byte("10.10.10.11"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.10"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.11"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeAName, "0/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeAName, "1/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeAName, "2/status"): []byte("initial"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeBName, "0/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeBName, "1/status"): []byte("initial"),
	})
	for i, operation := range operationTestCases {
		t.Run("testExecution_ResolveInputsOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecution_ResolveInputsOnRelationshipSource(t, kv, deploymentId, nodeAName, nodeBName, operation, relationshipTypeName)
		})
		t.Run("testExecution_GenerateOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecution_GenerateOnRelationshipSource(t, kv, deploymentId, nodeAName, operation)
		})
	}
}

func testExecution_ResolveInputsOnRelationshipSource(t *testing.T, kv *api.KV, deploymentId, nodeAName, nodeBName, operation, relationshipTypeName string) {

	execution := &executionCommon{kv: kv,
		DeploymentId:            deploymentId,
		NodeName:                nodeAName,
		Operation:               operation,
		OperationPath:           path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source"),
		isRelationshipOperation: true,
		isPerInstanceOperation:  false,
		requirementIndex:        "1",
		relationshipType:        relationshipTypeName,
		relationshipTargetName:  nodeBName,
		VarInputsNames:          make([]string, 0),
		EnvInputs:               make([]*EnvInput, 0)}

	err := execution.resolveInputs()
	require.Nil(t, err)
	require.Len(t, execution.EnvInputs, 5)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2":

			switch envInput.InstanceName {
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 5)
}

func testExecution_GenerateOnRelationshipSource(t *testing.T, kv *api.KV, deploymentId, nodeName, operation string) {

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/pre_configure_source.sh"
      environment:
        NodeA_0_A1: /var/www
        NodeA_1_A1: /var/www
        NodeA_2_A1: /var/www
        NodeB_0_A2: 10.10.10.10
        NodeB_1_A2: 10.10.10.11
        SOURCE_HOST: ComputeA
        SOURCE_INSTANCES: NodeA_0,NodeA_1,NodeA_2
        SOURCE_NODE: NodeA
        TARGET_HOST: ComputeB
        TARGET_INSTANCE: NodeB_0
        TARGET_INSTANCES: NodeB_0,NodeB_1
        TARGET_NODE: NodeB
        A1: "{{A1}}"
        A2: "{{A2}}"
        SOURCE_INSTANCE: "{{SOURCE_INSTANCE}}"


`
	execution, err := newExecution(kv, deploymentId, nodeName, operation)
	require.Nil(t, err)

	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemotePath = "tmp"
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}
	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err = tmpl.Parse(shell_ansible_playbook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}

func testExecution_OnRelationshipTarget(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()
	deploymentId := "d1"
	nodeAName := "NodeA"
	relationshipTypeName := "janus.types.Rel"
	nodeBName := "NodeB"

	var operationTestCases = []string{
		"tosca.interfaces.node.lifecycle.Configure.add_source/1",
		"tosca.interfaces.node.lifecycle.Configure.add_source/connect/" + nodeBName,
	}

	srv1.PopulateKV(map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types/janus.types.A/name"):                                                                        []byte("janus.types.A"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "name"):                                                             []byte(relationshipTypeName),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A1/name"):                   []byte("A1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A1/expression"):             []byte("get_property: [SOURCE, document_root]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A2/name"):                   []byte("A2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A2/expression"):             []byte("get_attribute: [TARGET, ip_address]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/implementation/primary"):           []byte("/tmp/add_source.sh"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/implementation/dependencies"):      []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A1/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A2/is_property_definition"): []byte("false"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types/janus.types.B/name"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "properties/document_root"): []byte("/var/www"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "type"):                     []byte("janus.types.A"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/node"):         []byte("ComputeA"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "type"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/node"):         []byte("ComputeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeBName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/1/name"):         []byte("connect"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/1/node"):         []byte("NodeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeAName, "requirements/1/relationship"): []byte(relationshipTypeName),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/0/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/1/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/2/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeA/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/0/attributes/ip_address"): []byte("10.10.10.10"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/1/attributes/ip_address"): []byte("10.10.10.11"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.10"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances/ComputeB/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.11"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeAName, "0/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeAName, "1/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeAName, "2/status"): []byte("initial"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeBName, "0/status"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeBName, "1/status"): []byte("initial"),
	})
	for i, operation := range operationTestCases {
		t.Run("testExecution_ResolveInputOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecution_ResolveInputOnRelationshipTarget(t, kv, deploymentId, nodeAName, nodeBName, operation, relationshipTypeName)
		})

		t.Run("testExecution_GenerateOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecution_GenerateOnRelationshipTarget(t, kv, deploymentId, nodeAName, operation)
		})
	}
}
func testExecution_ResolveInputOnRelationshipTarget(t *testing.T, kv *api.KV, deploymentId, nodeAName, nodeBName, operation, relationshipTypeName string) {
	execution := &executionCommon{kv: kv,
		DeploymentId:             deploymentId,
		NodeName:                 nodeAName,
		Operation:                operation,
		OperationPath:            path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/types", relationshipTypeName, "interfaces/Configure/add_source"),
		isRelationshipOperation:  true,
		isRelationshipTargetNode: true,
		isPerInstanceOperation:   false,
		requirementIndex:         "1",
		relationshipType:         relationshipTypeName,
		relationshipTargetName:   nodeBName,
		VarInputsNames:           make([]string, 0),
		EnvInputs:                make([]*EnvInput, 0)}

	err := execution.resolveInputs()
	require.Nil(t, err)
	require.Len(t, execution.EnvInputs, 5)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2":

			switch envInput.InstanceName {
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 5)
}

func testExecution_GenerateOnRelationshipTarget(t *testing.T, kv *api.KV, deploymentId, nodeName, operation string) {

	execution, err := newExecution(kv, deploymentId, nodeName, operation)
	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/add_source.sh"
      environment:
        NodeA_0_A1: /var/www
        NodeA_1_A1: /var/www
        NodeA_2_A1: /var/www
        NodeB_0_A2: 10.10.10.10
        NodeB_1_A2: 10.10.10.11
        SOURCE_HOST: ComputeA
        SOURCE_INSTANCES: NodeA_0,NodeA_1,NodeA_2
        SOURCE_NODE: NodeA
        TARGET_HOST: ComputeB
        TARGET_INSTANCES: NodeB_0,NodeB_1
        TARGET_NODE: NodeB
        A1: "{{A1}}"
        A2: "{{A2}}"
        SOURCE_INSTANCE: "{{SOURCE_INSTANCE}}"
        TARGET_INSTANCE: "{{TARGET_INSTANCE}}"


`
	require.Nil(t, err)

	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemotePath = "tmp"
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}
	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err = tmpl.Parse(shell_ansible_playbook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}
