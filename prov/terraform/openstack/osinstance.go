package openstack

import (
	"fmt"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"path"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"novaforge.bull.com/starlings-janus/janus/commands"
	"gopkg.in/yaml.v2"
	"time"
)


func (g *Generator) generateOSInstance(url, deploymentId string) (ComputeInstance, error) {
	var nodeType string
	var err error

	PREFIX := viper.GetString("TF_VAR_prefix")

	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.openstack.Compute" {
		return ComputeInstance{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	if nodeName, err := g.getStringFormConsul(url, "name"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.Name = PREFIX+nodeName
	}
	if image, err := g.getStringFormConsul(url, "properties/image"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.ImageId = PREFIX+image
	}
	if image, err := g.getStringFormConsul(url, "properties/imageName"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.ImageName = PREFIX+image
	}
	if flavor, err := g.getStringFormConsul(url, "properties/flavor"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.FlavorId = PREFIX+flavor
	}
	if flavor, err := g.getStringFormConsul(url, "properties/flavorName"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.FlavorName = PREFIX+flavor
	}

	if az, err := g.getStringFormConsul(url, "properties/availability_zone"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.AvailabilityZone = PREFIX+az
	}
	if region, err := g.getStringFormConsul(url, "properties/region"); err != nil {
		return ComputeInstance{}, err
	} else if region != "" {
		instance.Region = PREFIX+region
	} else {
		//TODO make this configurable
		instance.Region = PREFIX+"RegionOne"
	}

	if keyPair, err := g.getStringFormConsul(url, "properties/key_pair"); err != nil {
		return ComputeInstance{}, err
	} else {
		// TODO if empty use a default one or fail ?
		instance.KeyPair = PREFIX+keyPair
	}

	if instance.ImageId == "" && instance.ImageName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'image' or 'imageName' node type for %s", url)
	}
	if instance.FlavorId == "" && instance.FlavorName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'flavor' or 'flavorName' node type for %s", url)
	}

	if networkName, err := g.getStringFormConsul(url, "capabilities/endpoint/properties/network_name"); err != nil {
		return ComputeInstance{}, err
	} else {
		if networkName != "" {
			// TODO Deal with networks aliases (PUBLIC/PRIVATE)
			var networkSlice []ComputeNetwork
			networkSlice = append(networkSlice, ComputeNetwork{Name: networkName})
			instance.Networks = PREFIX+networkSlice
		}
	}

	var user string
	if user, err = g.getStringFormConsul(url, "properties/user"); err != nil {
		return ComputeInstance{}, err
	} else if user == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'user' node type for %s", url)
	}

	storagePrefix := path.Join(url, "requirements", "local_storage")
	if volumeNodeName, err := g.getStringFormConsul(storagePrefix, "node"); err != nil {
		return ComputeInstance{}, err
	} else if volumeNodeName != "" {
		log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)
		var device string
		if device, err = g.getStringFormConsul(storagePrefix, "properties/location"); err != nil {
			return ComputeInstance{}, err
		}
		if device != "" {
			resolver := deployments.NewResolver(g.kv, deploymentId, url, nodeType)
			expr := tosca.ValueAssignment{}
			if err := yaml.Unmarshal([]byte(device), &expr); err != nil {
				return ComputeInstance{}, err
			}
			if device, err = resolver.ResolveExpression(expr.Expression); err != nil {
				return ComputeInstance{}, err
			}
		}
		var volumeId string
		resultChan := make(chan string, 1)
		go func() {
			for {
				log.Debugf("Looking for volume_id")
				// ignore errors and retry
				if kp, _, _ := g.kv.Get(path.Join(deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", volumeNodeName, "properties/volume_id"), nil); kp != nil {
					if dId := string(kp.Value); dId != "" {
						resultChan <- dId
						return
					}
				}
				time.Sleep(1 * time.Second)
			}
		}()
		// TODO add a cancellation signal
		select {
		case volumeId = <- resultChan:
		}

		vol := Volume{VolumeId: volumeId, Device:device}
		instance.Volumes = []Volume{vol}
	}

	// Do this in order to be sure that ansible will be able to log on the instance
	// TODO private key should not be hard-coded
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: commons.Connection{User: user, PrivateKey: `${file("~/.ssh/janus.pem")}`}}
	instance.Provisioners = make(map[string]interface{})
	instance.Provisioners["remote-exec"] = re
	return instance, nil
}
