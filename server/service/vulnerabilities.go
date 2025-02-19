package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/it-laborato/MDM_Lab/server/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

var freeValidVulnSortColumns = []string{
	"cve",
	"hosts_count",
	"host_count_updated_at",
	"created_at",
}

type cveNotFoundError struct{}

var _ mdmlab.NotFoundError = (*cveNotFoundError)(nil)

func (p cveNotFoundError) Error() string {
	return "This is not a known CVE. None of MDMlab’s vulnerability sources are aware of this CVE."
}

func (p cveNotFoundError) IsNotFound() bool {
	return true
}

type listVulnerabilitiesRequest struct {
	mdmlab.VulnListOptions
}

type listVulnerabilitiesResponse struct {
	Vulnerabilities []mdmlab.VulnerabilityWithMetadata `json:"vulnerabilities"`
	Count           uint                              `json:"count"`
	CountsUpdatedAt time.Time                         `json:"counts_updated_at"`
	Meta            *mdmlab.PaginationMetadata         `json:"meta,omitempty"`
	Err             error                             `json:"error,omitempty"`
}

// Allow formats like: CVE-2017-12345, cve-2017-12345
var cveRegex = regexp.MustCompile(`(?i)^CVE-\d{4}-\d{4}\d*$`)

func (r listVulnerabilitiesResponse) error() error { return r.Err }

func listVulnerabilitiesEndpoint(ctx context.Context, req interface{}, svc mdmlab.Service) (errorer, error) {
	request := req.(*listVulnerabilitiesRequest)
	vulns, meta, err := svc.ListVulnerabilities(ctx, request.VulnListOptions)
	if err != nil {
		return listVulnerabilitiesResponse{Err: err}, nil
	}

	count, err := svc.CountVulnerabilities(ctx, request.VulnListOptions)
	if err != nil {
		return listVulnerabilitiesResponse{Err: err}, nil
	}

	updatedAt := time.Now()
	for _, vuln := range vulns {
		if vuln.HostsCountUpdatedAt.Before(updatedAt) {
			updatedAt = vuln.HostsCountUpdatedAt
		}
	}

	return listVulnerabilitiesResponse{
		Vulnerabilities: vulns,
		Meta:            meta,
		Count:           count,
		CountsUpdatedAt: updatedAt,
	}, nil
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt mdmlab.VulnListOptions) ([]mdmlab.VulnerabilityWithMetadata, *mdmlab.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, mdmlab.ActionRead); err != nil {
		return nil, nil, err
	}

	if len(opt.ValidSortColumns) == 0 {
		opt.ValidSortColumns = freeValidVulnSortColumns
	}

	if !opt.HasValidSortColumn() {
		return nil, nil, badRequest("invalid order key")
	}

	if opt.KnownExploit && !opt.IsEE {
		return nil, nil, mdmlab.ErrMissingLicense
	}

	vulns, meta, err := svc.ds.ListVulnerabilities(ctx, opt)
	if err != nil {
		return nil, nil, err
	}

	for i, vuln := range vulns {
		vulns[i].DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE.CVE)
	}

	return vulns, meta, nil
}

func (svc *Service) CountVulnerabilities(ctx context.Context, opts mdmlab.VulnListOptions) (uint, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AuthzSoftwareInventory{
		TeamID: opts.TeamID,
	}, mdmlab.ActionRead); err != nil {
		return 0, err
	}

	return svc.ds.CountVulnerabilities(ctx, opts)
}

func (svc *Service) IsCVEKnownToMDMlab(ctx context.Context, cve string) (bool, error) {
	return svc.ds.IsCVEKnownToMDMlab(ctx, cve)
}

type getVulnerabilityRequest struct {
	CVE    string `url:"cve"`
	TeamID *uint  `query:"team_id,optional"`
}

