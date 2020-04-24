package e2e

import (
	"fmt"
	"github.com/dikhan/terraform-provider-openapi/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestAcc_ProviderConfiguration_PluginExternalFile_HTTPEndpointTelemetry confirms regressions introduced in the logic related to the plugin
// external configuration. This test confirms that the plugin is able to start up properly and functions as expected even
// when the plugin uses the external configuration containing:
// - HTTPEndpoint telemetry configuration
// - Service configurations
func TestAcc_ProviderConfiguration_PluginExternalFile_HTTPEndpointTelemetry(t *testing.T) {
	httpEndpointTelemetryCalled := false
	var headersReceived http.Header
	var metricsReceived []string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/metrics":
			httpEndpointTelemetryCalled = true
			headersReceived = r.Header
			body, err := ioutil.ReadAll(r.Body)
			assert.Nil(t, err)
			metricsReceived = append(metricsReceived, string(body))
			w.WriteHeader(http.StatusOK)
			break
		case "/v1/cdns", "/v1/cdns/someID":
			httpEndpointTelemetryCalled = true
			w.Write([]byte(`{"id":"someID", "label": "some_label"}`))
			w.WriteHeader(http.StatusOK)
			break
		}
	}))
	apiHost := apiServer.URL[7:]

	swaggerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		swaggerYAMLTemplate := fmt.Sprintf(`swagger: "2.0"
host: "%s"

schemes:
- "http"

paths:
  /v1/cdns:
    post:
      parameters:
      - in: "body"
        name: "body"
        description: "Created CDN"
        required: true
        schema:
          $ref: "#/definitions/ContentDeliveryNetworkV1"
      - type: string
        x-terraform-header: some_header
        name: some_header
        in: header
        required: true
      responses:
        201:
          description: "successful operation"
          schema:
            $ref: "#/definitions/ContentDeliveryNetworkV1"

  /v1/cdns/{cdn_id}:
    get:
      parameters:
      - name: "cdn_id"
        in: "path"
        description: "The cdn id that needs to be fetched."
        required: true
        type: "string"
      responses:
        200:
          description: "successful operation"
          schema:
            $ref: "#/definitions/ContentDeliveryNetworkV1"
    delete:
      parameters:
      - name: "id"
        in: "path"
        description: "The cdn that needs to be deleted"
        required: true
        type: "string"
      responses:
        204:
          description: "successful operation, no content is returned"
definitions:
  ContentDeliveryNetworkV1:
    type: "object"
    required:
      - label
    properties:
      id:
        type: "string"
        readOnly: true
      label:
        type: "string"
securityDefinitions:
  some_token:
    in: header
    name: Token
    type: apiKey`, apiHost)
		w.Write([]byte(swaggerYAMLTemplate))
	}))

	testPluginConfig := fmt.Sprintf(`version: '1'
services:
  openapi:
    telemetry:
      http_endpoint:
        url: http://%s/v1/metrics
        provider_schema_properties: ["some_token", "some_header"]
    swagger-url: %s
    insecure_skip_verify: true`, apiHost, swaggerServer.URL)

	file := createPluginConfigFile(testPluginConfig)
	defer os.Remove(file.Name())

	otfVarPluginConfigEnvVariableName := fmt.Sprintf("OTF_VAR_%s_PLUGIN_CONFIGURATION_FILE", providerName)
	os.Setenv(otfVarPluginConfigEnvVariableName, file.Name())

	p := openapi.ProviderOpenAPI{ProviderName: providerName}
	provider, err := p.CreateSchemaProvider()
	assert.NoError(t, err)

	var testAccProviders = map[string]terraform.ResourceProvider{providerName: provider}
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  testAccProviders,
		PreCheck:   func() { testAccPreCheck(t, swaggerServer.URL) },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "openapi" {
  some_token = "token_value"
  some_header = "header_value" 
}
resource "openapi_cdns_v1" "my_cdn" { 
   label = "some_label"
}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"openapi_cdns_v1.my_cdn", "label", "some_label"),
					func(s *terraform.State) error { // asserting that the httpendpoint server received the expected metrics counter
						if !httpEndpointTelemetryCalled {
							return fmt.Errorf("http endpoint telemetry not called")
						}
						if headersReceived.Get("some_token") != "token_value" {
							return fmt.Errorf("expected header `some_token` in the metric API not received or not expected value received: %s", headersReceived.Get("some_token"))
						}
						if headersReceived.Get("some_header") != "header_value" {
							return fmt.Errorf("expected header `some_header` in the metric API not received or not expected value received: %s", headersReceived.Get("some_header"))
						}
						expectedPluginVersionMetric := `{"metric_type":"IncCounter","metric_name":"terraform.openapi_plugin_version.total_runs","tags":["openapi_plugin_version:dev"]}`
						if metricsReceived[0] != expectedPluginVersionMetric {
							return fmt.Errorf("metrics received [%s] don't match the expected ones [%s]", metricsReceived[0], expectedPluginVersionMetric)
						}
						expectedResourceMetrics := `{"metric_type":"IncCounter","metric_name":"terraform.provider","tags":["provider_name:openapi","resource_name:cdns_v1","terraform_operation:create"]}`
						if metricsReceived[2] != expectedResourceMetrics {
							return fmt.Errorf("metrics received [%s] don't match the expected ones [%s]", metricsReceived[2], expectedResourceMetrics)
						}
						return nil
					},
				),
			},
		},
	})
	os.Unsetenv(otfVarPluginConfigEnvVariableName)
}

