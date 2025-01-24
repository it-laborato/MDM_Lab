package service

import (
	"fmt"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

// SearchTargets searches for the supplied targets in the MDMlab instance.
func (c *Client) SearchTargets(query string, hostIDs, labelIDs []uint) (*mdmlab.TargetSearchResults, error) {
	req := searchTargetsRequest{
		MatchQuery: query,
		Selected: mdmlab.HostTargets{
			LabelIDs: labelIDs,
			HostIDs:  hostIDs,
			// TODO handle TeamIDs
		},
	}
	verb, path := "POST", "/api/latest/mdmlab/targets"
	var responseBody searchTargetsResponse
	err := c.authenticatedRequest(req, verb, path, &responseBody)
	if err != nil {
		return nil, fmt.Errorf("SearchTargets: %s", err)
	}

	hosts := make([]*mdmlab.Host, len(responseBody.Targets.Hosts))
	for i, h := range responseBody.Targets.Hosts {
		hosts[i] = h.Host
	}

	labels := make([]*mdmlab.Label, len(responseBody.Targets.Labels))
	for i, h := range responseBody.Targets.Labels {
		labels[i] = h.Label
	}

	return &mdmlab.TargetSearchResults{
		Hosts:  hosts,
		Labels: labels,
	}, nil
}
