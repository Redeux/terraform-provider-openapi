package openapi

import (
	"bytes"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestPluginConfigSchemaV1(t *testing.T) {
	Convey("Given a list of services and version", t, func() {
		version := ""
		services := map[string]*ServiceConfigV1{}
		Convey("When PluginConfigSchemaV1 method is constructed", func() {
			pluginConfigSchemaV1 := &PluginConfigSchemaV1{
				Services: services,
				Version:  version,
			}
			Convey("Then the pluginConfigSchemaV1 should comply with PluginConfigSchema interface", func() {
				var _ PluginConfigSchema = pluginConfigSchemaV1
			})
		})
	})
}

func TestNewPluginConfigSchemaV1(t *testing.T) {
	Convey("Given a schema version and a map of services and their swagger URLs", t, func() {
		services := map[string]*ServiceConfigV1{
			"test": {
				SwaggerURL:         "http://sevice-api.com/swagger.yaml",
				InsecureSkipVerify: true,
			},
		}
		telemetryConfig := &TelemetryConfig{
			Graphite: &TelemetryProviderGraphite{
				Host:   "some-host.com",
				Port:   8125,
				Prefix: "some_prefix",
			},
		}
		Convey("When NewPluginConfigSchemaV1 method is called", func() {
			pluginConfigSchemaV1 := NewPluginConfigSchemaV1(services, telemetryConfig)
			Convey("And the pluginConfigSchema returned should implement PluginConfigSchema interface", func() {
				var _ PluginConfigSchema = pluginConfigSchemaV1
			})
			Convey("And the pluginConfigSchema services", func() {
				So(pluginConfigSchemaV1.Services, ShouldNotBeNil)
			})
			Convey("And the pluginConfigSchema telemetry should not be nil", func() {
				So(pluginConfigSchemaV1.TelemetryConfig, ShouldNotBeNil)
			})
		})
	})
}

func TestPluginConfigSchemaV1Validate(t *testing.T) {
	Convey("Given a PluginConfigSchemaV1 containing a version supported and some services", t, func() {
		var pluginConfigSchema PluginConfigSchema
		services := map[string]*ServiceConfigV1{
			"test": {
				SwaggerURL:         "http://sevice-api.com/swagger.yaml",
				InsecureSkipVerify: true,
			},
		}
		pluginConfigSchema = NewPluginConfigSchemaV1(services, nil)
		Convey("When Validate method is called", func() {
			err := pluginConfigSchema.Validate()
			Convey("Then the error returned should be nil as configuration is correct", func() {
				So(err, ShouldBeNil)
			})
		})
	})
	Convey("Given a PluginConfigSchemaV1 containing a version that is NOT supported and some services", t, func() {
		var pluginConfigSchema PluginConfigSchema
		version := "2"
		services := map[string]*ServiceConfigV1{
			"test": {
				SwaggerURL:         "http://sevice-api.com/swagger.yaml",
				InsecureSkipVerify: true,
			},
		}
		pluginConfigSchema = &PluginConfigSchemaV1{
			Version:  version,
			Services: services,
		}
		Convey("When Validate method is called", func() {
			err := pluginConfigSchema.Validate()
			Convey("Then the error returned should NOT be nil as version is not supported", func() {
				So(err, ShouldNotBeNil)
			})
			Convey("And the error returned be equal to", func() {
				So(err.Error(), ShouldEqual, "provider configuration version not matching current implementation, please use version '1' of provider configuration specification")
			})
		})
	})
}

func TestPluginConfigSchemaV1GetServiceConfig(t *testing.T) {
	Convey("Given a PluginConfigSchemaV1 containing a version supported and some services", t, func() {
		var pluginConfigSchema PluginConfigSchema
		expectedURL := "http://sevice-api.com/swagger.yaml"
		services := map[string]*ServiceConfigV1{
			"test": {
				SwaggerURL:         expectedURL,
				InsecureSkipVerify: true,
			},
		}
		pluginConfigSchema = NewPluginConfigSchemaV1(services, nil)
		Convey("When GetServiceConfig method is called with a service described in the configuration", func() {
			serviceConfig, err := pluginConfigSchema.GetServiceConfig("test")
			Convey("Then the error returned should be nil as configuration is correct", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the serviceConfig should not be nil", func() {
				So(serviceConfig, ShouldNotBeNil)
			})
			Convey("And the url returned should be equal to the one in the service configuration", func() {
				So(serviceConfig.GetSwaggerURL(), ShouldEqual, expectedURL)
			})
		})
		Convey("When GetServiceConfig method is called with a service that DOES NOT exist in the plugin configuration", func() {
			_, err := pluginConfigSchema.GetServiceConfig("non-existing-service")
			Convey("Then the error returned should not be nil as provider specified does not exist in configuration file", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestPluginConfigSchemaV1GetVersion(t *testing.T) {
	Convey("Given a PluginConfigSchemaV1 containing a version supported and some services", t, func() {
		var pluginConfigSchema PluginConfigSchema
		expectedURL := "http://sevice-api.com/swagger.yaml"
		services := map[string]*ServiceConfigV1{
			"test": {
				SwaggerURL:         expectedURL,
				InsecureSkipVerify: true,
			},
		}
		pluginConfigSchema = NewPluginConfigSchemaV1(services, nil)
		Convey("When GetVersion method is called", func() {
			configVersion, err := pluginConfigSchema.GetVersion()
			Convey("Then the error returned should be nil as configuration is correct", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the serviceConfig should not be nil", func() {
				So(configVersion, ShouldEqual, "1")
			})
		})
	})
}

func TestPluginConfigSchemaV1GetAllServiceConfigurations(t *testing.T) {
	Convey("Given a PluginConfigSchemaV1 containing a version supported and some services", t, func() {
		var pluginConfigSchema PluginConfigSchema
		expectedURL := "http://sevice-api.com/swagger.yaml"
		serviceConfigName := "test"
		services := map[string]*ServiceConfigV1{
			serviceConfigName: {
				SwaggerURL:         expectedURL,
				InsecureSkipVerify: true,
			},
		}
		pluginConfigSchema = NewPluginConfigSchemaV1(services, nil)
		Convey("When GetAllServiceConfigurations method is called", func() {
			serviceConfigurations, err := pluginConfigSchema.GetAllServiceConfigurations()
			Convey("Then the error returned should be nil as configuration is correct", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the serviceConfigurations contain 1 configuration", func() {
				So(len(serviceConfigurations), ShouldEqual, 1)
			})
			Convey("And the serviceConfigurations item should be test", func() {
				So(serviceConfigurations[serviceConfigName], ShouldNotBeNil)
			})
		})
	})
}

func TestPluginConfigSchemaV1Marshal(t *testing.T) {
	Convey("Given a PluginConfigSchemaV1 containing a version supported and some services", t, func() {
		var pluginConfigSchema PluginConfigSchema
		expectedURL := "http://sevice-api.com/swagger.yaml"
		expectedPluginVersion := "0.14.0"
		serviceConfigName := "test"
		expectedInscureSkipVerify := true
		services := map[string]*ServiceConfigV1{
			serviceConfigName: {
				SwaggerURL:         expectedURL,
				PluginVersion:      expectedPluginVersion,
				InsecureSkipVerify: expectedInscureSkipVerify,
				SchemaConfigurationV1: []ServiceSchemaPropertyConfigurationV1{
					{
						SchemaPropertyName: "apikey_auth",
						DefaultValue:       "apiKeyValue",
						Command:            []string{"echo", "something"},
						CommandTimeout:     10,
						ExternalConfiguration: ServiceSchemaPropertyExternalConfigurationV1{
							File:        "some_file",
							KeyName:     "some_key_name",
							ContentType: "json",
						},
					},
				},
			},
		}
		pluginConfigSchema = NewPluginConfigSchemaV1(services, nil)
		Convey("When Marshal method is called", func() {
			marshalConfig, err := pluginConfigSchema.Marshal()
			Convey("Then the error returned should be nil as configuration is correct", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the marshalConfig should contain the right marshal configuration", func() {
				expectedConfig := fmt.Sprintf(`version: "1"
services:
  test:
    swagger-url: %s
    plugin_version: %s
    insecure_skip_verify: %t
    schema_configuration:
    - schema_property_name: apikey_auth
      default_value: apiKeyValue
      cmd: [echo, something]
      cmd_timeout: 10
      schema_property_external_configuration:
        file: some_file
        key_name: some_key_name
        content_type: json
`, expectedURL, expectedPluginVersion, expectedInscureSkipVerify)
				So(string(marshalConfig), ShouldEqual, expectedConfig)
			})
		})
	})

	Convey("Given a PluginConfigSchemaV1 containing a version supported and a service that does not specify a fix plugin version", t, func() {
		var pluginConfigSchema PluginConfigSchema
		expectedURL := "http://sevice-api.com/swagger.yaml"
		serviceConfigName := "test"
		expectedInscureSkipVerify := true
		services := map[string]*ServiceConfigV1{
			serviceConfigName: {
				SwaggerURL: expectedURL,
				//PluginVersion: expectedPluginVersion,
				InsecureSkipVerify: expectedInscureSkipVerify,
				SchemaConfigurationV1: []ServiceSchemaPropertyConfigurationV1{
					{
						SchemaPropertyName: "apikey_auth",
						DefaultValue:       "apiKeyValue",
						Command:            []string{"echo", "something"},
						CommandTimeout:     10,
						ExternalConfiguration: ServiceSchemaPropertyExternalConfigurationV1{
							File:        "some_file",
							KeyName:     "some_key_name",
							ContentType: "json",
						},
					},
				},
			},
		}
		pluginConfigSchema = NewPluginConfigSchemaV1(services, nil)
		Convey("When Marshal method is called", func() {
			marshalConfig, err := pluginConfigSchema.Marshal()
			Convey("Then the error returned should be nil as configuration is correct", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the marshalConfig should contain the right marshal configuration (and the plugin_version property should not be present)", func() {
				expectedConfig := fmt.Sprintf(`version: "1"
services:
  test:
    swagger-url: %s
    insecure_skip_verify: %t
    schema_configuration:
    - schema_property_name: apikey_auth
      default_value: apiKeyValue
      cmd: [echo, something]
      cmd_timeout: 10
      schema_property_external_configuration:
        file: some_file
        key_name: some_key_name
        content_type: json
`, expectedURL, expectedInscureSkipVerify)
				So(string(marshalConfig), ShouldEqual, expectedConfig)
			})
		})
	})
}

func TestGetTelemetryHandler(t *testing.T) {
	testCases := []struct {
		name                 string
		pluginConfigSchemaV1 PluginConfigSchemaV1
		inputPluginName      string
		expectedError        string
		expectedLogging      string
	}{
		{
			name: "handler is configured correctly",
			pluginConfigSchemaV1: PluginConfigSchemaV1{
				TelemetryConfig: &TelemetryConfig{
					Graphite: &TelemetryProviderGraphite{
						Host: "my-graphite.com",
						Port: 8125,
					},
				},
			},
			inputPluginName: "pluginName",
			expectedLogging: "[DEBUG] graphite telemetry provider enabled",
		},
		{
			name: "handler skips graphite telemetry due to the validation not passing",
			pluginConfigSchemaV1: PluginConfigSchemaV1{
				TelemetryConfig: &TelemetryConfig{
					Graphite: &TelemetryProviderGraphite{
						Host: "", // Configuration is missing the required host
						//Port: 8125,
					},
				},
			},
			inputPluginName: "pluginName",
			expectedLogging: "[WARN] ignoring graphite telemetry due to the following validation error: graphite telemetry configuration is missing a value for the 'host property'",
		},
	}
	for _, tc := range testCases {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		telemetryHandler := tc.pluginConfigSchemaV1.GetTelemetryHandler(tc.inputPluginName)
		assert.IsType(t, telemetryHandlerTimeoutSupport{}, telemetryHandler, tc.name)
		assert.Contains(t, buf.String(), tc.expectedLogging, tc.name)
	}
}