// TestAcc_ProviderConfiguration_PluginExternalFile_GraphiteTelemetry confirms regressions introduced in the logic related to the plugin
// external configuration. This test confirms that the plugin is able to start up properly and functions as expected even
// when the plugin uses the external configuration containing:
// - Graphite telemetry configuration
// - Service configurations
func TestAcc_ProviderConfiguration_PluginExternalFile_GraphiteTelemetry(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"someID", "label": "some_label"}`))
		w.WriteHeader(http.StatusOK)
	}))
	apiHost := apiServer.URL[7:]

	swaggerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		swaggerYAMLTemplate := fmt.Sprintf(`swagger: "2.0"
host: "%s"

schemes:
- "http"

paths:
  /v1/cdns:
    post:
      parameters:
      - in: "body"
        name: "body"
        description: "Created CDN"
        required: true
        schema:
          $ref: "#/definitions/ContentDeliveryNetworkV1"
      responses:
        201:
          description: "successful operation"
          schema:
            $ref: "#/definitions/ContentDeliveryNetworkV1"

  /v1/cdns/{cdn_id}:
    get:
      parameters:
      - name: "cdn_id"
        in: "path"
        description: "The cdn id that needs to be fetched."
        required: true
        type: "string"
      responses:
        200:
          description: "successful operation"
          schema:
            $ref: "#/definitions/ContentDeliveryNetworkV1"
    delete:
      parameters:
      - name: "id"
        in: "path"
        description: "The cdn that needs to be deleted"
        required: true
        type: "string"
      responses:
        204:
          description: "successful operation, no content is returned"
definitions:
  ContentDeliveryNetworkV1:
    type: "object"
    required:
      - label
    properties:
      id:
        type: "string"
        readOnly: true
      label:
        type: "string"`, apiHost)
		w.Write([]byte(swaggerYAMLTemplate))
	}))

	metricChannel := make(chan string)
	pc, telemetryHost, telemetryPort := graphiteServer(metricChannel)
	defer pc.Close()

	testPluginConfig := fmt.Sprintf(`version: '1'
services:
  openapi:
    telemetry:
      graphite:
        host: %s
        port: %s
    swagger-url: %s
    insecure_skip_verify: true`, telemetryHost, telemetryPort, swaggerServer.URL)

	file := createPluginConfigFile(testPluginConfig)
	defer os.Remove(file.Name())

	otfVarPluginConfigEnvVariableName := fmt.Sprintf("OTF_VAR_%s_PLUGIN_CONFIGURATION_FILE", providerName)
	os.Setenv(otfVarPluginConfigEnvVariableName, file.Name())

	p := openapi.ProviderOpenAPI{ProviderName: providerName}
	provider, err := p.CreateSchemaProvider()
	assert.NoError(t, err)

	var testAccProviders = map[string]terraform.ResourceProvider{providerName: provider}
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers:  testAccProviders,
		PreCheck:   func() { testAccPreCheck(t, swaggerServer.URL) },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "openapi_cdns_v1" "my_cdn" { label = "some_label"}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"openapi_cdns_v1.my_cdn", "label", "some_label"),
					func(s *terraform.State) error { // asserting that the graphite server received the expected metrics counter
						assertExpectedMetric(t, metricChannel, "terraform.providers.openapi.total_runs:1|c")
						assertExpectedMetric(t, metricChannel, "terraform.openapi_plugin_version.dev.total_runs:1|c")
						return nil
					},
				),
			},
		},
	})
	os.Unsetenv(otfVarPluginConfigEnvVariableName)
}

func assertExpectedMetric(t *testing.T, metricChannel chan string, expectedMetric string) {
	select {
	case metricReceived := <-metricChannel:
		assert.Contains(t, metricReceived, expectedMetric)
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("[FAIL] '%s' not received within the expected timeframe (timed out)", expectedMetric)
	}
}

func createPluginConfigFile(content string) *os.File {
	file, err := ioutil.TempFile("", "terraform-provider-openapi.yaml")
	if err != nil {
		log.Fatal(err)
	}
	file.Write([]byte(content))
	return file
}

func graphiteServer(metricChannel chan string) (net.PacketConn, string, string) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		log.Fatal(err)
	}
	telemetryServer := pc.LocalAddr().String()
	telemetryHost := strings.Split(telemetryServer, ":")[0]
	telemetryPort := strings.Split(telemetryServer, ":")[1]
	go func() {
		for {
			buf := make([]byte, 1024)
			n, _, err := pc.ReadFrom(buf)
			if err != nil {
				continue
			}
			body := string(buf[:n])
			metricChannel <- body
		}
	}()
	return pc, telemetryHost, telemetryPort
}
