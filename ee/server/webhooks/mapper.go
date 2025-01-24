package webhooks

import (
	"net/url"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	mdmlabwebhooks "github.com/it-laborato/MDM_Lab/server/webhooks"
)

type Mapper struct {
	mdmlabwebhooks.Mapper
}

func NewMapper() mdmlabwebhooks.VulnMapper {
	return &Mapper{}
}

func (m *Mapper) GetPayload(
	hostBaseURL *url.URL,
	hosts []mdmlab.HostVulnerabilitySummary,
	cve string,
	meta mdmlab.CVEMeta,
) mdmlabwebhooks.WebhookPayload {
	r := m.Mapper.GetPayload(hostBaseURL,
		hosts,
		cve,
		meta,
	)
	r.EPSSProbability = meta.EPSSProbability
	r.CVSSScore = meta.CVSSScore
	r.CISAKnownExploit = meta.CISAKnownExploit
	r.CVEPublished = meta.Published
	return r
}
