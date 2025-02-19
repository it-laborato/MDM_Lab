package update

import (
	"testing"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
)

func TestSwiftDialogUpdatesDisabled(t *testing.T) {
	cfg := &mdmlab.OrbitConfig{}
	cfg.Notifications.NeedsMDMMigration = true
	cfg.Notifications.RenewEnrollmentProfile = true
	r := ApplySwiftDialogDownloaderMiddleware(nil)

	// we used to get a panic if updates were disabled (see #11980)
	err := r.Run(cfg)
	require.NoError(t, err)
}
