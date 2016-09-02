package config

const DEFAULT_REST_CONSUL_PUB_MAX_ROUTINES int = 1000

type Configuration struct {
	OS_AUTH_URL                  string `json:"os_auth_url,omitempty"`
	OS_TENANT_ID                 string `json:"os_tenant_id,omitempty"`
	OS_TENANT_NAME               string `json:"os_tenant_name,omitempty"`
	OS_USER_NAME                 string `json:"os_user_name,omitempty"`
	OS_PASSWORD                  string `json:"os_password,omitempty"`
	OS_REGION                    string `json:"os_region,omitempty"`
	OS_PREFIX                    string `json:"os_prefix,omitempty"`
	CONSUL_TOKEN                 string `json:"consul_token,omitempty"`
	CONSUL_DATACENTER            string `json:"consul_datacenter,omitempty"`
	CONSUL_ADDRESS               string `json:"consul_address,omitempty"`
	REST_CONSUL_PUB_MAX_ROUTINES int    `json:"rest_consul_publisher_max_routines,omitempty"`
}
