package service

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

// ListSoftwareVersions retrieves the software versions installed on hosts.
func (c *Client) ListSoftwareVersions(query string) ([]mdmlab.Software, error) {
	verb, path := "GET", "/api/latest/mdmlab/software/versions"
	var responseBody listSoftwareVersionsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Software, nil
}

// ListSoftwareTitles retrieves the software titles installed on hosts.
func (c *Client) ListSoftwareTitles(query string) ([]mdmlab.SoftwareTitleListResult, error) {
	verb, path := "GET", "/api/latest/mdmlab/software/titles"
	var responseBody listSoftwareTitlesResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.SoftwareTitles, nil
}

func (c *Client) ApplyNoTeamSoftwareInstallers(softwareInstallers []mdmlab.SoftwareInstallerPayload, opts mdmlab.ApplySpecOptions) ([]mdmlab.SoftwarePackageResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	return c.applySoftwareInstallers(softwareInstallers, query, opts.DryRun)
}

func (c *Client) applySoftwareInstallers(softwareInstallers []mdmlab.SoftwareInstallerPayload, query url.Values, dryRun bool) ([]mdmlab.SoftwarePackageResponse, error) {
	path := "/api/latest/mdmlab/software/batch"
	var resp batchSetSoftwareInstallersResponse
	if err := c.authenticatedRequestWithQuery(map[string]interface{}{"software": softwareInstallers}, "POST", path, &resp, query.Encode()); err != nil {
		return nil, err
	}
	if dryRun && resp.RequestUUID == "" {
		return nil, nil
	}

	requestUUID := resp.RequestUUID
	for {
		var resp batchSetSoftwareInstallersResultResponse
		if err := c.authenticatedRequestWithQuery(nil, "GET", path+"/"+requestUUID, &resp, query.Encode()); err != nil {
			return nil, err
		}
		switch {
		case resp.Status == mdmlab.BatchSetSoftwareInstallersStatusProcessing:
			time.Sleep(5 * time.Second)
		case resp.Status == mdmlab.BatchSetSoftwareInstallersStatusFailed:
			return nil, errors.New(resp.Message)
		case resp.Status == mdmlab.BatchSetSoftwareInstallersStatusCompleted:
			return resp.Packages, nil
		default:
			return nil, fmt.Errorf("unknown status: %q", resp.Status)
		}
	}
}
