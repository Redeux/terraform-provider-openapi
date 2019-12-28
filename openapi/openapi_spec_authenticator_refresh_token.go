package openapi

import (
	"fmt"
	"github.com/dikhan/http_goclient"
	"net/http"
	"strings"
)

// Api Key Header Auth
type apiRefreshTokenAuthenticator struct {
	terraformConfigurationName string
	apiKey
	refreshTokenURL string
	httpClient      http_goclient.HttpClientIface
}

func newAPIRefreshTokenAuthenticator(name, refreshToken, refreshTokenURL, terraformConfigurationName string) apiRefreshTokenAuthenticator {
	return apiRefreshTokenAuthenticator{
		terraformConfigurationName: terraformConfigurationName,
		apiKey: apiKey{
			name:  name,
			value: refreshToken,
		},
		refreshTokenURL: refreshTokenURL,
		httpClient:      &http_goclient.HttpClient{HttpClient: &http.Client{}},
	}
}

func (a apiRefreshTokenAuthenticator) getContext() interface{} {
	return a.apiKey
}

func (a apiRefreshTokenAuthenticator) getType() authType {
	return authTypeAPIKeyHeader
}

// prepareAuth will send a post request to the refreshTokenURL and get the access token from the response Authorization
// header. Otherwise, it will fail.
func (a apiRefreshTokenAuthenticator) prepareAuth(authContext *authContext) error {
	apiKey := a.getContext().(apiKey)
	headers := map[string]string{apiKey.name: apiKey.value}
	r, err := a.httpClient.PostJson(a.refreshTokenURL, headers, nil, nil)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusNoContent {
		return fmt.Errorf("refresh token POST response '%s' status code '%d' not matching expected response status code [%d, %d]", a.refreshTokenURL, r.StatusCode, http.StatusOK, http.StatusNoContent)
	}
	accessToken := r.Header.Get(authorizationHeader)
	if accessToken == "" {
		return fmt.Errorf("refresh token POST response '%s' is missing the access token", a.refreshTokenURL)
	}
	if authContext.headers == nil {
		authContext.headers = map[string]string{}
	}
	authContext.headers[authorizationHeader] = accessToken
	return nil
}

func (a apiRefreshTokenAuthenticator) validate() error {
	if a.value == "" || strings.Trim(a.value, " ") == "Bearer" {
		return fmt.Errorf("required security definition '%s' is missing the value. Please make sure the property '%s' is configured with a value in the provider's terraform configuration", a.terraformConfigurationName, a.terraformConfigurationName)
	}
	return nil
}
