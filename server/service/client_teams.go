package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

// ListTeams retrieves the list of teams.
func (c *Client) ListTeams(query string) ([]mdmlab.Team, error) {
	verb, path := "GET", "/api/latest/mdmlab/teams"
	var responseBody listTeamsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Teams, nil
}

// CreateTeam creates a new team.
func (c *Client) CreateTeam(teamPayload mdmlab.TeamPayload) (*mdmlab.Team, error) {
	req := createTeamRequest{
		TeamPayload: teamPayload,
	}
	verb, path := "POST", "/api/latest/mdmlab/teams"
	var responseBody teamResponse
	err := c.authenticatedRequest(req, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Team, nil
}

func (c *Client) GetTeam(teamID uint) (*mdmlab.Team, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/mdmlab/teams/%d", teamID)
	var responseBody getTeamResponse
	if err := c.authenticatedRequest(getTeamRequest{}, verb, path, &responseBody); err != nil {
		return nil, err
	}
	return responseBody.Team, nil
}

// DeleteTeam deletes a team.
func (c *Client) DeleteTeam(teamID uint) error {
	verb, path := "DELETE", "/api/latest/mdmlab/teams/"+strconv.FormatUint(uint64(teamID), 10)
	var responseBody deleteTeamResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}

// ApplyTeams sends the list of Teams to be applied to the
// MDMlab instance.
func (c *Client) ApplyTeams(specs []json.RawMessage, opts mdmlab.ApplyTeamSpecOptions) (map[string]uint, error) {
	verb, path := "POST", "/api/latest/mdmlab/spec/teams"
	var responseBody applyTeamSpecsResponse
	params := map[string]interface{}{"specs": specs}
	if opts.DryRun && opts.DryRunAssumptions != nil {
		params["dry_run_assumptions"] = opts.DryRunAssumptions
	}
	err := c.authenticatedRequestWithQuery(params, verb, path, &responseBody, opts.RawQuery())
	if err != nil {
		return nil, err
	}
	return responseBody.TeamIDsByName, nil
}

// ApplyTeamProfiles sends the list of profiles to be applied for the specified
// team.
func (c *Client) ApplyTeamProfiles(tmName string, profiles []mdmlab.MDMProfileBatchPayload, opts mdmlab.ApplyTeamSpecOptions) error {
	verb, path := "POST", "/api/latest/mdmlab/mdm/profiles/batch"
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return err
	}
	query.Add("team_name", tmName)
	if opts.DryRunAssumptions != nil && opts.DryRunAssumptions.WindowsEnabledAndConfigured.Valid {
		query.Add("assume_enabled", strconv.FormatBool(opts.DryRunAssumptions.WindowsEnabledAndConfigured.Value))
	}
	return c.authenticatedRequestWithQuery(map[string]interface{}{"profiles": profiles}, verb, path, nil, query.Encode())
}

// ApplyTeamScripts sends the list of scripts to be applied for the specified
// team.
func (c *Client) ApplyTeamScripts(tmName string, scripts []mdmlab.ScriptPayload, opts mdmlab.ApplySpecOptions) ([]mdmlab.ScriptResponse, error) {
	verb, path := "POST", "/api/latest/mdmlab/scripts/batch"
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	query.Add("team_name", tmName)

	var resp batchSetScriptsResponse
	err = c.authenticatedRequestWithQuery(map[string]interface{}{"scripts": scripts}, verb, path, &resp, query.Encode())
	return resp.Scripts, err
}

func (c *Client) ApplyTeamSoftwareInstallers(tmName string, softwareInstallers []mdmlab.SoftwareInstallerPayload, opts mdmlab.ApplySpecOptions) ([]mdmlab.SoftwarePackageResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	query.Add("team_name", tmName)
	return c.applySoftwareInstallers(softwareInstallers, query, opts.DryRun)
}

func (c *Client) ApplyTeamAppStoreAppsAssociation(tmName string, vppBatchPayload []mdmlab.VPPBatchPayload, opts mdmlab.ApplySpecOptions) ([]mdmlab.VPPAppResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	query.Add("team_name", tmName)
	return c.applyAppStoreAppsAssociation(vppBatchPayload, query)
}

func (c *Client) ApplyNoTeamAppStoreAppsAssociation(vppBatchPayload []mdmlab.VPPBatchPayload, opts mdmlab.ApplySpecOptions) ([]mdmlab.VPPAppResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	return c.applyAppStoreAppsAssociation(vppBatchPayload, query)
}

func (c *Client) applyAppStoreAppsAssociation(vppBatchPayload []mdmlab.VPPBatchPayload, query url.Values) ([]mdmlab.VPPAppResponse, error) {
	verb, path := "POST", "/api/latest/mdmlab/software/app_store_apps/batch"
	var appsResponse batchAssociateAppStoreAppsResponse
	err := c.authenticatedRequestWithQuery(map[string]interface{}{"app_store_apps": vppBatchPayload}, verb, path, &appsResponse, query.Encode())
	if err != nil {
		return nil, err
	}
	return appsResponse.Apps, nil
}
