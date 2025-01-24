package apple_mdm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com/stretchr/testify/require"
)

func TestVerifyHostMDMProfiles(t *testing.T) {
	ctx := context.Background()
	host := &mdmlab.Host{
		DetailUpdatedAt: time.Now(),
	}
	ds := new(mock.Store)

	tests := []struct {
		name             string
		expectedProfiles map[string]*mdmlab.ExpectedMDMProfile
		retryCounts      []mdmlab.HostMDMProfileRetryCount
		installed        map[string]*mdmlab.HostMacOSProfile
		toVerify         []string
		toFail           []string
		toRetry          []string
		expectErr        bool
	}{
		{
			name:      "error on getting expected profiles",
			expectErr: true,
		},
		{
			name:             "all profiles verified",
			expectedProfiles: map[string]*mdmlab.ExpectedMDMProfile{"profile1": {}},
			installed:        map[string]*mdmlab.HostMacOSProfile{"profile1": {}},
			toVerify:         []string{"profile1"},
		},
		{
			name: "profiles missing, not within grace period, no retries yet",
			expectedProfiles: map[string]*mdmlab.ExpectedMDMProfile{
				"profile1": {},
				"profile2": {EarliestInstallDate: host.DetailUpdatedAt.Add(-24 * time.Hour)},
			},
			installed: map[string]*mdmlab.HostMacOSProfile{"profile1": {}},
			toVerify:  []string{"profile1"},
			toRetry:   []string{"profile2"},
		},
		{
			name:             "profiles missing, with and without retries, not within grace period",
			expectedProfiles: map[string]*mdmlab.ExpectedMDMProfile{"profile1": {}, "profile2": {}},
			retryCounts: []mdmlab.HostMDMProfileRetryCount{
				{ProfileIdentifier: "profile1", Retries: 0},
				{ProfileIdentifier: "profile2", Retries: 1},
			},
			installed: map[string]*mdmlab.HostMacOSProfile{},
			toRetry:   []string{"profile1"},
			toFail:    []string{"profile2"},
		},
		{
			name: "host profile installed prior to uploading profile to MDMlab",
			expectedProfiles: map[string]*mdmlab.ExpectedMDMProfile{
				"profile1": {EarliestInstallDate: time.Now().Add(-2 * time.Hour)},
			},
			installed: map[string]*mdmlab.HostMacOSProfile{
				"profile1": {InstallDate: time.Now().Add(-24 * time.Hour)},
			},
			toRetry: []string{"profile1"},
		},
		{
			name: "host profile installed prior to uploading profile to MDMlab with max retries",
			expectedProfiles: map[string]*mdmlab.ExpectedMDMProfile{
				"profile1": {EarliestInstallDate: time.Now().Add(-2 * time.Hour)},
			},
			installed: map[string]*mdmlab.HostMacOSProfile{
				"profile1": {InstallDate: time.Now().Add(-24 * time.Hour)},
			},
			retryCounts: []mdmlab.HostMDMProfileRetryCount{
				{ProfileIdentifier: "profile1", Retries: 1},
			},
			toFail: []string{"profile1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// setup mocks
			ds.GetHostMDMProfilesExpectedForVerificationFunc = func(ctx context.Context, host *mdmlab.Host) (map[string]*mdmlab.ExpectedMDMProfile, error) {
				if tc.expectErr {
					return nil, errors.New("error")
				}
				return tc.expectedProfiles, nil
			}

			ds.GetHostMDMProfilesRetryCountsFunc = func(ctx context.Context, host *mdmlab.Host) ([]mdmlab.HostMDMProfileRetryCount, error) {
				return tc.retryCounts, nil
			}

			ds.UpdateHostMDMProfilesVerificationFunc = func(ctx context.Context, host *mdmlab.Host, verified, toFail, toRetry []string) error {
				require.ElementsMatch(t, tc.toVerify, verified, "verified profiles do not match")
				require.ElementsMatch(t, tc.toFail, toFail, "failed profiles do not match")
				require.ElementsMatch(t, tc.toRetry, toRetry, "retried profiles do not match")
				return nil
			}

			// run the test
			err := VerifyHostMDMProfiles(ctx, ds, host, tc.installed)
			if tc.expectErr {
				require.Error(t, err)
				require.False(
					t,
					ds.UpdateHostMDMProfilesVerificationFuncInvoked,
					"UpdateHostMDMProfilesVerificationFunc should not have been called",
				)
			} else {
				require.NoError(t, err)
				require.True(t, ds.UpdateHostMDMProfilesVerificationFuncInvoked, "UpdateHostMDMProfilesVerificationFunc should have been called")
			}

			ds.UpdateHostMDMProfilesVerificationFuncInvoked = false
		})
	}
}
