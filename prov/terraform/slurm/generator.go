package slurm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
)

type Generator struct {
	kv  *api.KV
	cfg config.Configuration
}

func NewGenerator(kv *api.KV, cfg config.Configuration) *Generator {
	return &Generator{kv: kv, cfg: cfg}
}

func (g *Generator) getStringFormConsul(baseURL, property string) (string, error) {
	getResult, _, err := g.kv.Get(baseURL+"/"+property, nil)
	if err != nil {
		log.Printf("Can't get property %s for node %s", property, baseURL)
		return "", fmt.Errorf("Can't get property %s for node %s: %v", property, baseURL, err)
	}
	if getResult == nil {
		log.Debugf("Can't get property %s for node %s (not found)", property, baseURL)
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

func (g *Generator) GenerateTerraformInfraForNode(deploymentID, nodeName string) (bool, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	instancesKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	infrastructure := commons.Infrastructure{}
	log.Debugf("inspecting node %s", nodeKey)
	kvPair, _, err := g.kv.Get(nodeKey+"/type", nil)
	if err != nil {
		log.Print(err)
		return false, err
	}
	nodeType := string(kvPair.Value)
	log.Printf("GenerateTerraformInfraForNode switch begin")
	switch nodeType {
	case "janus.nodes.slurm.Compute":
		var instances []string
		instances, _, err = g.kv.Keys(instancesKey+"/", "/", nil)
		if err != nil {
			return false, err
		}

		for _, instanceName := range instances {
			instanceName = path.Base(instanceName)
			var compute ComputeInstance
			compute, err = g.generateSlurmNode(nodeKey, deploymentID)
			var computeName = nodeName + "-" + instanceName
			log.Debugf("XBD computeName : IN FOR  %s", computeName)
			log.Debugf("XBD instanceName: IN FOR  %s", instanceName)
			if err != nil {
				return false, err
			}

			addResource(&infrastructure, "slurm_node", computeName, &compute)

			consulKey := commons.ConsulKey{Name: computeName + "-node_name", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}
			consulKey2 := commons.ConsulKey{Name: computeName + "-ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}
			consulKeyAttrib := commons.ConsulKey{Name: computeName + "-attrib_ip_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}

			consulKeyJobID := commons.ConsulKey{Name: computeName + "-job_id", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/job_id"), Value: fmt.Sprintf("${slurm_node.%s.job_id}", computeName)}

			var consulKeys commons.ConsulKeys
			consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey, consulKey2, consulKeyAttrib, consulKeyJobID}}
			addResource(&infrastructure, "consul_keys", computeName, &consulKeys)

		} //End instances loop

		// Add Provider
		infrastructure.Provider = make(map[string]interface{})
		providerSlurmMap := make(map[string]interface{})
		infrastructure.Provider["slurm"] = providerSlurmMap

	default:
		return false, fmt.Errorf("In Slurm : Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}
	infraPath := filepath.Join("work", "deployments", fmt.Sprint(deploymentID), "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		log.Printf("%+v", err)
		return false, err
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		log.Print("Failed to write file")
		return false, err
	}

	log.Printf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, nil
}
