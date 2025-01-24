package webhooks

import (
	"net/url"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestGetPayload(t *testing.T) {
	serverURL, err := url.Parse("http://mywebsite.com")
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	vuln := mdmlab.SoftwareVulnerability{
		CVE:        "cve-1",
		SoftwareID: 1,
	}
	meta := mdmlab.CVEMeta{
		CVE:              "cve-1",
		CVSSScore:        ptr.Float64(1),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
		Published:        ptr.Time(now),
	}

	sut := Mapper{}

	result := sut.GetPayload(serverURL, nil, vuln.CVE, meta)
	require.Equal(t, *meta.CISAKnownExploit, *result.CISAKnownExploit)
	require.Equal(t, *meta.EPSSProbability, *result.EPSSProbability)
	require.Equal(t, *meta.CVSSScore, *result.CVSSScore)
	require.NotNil(t, result.CVEPublished)
	require.Equal(t, *meta.Published, *result.CVEPublished)
}
