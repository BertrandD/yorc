package operations

import (
	"github.com/hashicorp/consul/api"

	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

// GetOperation returns a Prov.Operation structure describing precisely operation in order to execute it
func GetOperation(kv *api.KV, deploymentID, nodeName, operationName, requirementName, operationHost string) (prov.Operation, error) {
	var (
		opHost, implementingType, requirementIndex string
		err                                        error
		isRelationshipOp                           bool
	)
	// if requirementName is filled, operation is associated to a relationship
	isRelationshipOp = requirementName != ""
	// Default operation host is HOST
	if operationHost != "" {
		opHost = operationHost
	} else {
		opHost = "HOST"
	}
	if requirementName != "" {
		key, err := deployments.GetRequirementKeyByNameForNode(kv, deploymentID, nodeName, requirementName)
		if err != nil {
			return prov.Operation{}, err
		}
		if key == "" {
			return prov.Operation{}, errors.Errorf("Unable to found requirement key for requirement name:%q", requirementName)
		}
		requirementIndex = deployments.GetRequirementIndexFromRequirementKey(key)
	}
	if isRelationshipOp {
		implementingType, err = deployments.GetRelationshipTypeImplementingAnOperation(kv, deploymentID, nodeName, operationName, requirementIndex)
	} else {
		implementingType, err = deployments.GetNodeTypeImplementingAnOperation(kv, deploymentID, nodeName, operationName)
	}
	if err != nil {
		return prov.Operation{}, err
	}
	implArt, err := deployments.GetImplementationArtifactForOperation(kv, deploymentID, nodeName, operationName, isRelationshipOp, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}
	targetNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}

	op := prov.Operation{
		Name:                   operationName,
		ImplementedInType:      implementingType,
		ImplementationArtifact: implArt,
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: isRelationshipOp,
			RequirementIndex:        requirementIndex,
			TargetNodeName:          targetNodeName,
		},
		OperationHost: opHost,
	}
	return op, nil
}
