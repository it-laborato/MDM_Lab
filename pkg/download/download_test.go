package download

import (
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/pkg/mdmlabhttp"
	"github.com/stretchr/testify/require"
)

func TestDownloadNotFoundNoRetries(t *testing.T) {
	c := mdmlabhttp.NewClient()
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "not-used")
	url, err := url.Parse("https://github.com/mdmlabdm/non-existent")
	require.NoError(t, err)
	start := time.Now()
	err = Download(c, url, outputFile)
	require.Error(t, err)
	require.ErrorIs(t, err, NotFound)
	require.True(t, time.Since(start) < backoffMaxElapsedTime)
}
