package apple_mdm

import (
	"testing"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleEnrollURL(t *testing.T) {
	cases := []struct {
		appConfig   *mdmlab.AppConfig
		expectedURL string
	}{
		{
			appConfig: &mdmlab.AppConfig{
				ServerSettings: mdmlab.ServerSettings{
					ServerURL: "https://foo.example.com",
				},
			},
			expectedURL: "https://foo.example.com/api/mdm/apple/enroll?token=tok",
		},
		{
			appConfig: &mdmlab.AppConfig{
				ServerSettings: mdmlab.ServerSettings{
					ServerURL: "https://foo.example.com/",
				},
			},
			expectedURL: "https://foo.example.com/api/mdm/apple/enroll?token=tok",
		},
	}

	for _, tt := range cases {
		enrollURL, err := EnrollURL("tok", tt.appConfig)
		require.NoError(t, err)
		require.Equal(t, tt.expectedURL, enrollURL)
	}
}
