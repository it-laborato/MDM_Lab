package main

import (
	"testing"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCfgsDiffer(t *testing.T) {
	for _, tc := range []struct {
		name           string
		overrideCfg    *serverOverridesConfig
		orbitConfig    *mdmlab.OrbitConfig
		desktopEnabled bool
		expected       bool
	}{
		{
			name:        "initial set of remote configuration",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Orbit:    "stable",
					Osqueryd: "stable",
					Desktop:  "stable",
				},
			},
			desktopEnabled: false,
			expected:       false,
		},
		{
			name:        "initial set of remote configuration, omit some channels",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Orbit: "stable",
				},
			},
			desktopEnabled: false,
			expected:       false,
		},
		{
			name:        "initial set of remote configuration, change orbit and omit some channels",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Orbit: "edge",
				},
			},
			desktopEnabled: false,
			expected:       true,
		},
		{
			name:        "initial set of remote configuration, set desktop when MDMlab Desktop disabled",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Desktop: "foobar",
				},
			},
			desktopEnabled: false,
			expected:       false,
		},
		{
			name:        "initial set of remote configuration, set desktop with MDMlab Desktop enabled",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Desktop: "foobar",
				},
			},
			desktopEnabled: true,
			expected:       true,
		},
		{
			name: "overrides update, set desktop with MDMlab Desktop enabled",
			overrideCfg: &serverOverridesConfig{
				DesktopChannel: "other",
			},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Desktop: "foobar",
				},
			},
			desktopEnabled: true,
			expected:       true,
		},
		{
			name: "overrides update, change orbit",
			overrideCfg: &serverOverridesConfig{
				OrbitChannel: "first",
			},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Orbit: "second",
				},
			},
			desktopEnabled: false,
			expected:       true,
		},
		{
			name: "overrides update, change osqueryd",
			overrideCfg: &serverOverridesConfig{
				OsquerydChannel: "first",
			},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{
					Osqueryd: "second",
				},
			},
			desktopEnabled: false,
			expected:       true,
		},
		{
			name: "overrides update, empty means stable",
			overrideCfg: &serverOverridesConfig{
				OrbitChannel:    "stable",
				OsquerydChannel: "stable",
				DesktopChannel:  "stable",
			},
			orbitConfig: &mdmlab.OrbitConfig{
				UpdateChannels: &mdmlab.OrbitUpdateChannels{},
			},
			desktopEnabled: true,
			expected:       false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := cfgsDiffer(tc.overrideCfg, tc.orbitConfig, tc.desktopEnabled)
			require.Equal(t, tc.expected, v)
		})
	}
}

func TestProcessLog(t *testing.T) {
	runner := desktopRunner{}
	runner.errorNotifyCh = make(chan string, 1)

	// Nothing to report
	runner.processLog("")
	assert.Empty(t, runner.errorNotifyCh)
	assert.Nil(t, runner.errorsReported)

	// No errors found
	runner.processLog("line 1\nline 2")
	assert.Empty(t, runner.errorNotifyCh)
	assert.Nil(t, runner.errorsReported)

	// Process log with known error
	runner.processLog("line 1\n" + string(logErrorLaunchServicesSubstr) + "bozo")
	require.Len(t, runner.errorNotifyCh, 1)
	msg := <-runner.errorNotifyCh
	assert.Equal(t, string(logErrorLaunchServicesMsg), msg)

	// Process known error again
	runner.processLog(string(logErrorLaunchServicesSubstr))
	assert.Empty(t, runner.errorNotifyCh)

	// Process another error
	runner.processLog("line 1" + string(logErrorMissingExecSubstr) + "\nbozo")
	require.Len(t, runner.errorNotifyCh, 1)
	msg = <-runner.errorNotifyCh
	assert.Equal(t, string(logErrorMissingExecMsg), msg)
}
