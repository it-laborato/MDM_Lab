package webhooks

import (
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

type VulnArgs struct {
	Vulnerablities []mdmlab.SoftwareVulnerability
	Meta           map[string]mdmlab.CVEMeta
	AppConfig      *mdmlab.AppConfig
	Time           time.Time
}
