package service

import (
	"context"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

var eeValidVulnSortColumns = []string{
	"cve",
	"hosts_count",
	"created_at",
	"cvss_score",
	"epss_probability",
	"cve_published",
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt mdmlab.VulnListOptions) ([]mdmlab.VulnerabilityWithMetadata, *mdmlab.PaginationMetadata, error) {
	opt.ValidSortColumns = eeValidVulnSortColumns
	opt.IsEE = true
	return svc.Service.ListVulnerabilities(ctx, opt)
}

func (svc *Service) Vulnerability(ctx context.Context, cve string, teamID *uint, useCVSScores bool) (vuln *mdmlab.VulnerabilityWithMetadata,
	known bool, err error) {
	return svc.Service.Vulnerability(ctx, cve, teamID, true)
}
