package deployments

import (
	"path"

	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
)

// GetArtifactsForType returns a map of artifact name / artifact file for the given type.
//
// It traverse the 'derived_from' relations to support inheritance of artifacts. Parent artifacts are fetched first and may be overridden by child types
func GetArtifactsForType(kv *api.KV, deploymentID, typeName string) (map[string]string, error) {
	parentType, err := GetParentType(kv, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	var artifacts map[string]string
	if parentType != "" {
		artifacts, err = GetArtifactsForType(kv, deploymentID, parentType)
		if err != nil {
			return nil, err
		}
	} else {
		artifacts = make(map[string]string)
	}
	artifactsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", typeName, "artifacts")

	err = updateArtifactsFromPath(kv, artifacts, artifactsPath)
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for type: %q", typeName)
}

// GetArtifactsForNode returns a map of artifact name / artifact file for the given node.
//
// It will first fetch artifacts from it node type and its parents and fetch artifacts for the node template itself.
// This way artifacts from a parent type may be overridden by child types and artifacts from node type may be overridden by the node template
func GetArtifactsForNode(kv *api.KV, deploymentID, nodeName string) (map[string]string, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	artifacts, err := GetArtifactsForType(kv, deploymentID, nodeType)
	if err != nil {
		return nil, err
	}
	artifactsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "artifacts")

	err = updateArtifactsFromPath(kv, artifacts, artifactsPath)
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for node: %q", nodeName)
}

// updateArtifactsFromPath returns a map of artifact name / artifact file for the given node or type denoted by the given artifactsPath.
func updateArtifactsFromPath(kv *api.KV, artifacts map[string]string, artifactsPath string) error {
	kvps, _, err := kv.Keys(artifactsPath+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, artifactPath := range kvps {
		kvp, _, err := kv.Get(path.Join(artifactPath, "name"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return errors.Errorf("Missing mandatory attribute \"name\" for artifact %q", path.Base(artifactPath))
		}
		artifactName := string(kvp.Value)
		kvp, _, err = kv.Get(path.Join(artifactPath, "file"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return errors.Errorf("Missing mandatory attribute \"file\" for artifact %q", path.Base(artifactPath))
		}
		// TODO path is relative to the type and may not be the same as a child type
		artifacts[artifactName] = string(kvp.Value)
	}
	return nil
}

// GetArtifactTypeExtensions returns the extensions defined in this artifact type.
// If the artifact doesn't define any extension then a nil slice is returned
func GetArtifactTypeExtensions(kv *api.KV, deploymentID, artifactType string) ([]string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", artifactType, "file_ext"), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return nil, nil
	}
	return strings.Split(string(kvp.Value), ","), nil
}
