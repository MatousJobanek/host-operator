// Package configuration is in charge of the validation and extraction of all
// the configuration details from a configuration file or environment variables.
package configuration

import (
	errs "github.com/pkg/errors"
	"github.com/spf13/viper"
	"strings"
)

const (
	// HostEnvPrefix will be used for host environment variable name prefixing.
	HostEnvPrefix = "HOST"

	// RegServiceEnvPrefix will be used for registration service environment variable name prefixing.
	RegServiceEnvPrefix = "REGISTRATION_SERVICE"

	// varImage specifies registration service image to be used for deployment
	varImage = "image"

	// varEnvironment specifies registration service environment such as prod, stage, unit-tests, e2e-tests, dev, etc
	varEnvironment = "environment"
	// DefaultEnvironment is the default registration service environment
	DefaultEnvironment = "prod"
)

// Registry encapsulates the Viper configuration registry which stores the
// configuration data in-memory.
type Registry struct {
	host       *viper.Viper
	regService *viper.Viper
}

// CreateEmptyRegistry creates an initial, empty registry.
func CreateEmptyRegistry() *Registry {
	c := Registry{
		host:       viper.New(),
		regService: viper.New(),
	}
	c.host.SetEnvPrefix(HostEnvPrefix)
	c.regService.SetEnvPrefix(RegServiceEnvPrefix)
	for _, v := range []*viper.Viper{c.host, c.regService} {
		v.AutomaticEnv()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.SetTypeByDefaultValue(true)
	}
	c.setConfigDefaults()
	return &c
}

// New creates a configuration reader object using a configurable configuration
// file path. If the provided config file path is empty, a default configuration
// will be created.
func New(configFilePath string) (*Registry, error) {
	c := CreateEmptyRegistry()
	if configFilePath != "" {
		c.host.SetConfigType("yaml")
		c.host.SetConfigFile(configFilePath)
		err := c.host.ReadInConfig() // Find and read the config file
		if err != nil {              // Handle errors reading the config file.
			return nil, errs.Wrap(err, "failed to read config file")
		}
	}
	return c, nil
}

// GetViperInstance returns the underlying Viper instance.
func (c *Registry) GetViperInstance() *viper.Viper {
	return c.host
}

func (c *Registry) setConfigDefaults() {
	c.host.SetTypeByDefaultValue(true)
	c.regService.SetTypeByDefaultValue(true)

	c.regService.SetDefault(varEnvironment, DefaultEnvironment)
}

// GetRegServiceImage returns the registration service image.
func (c *Registry) GetRegServiceImage() string {
	return c.regService.GetString(varImage)
}

// GetRegServiceEnvironment returns the registration service environment such as prod, stage, unit-tests, e2e-tests, dev, etc
func (c *Registry) GetRegServiceEnvironment() string {
	return c.regService.GetString(varEnvironment)
}
