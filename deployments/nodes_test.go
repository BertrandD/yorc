package deployments

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func TestDeploymentNodes(t *testing.T) {
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	srv1.PopulateKV(map[string][]byte{
		// Test testIsNodeTypeDerivedFrom
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/janus.type.1/derived_from":                 []byte("janus.type.2"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/janus.type.1/name":                         []byte("janus.type.1"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/janus.type.2/derived_from":                 []byte("janus.type.3"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/janus.type.2/name":                         []byte("janus.type.2"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/janus.type.3/derived_from":                 []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/janus.type.3/name":                         []byte("janus.type.3"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.HostedOn/name":         []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.HostedOn/derived_from": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.Root/name":             []byte("tosca.relationships.Root"),

		// Test testGetNbInstancesForNode
		// Default case type "tosca.nodes.Compute" default_instance specified
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/type":                                               []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/attributes/id":                                      []byte("Not Used as it exists in instances"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/capabilities/scalable/properties/default_instances": []byte("10"),
		// Case type "tosca.nodes.Compute" default_instance not specified (1 assumed)
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute2/type": []byte("tosca.nodes.Compute"),
		// Error case default_instance specified but not an uint
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute3/type":                                               []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute3/capabilities/scalable/properties/default_instances": []byte("-10"),
		// Case Node Hosted on another node

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/janus.type.1/derived_from":                 []byte("janus.type.2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/janus.type.1/name":                         []byte("janus.type.1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/janus.type.2/derived_from":                 []byte("janus.type.3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/janus.type.2/name":                         []byte("janus.type.2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/janus.type.3/derived_from":                 []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/janus.type.3/name":                         []byte("janus.type.3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.HostedOn/name":         []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.HostedOn/derived_from": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.Root/name":             []byte("tosca.relationships.Root"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Root/name":                                           []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/properties/parenttypeprop/default": []byte("RootComponentTypeProp"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/name":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/derived_from":                []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/properties/typeprop/default": []byte("SoftwareComponentTypeProp"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/attributes/id/default":       []byte("DefaultSoftwareComponentTypeid"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/attributes/type/default":     []byte("DefaultSoftwareComponentTypeid"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/0/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/0/name":         []byte("req1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/1/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/1/name":         []byte("req2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/2/relationship": []byte("janus.type.1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/2/node":         []byte("Node2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/2/name":         []byte("req3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/3/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/3/name":         []byte("req4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/properties/simple":           []byte("simple"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/attributes/id":               []byte("Node1-id"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/0/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/0/name":         []byte("req1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/1/relationship": []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/1/node":         []byte("Compute1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/1/name":         []byte("req2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/properties/recurse":          []byte("Node2"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/requirements/0/relationship": []byte("janus.type.3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/requirements/0/node":         []byte("Compute2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/requirements/0/name":         []byte("req1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/attributes/simple":           []byte("simple"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/requirements/0/relationship": []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/requirements/0/node":         []byte("Node2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/requirements/0/name":         []byte("host"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/0/attributes/id": []byte("Compute1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/1/attributes/id": []byte("Compute1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/2/attributes/id": []byte("Compute1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/3/attributes/id": []byte("Compute1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/4/attributes/id": []byte("Compute1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/5/attributes/id": []byte("Compute1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/6/attributes/id": []byte("Compute1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/7/attributes/id": []byte("Compute1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/8/attributes/id": []byte("Compute1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/9/attributes/id": []byte("Compute1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/0/attributes/recurse": []byte("Recurse-Compute1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/1/attributes/recurse": []byte("Recurse-Compute1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/2/attributes/recurse": []byte("Recurse-Compute1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/3/attributes/recurse": []byte("Recurse-Compute1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/4/attributes/recurse": []byte("Recurse-Compute1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/5/attributes/recurse": []byte("Recurse-Compute1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/6/attributes/recurse": []byte("Recurse-Compute1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/7/attributes/recurse": []byte("Recurse-Compute1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/8/attributes/recurse": []byte("Recurse-Compute1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/9/attributes/recurse": []byte("Recurse-Compute1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/0/attributes/id": []byte("Node2-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/1/attributes/id": []byte("Node2-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/2/attributes/id": []byte("Node2-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/3/attributes/id": []byte("Node2-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/4/attributes/id": []byte("Node2-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/5/attributes/id": []byte("Node2-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/6/attributes/id": []byte("Node2-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/7/attributes/id": []byte("Node2-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/8/attributes/id": []byte("Node2-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/9/attributes/id": []byte("Node2-9"),
	})

	t.Run("deployment/nodes", func(t *testing.T) {
		t.Run("IsNodeTypeDerivedFrom", func(t *testing.T) {
			testIsNodeTypeDerivedFrom(t, kv)
		})
		t.Run("GetNbInstancesForNode", func(t *testing.T) {
			testGetNbInstancesForNode(t, kv)
		})
		t.Run("GetNodeProperty", func(t *testing.T) {
			testGetNodeProperty(t, kv)
		})
		t.Run("GetNodeAttributes", func(t *testing.T) {
			testGetNodeAttributes(t, kv)
		})
	})
}

func testIsNodeTypeDerivedFrom(t *testing.T, kv *api.KV) {
	t.Parallel()

	ok, err := IsNodeTypeDerivedFrom(kv, "testIsNodeTypeDerivedFrom", "janus.type.1", "tosca.relationships.HostedOn")
	require.Nil(t, err)
	require.True(t, ok)

	ok, err = IsNodeTypeDerivedFrom(kv, "testIsNodeTypeDerivedFrom", "janus.type.1", "tosca.relationships.ConnectsTo")
	require.Nil(t, err)
	require.False(t, ok)

	ok, err = IsNodeTypeDerivedFrom(kv, "testIsNodeTypeDerivedFrom", "janus.type.1", "janus.type.1")
	require.Nil(t, err)
	require.True(t, ok)
}

func testGetNbInstancesForNode(t *testing.T, kv *api.KV) {
	t.Parallel()

	res, nb, err := GetNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, uint32(10), nb)

	res, nb, err = GetNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute2")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, uint32(1), nb)

	res, nb, err = GetNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute3")
	require.NotNil(t, err)

	res, nb, err = GetNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, uint32(10), nb)

	res, nb, err = GetNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, uint32(10), nb)

	res, nb, err = GetNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, uint32(1), nb)
}

func testGetNodeProperty(t *testing.T, kv *api.KV) {
	t.Parallel()

	// Property is directly in node
	res, value, err := GetNodeProperty(kv, "testGetNbInstancesForNode", "Node1", "simple")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, "simple", value)

	// Property is in a parent node we found it with recurse
	res, value, err = GetNodeProperty(kv, "testGetNbInstancesForNode", "Node4", "recurse")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, "Node2", value)

	// Property has a default in node type
	res, value, err = GetNodeProperty(kv, "testGetNbInstancesForNode", "Node4", "typeprop")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, "SoftwareComponentTypeProp", value)

	res, value, err = GetNodeProperty(kv, "testGetNbInstancesForNode", "Node4", "typeprop")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, "SoftwareComponentTypeProp", value)

	// Property has a default in a parent of the node type
	res, value, err = GetNodeProperty(kv, "testGetNbInstancesForNode", "Node4", "parenttypeprop")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, "RootComponentTypeProp", value)

	res, value, err = GetNodeProperty(kv, "testGetNbInstancesForNode", "Node4", "parenttypeprop")
	require.Nil(t, err)
	require.True(t, res)
	require.Equal(t, "RootComponentTypeProp", value)

}

func testGetNodeAttributes(t *testing.T, kv *api.KV) {
	t.Parallel()
	// Attribute is directly in node
	res, instancesValues, err := GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node3", "simple")
	require.Nil(t, err)
	require.True(t, res)
	require.Len(t, instancesValues, 1)
	require.Equal(t, "simple", instancesValues[""])

	// Attribute is directly in instances
	res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Compute1", "id")
	require.Nil(t, err)
	require.True(t, res)
	require.Len(t, instancesValues, 10)
	require.Equal(t, "Compute1-0", instancesValues["0"])
	require.Equal(t, "Compute1-1", instancesValues["1"])
	require.Equal(t, "Compute1-2", instancesValues["2"])
	require.Equal(t, "Compute1-3", instancesValues["3"])

	// Look at generic node attribute before parents
	res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node1", "id")
	require.Nil(t, err)
	require.True(t, res)
	require.Len(t, instancesValues, 1)
	require.Equal(t, "Node1-id", instancesValues[""])

	// Look at generic node type attribute before parents
	res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node3", "id")
	require.Nil(t, err)
	require.True(t, res)
	require.Len(t, instancesValues, 1)
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues[""])

	// Look at generic node type attribute before parents
	res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node2", "type")
	require.Nil(t, err)
	require.True(t, res)
	require.Len(t, instancesValues, 10)
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["0"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["3"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["6"])

	//
	res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node2", "recurse")
	require.Nil(t, err)
	require.True(t, res)
	require.Len(t, instancesValues, 10)
	require.Equal(t, "Recurse-Compute1-0", instancesValues["0"])
	require.Equal(t, "Recurse-Compute1-3", instancesValues["3"])
	require.Equal(t, "Recurse-Compute1-6", instancesValues["6"])

	//
	res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node1", "recurse")
	require.Nil(t, err)
	require.True(t, res)
	require.Len(t, instancesValues, 10)
	require.Equal(t, "Recurse-Compute1-0", instancesValues["0"])
	require.Equal(t, "Recurse-Compute1-3", instancesValues["3"])
	require.Equal(t, "Recurse-Compute1-6", instancesValues["6"])
}
