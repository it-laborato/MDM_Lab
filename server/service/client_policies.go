package service

import (
	"fmt"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

func (c *Client) CreateGlobalPolicy(name, query, description, resolution, platform string) error {
	req := globalPolicyRequest{
		Name:        name,
		Query:       query,
		Description: description,
		Resolution:  resolution,
		Platform:    platform,
	}
	verb, path := "POST", "/api/latest/mdmlab/global/policies"
	var responseBody globalPolicyResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// ApplyPolicies sends the list of Policies to be applied to the
// MDMlab instance.
func (c *Client) ApplyPolicies(specs []*mdmlab.PolicySpec) error {
	req := applyPolicySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/mdmlab/spec/policies"
	var responseBody applyPolicySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetPolicies retrieves the list of Policies. Inherited policies are excluded.
func (c *Client) GetPolicies(teamID *uint) ([]*mdmlab.Policy, error) {
	verb, path := "GET", ""
	if teamID != nil {
		path = fmt.Sprintf("/api/latest/mdmlab/teams/%d/policies", *teamID)
	} else {
		path = "/api/latest/mdmlab/policies"
	}
	// The response body also works for listTeamPoliciesResponse because they contain some of the same members.
	var responseBody listGlobalPoliciesResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Policies, nil
}

// DeletePolicies deletes several policies.
func (c *Client) DeletePolicies(teamID *uint, ids []uint) error {
	verb, path := "POST", ""
	req := deleteTeamPoliciesRequest{IDs: ids}
	if teamID != nil {
		path = fmt.Sprintf("/api/latest/mdmlab/teams/%d/policies/delete", *teamID)
		req.TeamID = *teamID
	} else {
		path = "/api/latest/mdmlab/policies/delete"
	}
	// The response body also works for deleteTeamPoliciesResponse because they contain some of the same members.
	var responseBody deleteGlobalPoliciesResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