type getVulnerabilityResponse struct {
	Vulnerability *mdmlab.VulnerabilityWithMetadata `json:"vulnerability"`
	OSVersions    []*mdmlab.VulnerableOS            `json:"os_versions"`
	Software      []*mdmlab.VulnerableSoftware      `json:"software"`
	Err           error                            `json:"error,omitempty"`
	statusCode    int
}

func (r getVulnerabilityResponse) error() error { return r.Err }

func (r getVulnerabilityResponse) Status() int {
	if r.statusCode == 0 {
		return http.StatusOK
	}
	return r.statusCode
}

func getVulnerabilityEndpoint(ctx context.Context, req interface{}, svc mdmlab.Service) (errorer, error) {
	request := req.(*getVulnerabilityRequest)

	vuln, known, err := svc.Vulnerability(ctx, request.CVE, request.TeamID, false)
	if err != nil {
		return getVulnerabilityResponse{Err: err}, nil
	}
	if vuln == nil && known {
		// Return 204 status code if the vulnerability is known to MDMlab but does not match any host software/OS
		return getVulnerabilityResponse{statusCode: http.StatusNoContent}, nil
	}

	vuln.DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE.CVE)

	osVersions, _, err := svc.ListOSVersionsByCVE(ctx, vuln.CVE.CVE, request.TeamID)
	if err != nil {
		return getVulnerabilityResponse{Err: err}, nil
	}

	software, _, err := svc.ListSoftwareByCVE(ctx, vuln.CVE.CVE, request.TeamID)
	if err != nil {
		return getVulnerabilityResponse{Err: err}, nil
	}

	return getVulnerabilityResponse{
		Vulnerability: vuln,
		OSVersions:    osVersions,
		Software:      software,
	}, nil
}

func (svc *Service) Vulnerability(ctx context.Context, cve string, teamID *uint, useCVSScores bool) (vuln *mdmlab.VulnerabilityWithMetadata,
	known bool, err error,
) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AuthzSoftwareInventory{TeamID: teamID}, mdmlab.ActionRead); err != nil {
		return nil, false, err
	}

	if err := svc.authz.Authorize(ctx, &mdmlab.Host{TeamID: teamID}, mdmlab.ActionRead); err != nil {
		return nil, false, err
	}

	if !cveRegex.Match([]byte(cve)) {
		return nil, false, badRequest("That vulnerability (CVE) is not valid. Try updating your search to use CVE format: \"CVE-YYYY-<4 or more digits>\"")
	}

	if teamID != nil && *teamID != 0 {
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return nil, false, ctxerr.Wrap(ctx, err, "checking if team exists")
		} else if !exists {
			return nil, false, authz.ForbiddenWithInternal("team does not exist", nil, nil, nil)
		}
	}

	vuln, err = svc.ds.Vulnerability(ctx, cve, teamID, useCVSScores)
	switch {
	case mdmlab.IsNotFound(err):
		var errKnown error
		known, errKnown = svc.ds.IsCVEKnownToMDMlab(ctx, cve)
		if errKnown != nil {
			return nil, false, errKnown
		}
		if !known {
			return nil, false, cveNotFoundError{}
		}
	case err != nil:
		return nil, false, err
	default:
		known = true
	}

	return vuln, known, nil
}

func (svc *Service) ListOSVersionsByCVE(ctx context.Context, cve string, teamID *uint) (result []*mdmlab.VulnerableOS, updatedAt time.Time, err error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.Host{TeamID: teamID}, mdmlab.ActionRead); err != nil {
		return nil, updatedAt, err
	}
	return svc.ds.OSVersionsByCVE(ctx, cve, teamID)
}

func (svc *Service) ListSoftwareByCVE(ctx context.Context, cve string, teamID *uint) (result []*mdmlab.VulnerableSoftware, updatedAt time.Time, err error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AuthzSoftwareInventory{TeamID: teamID}, mdmlab.ActionRead); err != nil {
		return nil, updatedAt, err
	}
	return svc.ds.SoftwareByCVE(ctx, cve, teamID)
}
