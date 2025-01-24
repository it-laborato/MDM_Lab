package service

import "github.com/it-laborato/MDM_Lab/server/mdmlab"

func (c *Client) SaveSecretVariables(secretVariables []mdmlab.SecretVariable, dryRun bool) error {
	verb, path := "PUT", "/api/latest/mdmlab/spec/secret_variables"
	params := secretVariablesRequest{
		SecretVariables: secretVariables,
		DryRun:          dryRun,
	}
	var responseBody secretVariablesResponse
	return c.authenticatedRequest(params, verb, path, &responseBody)
}
