package plugin

import (
	"os"

	"encoding/gob"
	"github.com/hashicorp/go-plugin"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

const (
	// DelegatePluginName is the name of Delegates Plugins it could be used as a lookup key in Client.Dispense
	DelegatePluginName = "delegate"
	// DefinitionsPluginName is the name of Delegates Plugins it could be used as a lookup key in Client.Dispense
	DefinitionsPluginName = "definitions"
	// ConfigManagerPluginName is the name of ConfigManager plugin it could be used as a lookup key in Client.Dispense
	ConfigManagerPluginName = "cfgManager"
	// OperationPluginName is the name of Operation Plugins it could be used as a lookup key in Client.Dispense
	OperationPluginName = "operation"
)

// HandshakeConfig are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "JANUS_PLUG_API",
	MagicCookieValue: "a3292e718f7c96578aae47e92b7475394e72e6da3de3455554462ba15dde56d1b3187ad0e5f809f50767e0d10ca6944fdf4c6c412380d3aa083b9e8951f7101e",
}

// DelegateFunc is a function that is called when creating a plugin server
type DelegateFunc func() prov.DelegateExecutor

// OperationFunc is a function that is called when creating a plugin server
type OperationFunc func() prov.OperationExecutor

// ServeOpts are the configurations to serve a plugin.
type ServeOpts struct {
	DelegateFunc                    DelegateFunc
	DelegateSupportedTypes          []string
	Definitions                     map[string][]byte
	OperationFunc                   OperationFunc
	OperationSupportedArtifactTypes []string
}

// Serve serves a plugin. This function never returns and should be the final
// function called in the main function of the plugin.
func Serve(opts *ServeOpts) {
	// As we have type []interface{} in the config.Configuration structure, we need to register it before receiving config from janus server
	// The same registration needs has been done server side
	gob.Register(make([]interface{}, 0))

	// As a plugin configure janus logs to go to stderr in order to be show in the parent process
	log.SetOutput(os.Stderr)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         getPlugins(opts),
	})
}

func getPlugins(opts *ServeOpts) map[string]plugin.Plugin {
	if opts == nil {
		opts = new(ServeOpts)
	}
	return map[string]plugin.Plugin{
		DelegatePluginName:      &DelegatePlugin{F: opts.DelegateFunc, SupportedTypes: opts.DelegateSupportedTypes},
		OperationPluginName:     &OperationPlugin{F: opts.OperationFunc, SupportedTypes: opts.OperationSupportedArtifactTypes},
		DefinitionsPluginName:   &DefinitionsPlugin{Definitions: opts.Definitions},
		ConfigManagerPluginName: &ConfigManagerPlugin{&defaultConfigManager{}},
	}
}
