// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
)

// GetInputValue tries to retrieve the value of the given input name.
//
// GetInputValue first checks if a non-empty field value exists for this input, if it doesn't then it checks for a non-empty field default.
// If none of them exists then it returns an empty string.
func GetInputValue(kv *api.KV, deploymentID, inputName string, nestedKeys ...string) (string, error) {
	dataType, err := GetTopologyInputType(kv, deploymentID, inputName)

	found, result, err := getValueAssignmentWithDataType(kv, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/inputs", inputName, "value"), "", "", "", dataType, nestedKeys...)
	if err != nil || found {
		return result, errors.Wrapf(err, "Failed to get input %q value", inputName)
	}
	_, result, err = getValueAssignmentWithDataType(kv, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/inputs", inputName, "default"), "", "", "", dataType, nestedKeys...)

	return result, errors.Wrapf(err, "Failed to get input %q value", inputName)
}
