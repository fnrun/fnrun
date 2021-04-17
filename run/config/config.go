// Package config provides interfaces for handling configuration data and a
// function to apply configuration data.
package config

import (
	"errors"
	"fmt"
)

// Required indicates whether an object is required to have configuration data.
type Required interface {
	RequiresConfig() bool
}

// StringConfigurer configures an object from a string.
type StringConfigurer interface {
	ConfigureString(string) error
}

// IntegerConfigurer configures an object from an int.
type IntegerConfigurer interface {
	ConfigureInteger(int) error
}

// FloatConfigurer configures an object from a float64.
type FloatConfigurer interface {
	ConfigureFloat(float64) error
}

// BoolConfigurer configures an object from a boolean.
type BoolConfigurer interface {
	ConfigureBool(bool) error
}

// MapConfigurer configures an object from a map.
type MapConfigurer interface {
	ConfigureMap(map[string]interface{}) error
}

// ArrayConfigurer configures an object from an array of objects.
type ArrayConfigurer interface {
	ConfigureArray([]interface{}) error
}

// GenericConfigurer configures an object from an interface{}.
type GenericConfigurer interface {
	ConfigureGeneric(interface{}) error
}

// EmptyConfigurer configures an object without a config value.
type EmptyConfigurer interface {
	Configure() error
}

// Configure applies the config to the target object. Configure will select the
// most appropriate *Configurer interface on the target and call the method from
// that interface. Configure will return an error if an appropriate
// implementation of a *Configurer interface is not found and the object
// requires configuration, as specified by the Required interface.
//gocyclo:ignore
func Configure(target interface{}, config interface{}) error {
	switch config.(type) {
	case nil:
		if t, ok := target.(EmptyConfigurer); ok {
			return t.Configure()
		}
		if t, ok := target.(Required); !ok || !t.RequiresConfig() {
			return nil
		}
	case string:
		if t, ok := target.(StringConfigurer); ok {
			return t.ConfigureString(config.(string))
		}
	case int:
		if t, ok := target.(IntegerConfigurer); ok {
			return t.ConfigureInteger(config.(int))
		}
	case float64:
		if t, ok := target.(FloatConfigurer); ok {
			return t.ConfigureFloat(config.(float64))
		}
	case bool:
		if t, ok := target.(BoolConfigurer); ok {
			return t.ConfigureBool(config.(bool))
		}
	case map[string]interface{}:
		if t, ok := target.(MapConfigurer); ok {
			return t.ConfigureMap(config.(map[string]interface{}))
		}
	case []interface{}:
		if t, ok := target.(ArrayConfigurer); ok {
			return t.ConfigureArray(config.([]interface{}))
		}
	default:
		if t, ok := target.(GenericConfigurer); ok {
			return t.ConfigureGeneric(config)
		}
	}

	return misconfiguredTypeError(target, config)
}

func misconfiguredTypeError(target interface{}, config interface{}) error {
	return fmt.Errorf("%T could not be configured with object of type %T", target, config)
}

// GetSinglePair gets the key/value pair of the single value in m. If there is
// not exactly one key/value pair in m, GetSinglePair returns an error.
func GetSinglePair(m map[string]interface{}) (string, interface{}, error) {
	if len(m) == 1 {
		for k, v := range m {
			return k, v, nil
		}
	}

	return "", nil, errors.New("expected map to have exactly one entry")
}
