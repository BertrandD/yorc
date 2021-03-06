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

package commons

// An Infrastructure is the top-level element of a Terraform infrastructure definition
type Infrastructure struct {
	Terraform map[string]interface{} `json:"terraform,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Variable  map[string]interface{} `json:"variable,omitempty"`
	Provider  map[string]interface{} `json:"provider,omitempty"`
	Resource  map[string]interface{} `json:"resource,omitempty"`
	Output    map[string]*Output     `json:"output,omitempty"`
}

// The ConsulKeys can be used as 'resource' to writes or 'data' to read sets of individual values into Consul.
type ConsulKeys struct {
	Resource
	Datacenter string      `json:"datacenter,omitempty"`
	Token      string      `json:"token,omitempty"`
	Keys       []ConsulKey `json:"key"`
}

// A Resource is the base type for terraform resources
type Resource struct {
	Count        int                      `json:"count,omitempty"`
	DependsOn    []string                 `json:"depends_on,omitempty"`
	Connection   *Connection              `json:"connection,omitempty"`
	Provisioners []map[string]interface{} `json:"provisioner,omitempty"`
}

// A ConsulKey can be used in a ConsulKeys 'resource' to writes or a ConsulKeys 'data' to read an individual Key/Value pair into Consul
type ConsulKey struct {
	Path string `json:"path"`

	// Should only be use in datasource (read) mode, this is the name to use to access this key within the terraform interpolation syntax
	Name string `json:"name,omitempty"`
	// Should only be use in datasource (read) mode, default value if the key is not found

	Default string `json:"default,omitempty"`
	// Should only be use in resource (write) mode, the value to set to the key

	Value string `json:"value,omitempty"`

	// Should only be use in resource (write) mode, deletes the key
	Delete bool `json:"delete,omitempty"`
}

// The RemoteExec provisioner invokes a script on a remote resource after it is created.
//
// The remote-exec provisioner supports both ssh and winrm type connections.
type RemoteExec struct {
	Connection *Connection `json:"connection,omitempty"`
	Inline     []string    `json:"inline,omitempty"`
	Script     string      `json:"script,omitempty"`
	Scripts    []string    `json:"scripts,omitempty"`
}

// A Connection allows to overwrite the way Terraform connects to a resource
type Connection struct {
	ConnType   string `json:"type,omitempty"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       string `json:"port,omitempty"`
	Timeout    string `json:"timeout,omitempty"` // defaults to "5m"
	PrivateKey string `json:"private_key,omitempty"`
}

// An Output allows to define a terraform output value
type Output struct {
	// Value is the value of the output. This can be a string, list, or map.
	// This usually includes an interpolation since outputs that are static aren't usually useful.
	Value     interface{} `json:"value"`
	Sensitive bool        `json:"sensitive,omitempty"`
}

// AddResource allows to add a Resource to a defined Infrastructure
func AddResource(infrastructure *Infrastructure, resourceType, resourceName string, resource interface{}) {
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

// AddOutput allows to add an Output to a defined Infrastructure
func AddOutput(infrastructure *Infrastructure, outputName string, output *Output) {
	if infrastructure.Output == nil {
		infrastructure.Output = make(map[string]*Output)
	}
	infrastructure.Output[outputName] = output
}
