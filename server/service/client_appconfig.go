package service

import (
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/version"
)

// ApplyAppConfig sends the application config to be applied to the MDMlab instance.
func (c *Client) ApplyAppConfig(payload interface{}, opts mdmlab.ApplySpecOptions) error {
	verb, path := "PATCH", "/api/latest/mdmlab/config"
	var responseBody appConfigResponse
	return c.authenticatedRequestWithQuery(payload, verb, path, &responseBody, opts.RawQuery())
}

// ApplyNoTeamProfiles sends the list of profiles to be applied for the hosts
// in no team.
func (c *Client) ApplyNoTeamProfiles(profiles []mdmlab.MDMProfileBatchPayload, opts mdmlab.ApplySpecOptions, assumeEnabled bool) error {
	verb, path := "POST", "/api/latest/mdmlab/mdm/profiles/batch"
	query := opts.RawQuery()
	if assumeEnabled {
		if query != "" {
			query += "&"
		}
		query += "assume_enabled=true"
	}
	return c.authenticatedRequestWithQuery(map[string]interface{}{"profiles": profiles}, verb, path, nil, query)
}

// GetAppConfig fetches the application config from the server API
func (c *Client) GetAppConfig() (*mdmlab.EnrichedAppConfig, error) {
	verb, path := "GET", "/api/latest/mdmlab/config"
	var responseBody mdmlab.EnrichedAppConfig
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return &responseBody, err
}

// GetEnrollSecretSpec fetches the enroll secrets stored on the server
func (c *Client) GetEnrollSecretSpec() (*mdmlab.EnrollSecretSpec, error) {
	verb, path := "GET", "/api/latest/mdmlab/spec/enroll_secret"
	var responseBody getEnrollSecretSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// ApplyEnrollSecretSpec applies the enroll secrets.
func (c *Client) ApplyEnrollSecretSpec(spec *mdmlab.EnrollSecretSpec, opts mdmlab.ApplySpecOptions) error {
	req := applyEnrollSecretSpecRequest{Spec: spec}
	verb, path := "POST", "/api/latest/mdmlab/spec/enroll_secret"
	var responseBody applyEnrollSecretSpecResponse
	return c.authenticatedRequestWithQuery(req, verb, path, &responseBody, opts.RawQuery())
}

func (c *Client) Version() (*version.Info, error) {
	verb, path := "GET", "/api/latest/mdmlab/version"
	var responseBody versionResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Info, err
}
