package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com:it-laborato/MDM_Lab/orbit/pkg/packaging"
	"github.com:it-laborato/MDM_Lab/orbit/pkg/update"
	"github.com:it-laborato/MDM_Lab/pkg/nettest"
	"github.com/stretchr/testify/require"
)

func TestPackage(t *testing.T) {
	nettest.Run(t)

	updateOpt := update.DefaultOptions
	updateOpt.RootDirectory = t.TempDir()
	updatesData, err := packaging.InitializeUpdates(updateOpt)
	require.NoError(t, err)

	// --type is required
	runAppCheckErr(t, []string{"package", "deb"}, "Required flag \"type\" not set")

	// if you provide -mdmlab-url & --enroll-secret are required together
	runAppCheckErr(t, []string{"package", "--type=deb", "--mdmlab-url=https://localhost:8080"}, "--enroll-secret and --mdmlab-url must be provided together")
	runAppCheckErr(t, []string{"package", "--type=deb", "--enroll-secret=foobar"}, "--enroll-secret and --mdmlab-url must be provided together")

	// --insecure and --mdmlab-certificate are mutually exclusive
	runAppCheckErr(t, []string{"package", "--type=deb", "--insecure", "--mdmlab-certificate=test123"}, "--insecure and --mdmlab-certificate may not be provided together")

	// Test invalid PEM file provided in --mdmlab-certificate.
	certDir := t.TempDir()
	mdmlabCertificate := filepath.Join(certDir, "mdmlab.pem")
	err = os.WriteFile(mdmlabCertificate, []byte("undefined"), os.FileMode(0o644))
	require.NoError(t, err)
	runAppCheckErr(t, []string{"package", "--type=deb", fmt.Sprintf("--mdmlab-certificate=%s", mdmlabCertificate)}, fmt.Sprintf("failed to read mdmlab server certificate %q: invalid PEM file", mdmlabCertificate))

	if runtime.GOOS != "linux" {
		runAppCheckErr(t, []string{"package", "--type=msi", "--native-tooling"}, "native tooling is only available in Linux")
	}

	t.Run("deb", func(t *testing.T) {
		runAppForTest(t, []string{"package", "--type=deb", "--insecure", "--disable-open-folder"})
		info, err := os.Stat(fmt.Sprintf("mdmlab-osquery_%s_amd64.deb", updatesData.OrbitVersion))
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0)) // TODO verify contents
	})

	t.Run("--use-sytem-configuration can't be used on installers that aren't pkg", func(t *testing.T) {
		for _, p := range []string{"deb", "msi", "rpm", ""} {
			runAppCheckErr(
				t,
				[]string{"package", fmt.Sprintf("--type=%s", p), "--use-system-configuration"},
				"--use-system-configuration is only available for pkg installers",
			)
		}
	})

	// mdmlab-osquery.msi
	// runAppForTest(t, []string{"package", "--type=msi", "--insecure"}) TODO: this is currently failing on Github runners due to permission issues
	// info, err = os.Stat("orbit-osquery_0.0.3.msi")
	// require.NoError(t, err)
	// require.Greater(t, info.Size(), int64(0))

	// runAppForTest(t, []string{"package", "--type=pkg", "--insecure"}) TODO: had a hard time getting xar installed on Ubuntu
}
