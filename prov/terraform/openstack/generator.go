package openstack

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"os"
	"path"
	"path/filepath"
)

type Generator struct {
	kv *api.KV
}

func NewGenerator(kv *api.KV) *Generator {
	return &Generator{kv: kv}
}

func (g *Generator) getStringFormConsul(baseUrl, property string) (string, error) {
	getResult, _, err := g.kv.Get(baseUrl + "/" + property, nil)
	if err != nil {
		log.Printf("Can't get property %s for node %s", property, baseUrl)
		return "", fmt.Errorf("Can't get property %s for node %s: %v", property, baseUrl, err)
	}
	if getResult == nil {
		log.Debugf("Can't get property %s for node %s (not found)", property, baseUrl)
		return "", nil
	}
	return string(getResult.Value), nil
}

func addResource(infrastructure *commons.Infrastructure, resourceType, resourceName string, resource interface{}) {
	if len(infrastructure.Resource) != 0 {
		if infrastructure.Resource[resourceType] != nil && len(infrastructure.Resource[resourceType].(map[string]interface{})) != 0 {
			resourcesMap := infrastructure.Resource[resourceType].(map[string]interface{})
			resourcesMap[resourceName] = resource
		} else {
			resourcesMap := make(map[string]interface{})
			resourcesMap[resourceName] = resource
			infrastructure.Resource[resourceType] = resourcesMap
		}

	} else {
		resourcesMap := make(map[string]interface{})
		infrastructure.Resource = resourcesMap
		resourcesMap = make(map[string]interface{})
		resourcesMap[resourceName] = resource
		infrastructure.Resource[resourceType] = resourcesMap
	}
}

func (g *Generator) GenerateTerraformInfraForNode(depId, nodeName string) error {
	log.Debugf("Generating infrastructure for deployment with id %s", depId)
	nodeKey := path.Join(deployments.DeploymentKVPrefix, depId, "topology", "nodes", nodeName)
	infrastructure := commons.Infrastructure{}
	log.Debugf("inspecting node %s", nodeKey)
	kvPair, _, err := g.kv.Get(nodeKey + "/type", nil)
	if err != nil {
		log.Print(err)
		return err
	}
	nodeType := string(kvPair.Value)
	switch nodeType {
	case "janus.nodes.openstack.Compute":
		compute, err := g.generateOSInstance(nodeKey, depId)
		if err != nil {
			return err
		}
		addResource(&infrastructure, "openstack_compute_instance_v2", compute.Name, &compute)

		consulKey := commons.ConsulKey{Name: compute.Name + "-ip_address-key", Path: nodeKey + "/capabilities/endpoint/attributes/ip_address", Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.access_ip_v4}", compute.Name)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", compute.Name, &consulKeys)
	case "janus.nodes.openstack.BlockStorage":
		bsvol, err := g.generateOSBSVolume(nodeKey)
		if err != nil {
			return err
		}

		addResource(&infrastructure, "openstack_blockstorage_volume_v1", nodeName, &bsvol)
		consulKey := commons.ConsulKey{Name: nodeName + "-bsVolumeID", Path: nodeKey + "/properties/volume_id", Value: fmt.Sprintf("${openstack_blockstorage_volume_v1.%s.id}", nodeName)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", nodeName, &consulKeys)
	default:
		return fmt.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, depId)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	infraPath := filepath.Join("work", "deployments", fmt.Sprint(depId), "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		log.Printf("%+v", err)
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		log.Print("Failed to write file")
		return err
	}

	log.Printf("Infrastructure generated for deployment with id %s", depId)
	return nil
}
