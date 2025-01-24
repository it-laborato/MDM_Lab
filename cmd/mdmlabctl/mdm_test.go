package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mdm/nanomdm/mdm"
	"github.com:it-laborato/MDM_Lab/server/mdm/nanomdm/push"
	"github.com:it-laborato/MDM_Lab/server/mock"
	mdmmock "github.com:it-laborato/MDM_Lab/server/mock/mdm"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com:it-laborato/MDM_Lab/server/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockPusher struct{}

type testhost struct {
	host    *mdmlab.Host
	mdmInfo *mdmlab.HostMDM
}

func (mockPusher) Push(ctx context.Context, ids []string) (map[string]*push.Response, error) {
	m := make(map[string]*push.Response, len(ids))
	for _, id := range ids {
		m[id] = &push.Response{Id: id}
	}
	return m, nil
}

func TestMDMRunCommand(t *testing.T) {
	// define some hosts to use in the tests
	hosts := []testhost{
		{
			host: &mdmlab.Host{
				ID:       1,
				UUID:     "mac-enrolled",
				Platform: "darwin",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
		},
		{
			host: &mdmlab.Host{
				ID:       2,
				UUID:     "win-enrolled",
				Platform: "windows",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
		},
		{
			host: &mdmlab.Host{
				ID:       3,
				UUID:     "mac-unenrolled",
				Platform: "darwin",
			},
		},
		{
			host: &mdmlab.Host{
				ID:       4,
				UUID:     "win-unenrolled",
				Platform: "windows",
			},
		},
		{
			host: &mdmlab.Host{
				ID:       5,
				UUID:     "linux-unenrolled",
				Platform: "linux",
			},
		},
		{
			host: &mdmlab.Host{
				ID:       6,
				UUID:     "mac-enrolled-2",
				Platform: "darwin",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
		},
		{
			host: &mdmlab.Host{
				ID:       7,
				UUID:     "win-enrolled-2",
				Platform: "windows",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
		},
		{
			host: &mdmlab.Host{
				ID:       8,
				UUID:     "mac-non-mdmlab-enrolled",
				Platform: "darwin",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMJamf, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(false)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMJamf},
		},
		{
			host: &mdmlab.Host{
				ID:       9,
				UUID:     "win-non-mdmlab-enrolled",
				Platform: "windows",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMIntune, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(false)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMIntune},
		},
		{
			host: &mdmlab.Host{
				ID:       10,
				UUID:     "mac-pending",
				Platform: "darwin",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
		},
		{
			host: &mdmlab.Host{
				ID:       11,
				UUID:     "win-pending",
				Platform: "windows",
				MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
			},
			mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
		},
	}
	hostByUUID := make(map[string]testhost)
	hostByID := make(map[uint]testhost)
	for _, h := range hosts {
		hostByUUID[h.host.UUID] = h
		hostByID[h.host.ID] = h
	}

	// define some files to use in the tests
	yamlFilePath := writeTmpYml(t, `invalid`)
	mobileConfigFilePath := writeTmpMobileconfig(t, "Mobileconfig")
	appleCmdFilePath := writeTmpAppleMDMCmd(t, "FooBar")
	winCmdFilePath := writeTmpWindowsMDMCmd(t, "FooBar")
	applePremiumCmdFilePath := writeTmpAppleMDMCmd(t, "EraseDevice")
	winPremiumCmdFilePath := writeTmpWindowsMDMCmd(t, "./Device/Vendor/MSFT/RemoteWipe/doWipe")

	emptyAppleCmdFilePath, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = emptyAppleCmdFilePath.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
</plist>`)
	require.NoError(t, err)
	emptyAppleCmdFilePath.Close()

	emptyWinCmdFilePath, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = emptyWinCmdFilePath.WriteString(`<Exec>
</Exec>`)
	require.NoError(t, err)
	emptyWinCmdFilePath.Close()

	nonExecWinCmdFilePath, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = nonExecWinCmdFilePath.WriteString(`<Get>
	<CmdID>22</CmdID>
	<Item>
		<Target>
			<LocURI>FooBar</LocURI>
		</Target>
		<Meta>
			<Format xmlns="syncml:metinf">chr</Format>
			<Type xmlns="syncml:metinf">text/plain</Type>
		</Meta>
		<Data>NamedValuesList=MinPasswordLength,8;</Data>
	</Item>
</Get>`)
	require.NoError(t, err)
	nonExecWinCmdFilePath.Close()

	// define some app configs variations to use in the tests
	appCfgAllMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{EnabledAndConfigured: true, WindowsEnabledAndConfigured: true}}
	appCfgWinMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{WindowsEnabledAndConfigured: true}}
	appCfgMacMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{EnabledAndConfigured: true}}
	appCfgNoMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{}}

	for _, lic := range []string{mdmlab.TierFree, mdmlab.TierPremium} {
		t.Run(lic, func(t *testing.T) {
			enqueuer := new(mdmmock.MDMAppleStore)
			license := &mdmlab.LicenseInfo{Tier: lic, Expiration: time.Now().Add(24 * time.Hour)}

			_, ds := runServerWithMockedDS(t, &service.TestServerOpts{
				MDMStorage:       enqueuer,
				MDMPusher:        mockPusher{},
				License:          license,
				NoCacheDatastore: true,
			})

			ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*mdmlab.Host, error) {
				h, ok := hostByUUID[identifier]
				if !ok {
					return nil, &notFoundError{}
				}
				return h.host, nil
			}
			ds.LoadHostSoftwareFunc = func(ctx context.Context, host *mdmlab.Host, includeCVEScores bool) error {
				return nil
			}
			ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*mdmlab.Label, error) {
				return nil, nil
			}
			ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*mdmlab.Pack, err error) {
				return nil, nil
			}
			ds.ListHostBatteriesFunc = func(ctx context.Context, id uint) ([]*mdmlab.HostBattery, error) {
				return nil, nil
			}
			ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hid uint) ([]*mdmlab.HostMaintenanceWindow, error) {
				return nil, nil
			}
			ds.ListPoliciesForHostFunc = func(ctx context.Context, host *mdmlab.Host) ([]*mdmlab.HostPolicy, error) {
				return nil, nil
			}
			ds.GetHostMDMAppleProfilesFunc = func(ctx context.Context, hostUUID string) ([]mdmlab.HostMDMAppleProfile, error) {
				return nil, nil
			}
			ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hostUUID string) ([]mdmlab.HostMDMWindowsProfile, error) {
				return nil, nil
			}
			ds.GetHostMDMMacOSSetupFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDMMacOSSetup, error) {
				return nil, nil
			}
			ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *mdmlab.Host) (*mdmlab.HostLockWipeStatus, error) {
				return &mdmlab.HostLockWipeStatus{}, nil
			}
			ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, uuids []string) ([]*mdmlab.Host, error) {
				if len(uuids) == 0 {
					return nil, nil
				}
				hosts := make([]*mdmlab.Host, 0, len(uuids))
				for _, uid := range uuids {
					if h := hostByUUID[uid]; h.host != nil {
						hosts = append(hosts, h.host)
					}
				}
				return hosts, nil
			}
			winCmds := map[string]struct{}{}
			ds.MDMWindowsInsertCommandForHostsFunc = func(ctx context.Context, deviceIDs []string, cmd *mdmlab.MDMWindowsCommand) error {
				// every command uuid is different
				require.NotContains(t, winCmds, cmd.CommandUUID)
				winCmds[cmd.CommandUUID] = struct{}{}
				return nil
			}
			ds.GetMDMWindowsBitLockerStatusFunc = func(ctx context.Context, host *mdmlab.Host) (*mdmlab.HostMDMDiskEncryption, error) {
				return &mdmlab.HostMDMDiskEncryption{}, nil
			}
			ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
				h, ok := hostByID[hostID]
				require.True(t, ok)
				if h.mdmInfo == nil {
					return nil, &notFoundError{}
				}
				return h.mdmInfo, nil
			}
			ds.AreHostsConnectedToMDMlabMDMFunc = func(ctx context.Context, hosts []*mdmlab.Host) (map[string]bool, error) {
				res := make(map[string]bool, len(hosts))
				for _, h := range hosts {
					res[h.UUID] = h.MDM.ConnectedToMDMlab != nil && *h.MDM.ConnectedToMDMlab
				}
				return res, nil
			}

			enqueuer.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
				return map[string]error{}, nil
			}

			cases := []struct {
				desc    string
				flags   []string
				appCfg  *mdmlab.AppConfig
				wantErr string
			}{
				{"no flags", nil, appCfgAllMDM, `Required flags "hosts, payload" not set`},
				{"no payload", []string{"--hosts", "abc"}, appCfgAllMDM, `Required flag "payload" not set`},
				{"no hosts", []string{"--payload", winCmdFilePath}, appCfgAllMDM, `Required flag "hosts" not set`},
				{"invalid payload", []string{"--hosts", "abc", "--payload", "no-such-file"}, appCfgAllMDM, `open no-such-file: no such file or directory`},
				{"macOS yaml payload", []string{"--hosts", "mac-enrolled", "--payload", yamlFilePath}, appCfgAllMDM, `The payload isn't valid XML`},
				{"win yaml payload", []string{"--hosts", "win-enrolled", "--payload", yamlFilePath}, appCfgAllMDM, `The payload isn't valid XML`},
				{"non-mdm-command plist payload", []string{"--hosts", "mac-enrolled", "--payload", mobileConfigFilePath}, appCfgAllMDM, `The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file:`},
				{"single host not found", []string{"--hosts", "no-such-host", "--payload", appleCmdFilePath}, appCfgAllMDM, mdmlab.TargetedHostsDontExistErrMsg},
				{"unenrolled macOS host", []string{"--hosts", "mac-unenrolled", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"unenrolled windows host", []string{"--hosts", "win-unenrolled", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"macOS non-mdmlab host", []string{"--hosts", "mac-non-mdmlab-enrolled", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"windows non-mdmlab host", []string{"--hosts", "win-non-mdmlab-enrolled", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"macOS pending host", []string{"--hosts", "mac-pending", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"windows pending host", []string{"--hosts", "win-pending", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"valid single mac", []string{"--hosts", "mac-enrolled", "--payload", appleCmdFilePath}, appCfgAllMDM, ""},
				{"valid single windows", []string{"--hosts", "win-enrolled", "--payload", winCmdFilePath}, appCfgAllMDM, ""},
				{"no mdm enabled", []string{"--hosts", "win-enrolled", "--payload", winCmdFilePath}, appCfgNoMDM, "MDM features aren't turned on in MDMlab."},
				{"valid single mac only win mdm", []string{"--hosts", "mac-enrolled", "--payload", appleCmdFilePath}, appCfgWinMDM, "macOS MDM isn't turned on."},
				{"valid single win only mac mdm", []string{"--hosts", "win-enrolled", "--payload", winCmdFilePath}, appCfgMacMDM, "Windows MDM isn't turned on."},
				{"macOS premium cmd", []string{"--hosts", "mac-enrolled", "--payload", applePremiumCmdFilePath}, appCfgAllMDM, func() string {
					if lic == mdmlab.TierFree {
						return `missing or invalid license`
					}
					return ""
				}()},
				{"windows premium cmd", []string{"--hosts", "win-enrolled", "--payload", winPremiumCmdFilePath}, appCfgAllMDM, func() string {
					if lic == mdmlab.TierFree {
						return `Missing or invalid license. Wipe command is available in MDMlab Premium only.`
					}
					return ""
				}()},
				{"empty plist file", []string{"--hosts", "mac-enrolled", "--payload", emptyAppleCmdFilePath.Name()}, appCfgAllMDM, `The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file.`},
				{"non-Exec win file", []string{"--hosts", "win-enrolled", "--payload", nonExecWinCmdFilePath.Name()}, appCfgAllMDM, `You can run only <Exec> command type.`},
				{"empty win file", []string{"--hosts", "win-enrolled", "--payload", emptyWinCmdFilePath.Name()}, appCfgAllMDM, `You can run only a single <Exec> command.`},
				{"hosts with different platforms", []string{"--hosts", "win-enrolled,mac-enrolled", "--payload", winCmdFilePath}, appCfgAllMDM, `Command can't run on hosts with different platforms.`},
				{"all hosts not found", []string{"--hosts", "no-such-1,no-such-2,no-such-3", "--payload", winCmdFilePath}, appCfgAllMDM, mdmlab.TargetedHostsDontExistErrMsg},
				{"one host not found", []string{"--hosts", "win-enrolled,no-such-2,win-enrolled-2", "--payload", winCmdFilePath}, appCfgAllMDM, `One or more targeted hosts don't exist.`},
				{"one windows host not enrolled", []string{"--hosts", "win-enrolled,win-unenrolled,win-enrolled-2", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"one macOS host not enrolled", []string{"--hosts", "mac-enrolled,mac-unenrolled,mac-enrolled-2", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"valid multiple mac", []string{"--hosts", "mac-enrolled,mac-enrolled-2", "--payload", appleCmdFilePath}, appCfgAllMDM, ""},
				{"valid multiple windows", []string{"--hosts", "win-enrolled,win-enrolled-2", "--payload", winCmdFilePath}, appCfgAllMDM, ""},
				{"valid multiple mac mac-enabled only", []string{"--hosts", "mac-enrolled,mac-enrolled-2", "--payload", appleCmdFilePath}, appCfgMacMDM, ""},
				{"valid multiple windows win-enabled only", []string{"--hosts", "win-enrolled,win-enrolled-2", "--payload", winCmdFilePath}, appCfgWinMDM, ""},
			}
			for _, c := range cases {
				t.Run(c.desc, func(t *testing.T) {
					ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
						return c.appCfg, nil
					}

					buf, err := runAppNoChecks(append([]string{"mdm", "run-command"}, c.flags...))
					if c.wantErr != "" {
						require.Error(t, err)
						require.ErrorContains(t, err, c.wantErr)
					} else {
						require.NoError(t, err)
						require.Contains(t, buf.String(), `Hosts will run the command the next time they check into MDMlab.`)
						require.Contains(t, buf.String(), `mdmlabctl get mdm-command-results --id=`)
					}
				})
			}
		})
	}
}

func TestMDMLockCommand(t *testing.T) {
	macEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       1,
			UUID:     "mac-enrolled",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       2,
			UUID:     "win-enrolled",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}

	linuxEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       3,
			UUID:     "linux-enrolled",
			Platform: "linux",
		},
	}
	winNotEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       4,
			UUID:     "win-not-enrolled",
			Platform: "windows",
		},
	}
	macNotEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       5,
			UUID:     "mac-not-enrolled",
			Platform: "darwin",
		},
	}
	macPending := testhost{
		host: &mdmlab.Host{
			ID:       6,
			UUID:     "mac-pending",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winPending := testhost{
		host: &mdmlab.Host{
			ID:       7,
			UUID:     "win-pending",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledUP := testhost{
		host: &mdmlab.Host{
			ID:       8,
			UUID:     "win-enrolled-up",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledLP := testhost{
		host: &mdmlab.Host{
			ID:       10,
			UUID:     "win-enrolled-lp",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledLP := testhost{
		host: &mdmlab.Host{
			ID:       11,
			UUID:     "mac-enrolled-lp",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledWP := testhost{
		host: &mdmlab.Host{
			ID:       12,
			UUID:     "win-enrolled-wp",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledWP := testhost{
		host: &mdmlab.Host{
			ID:       13,
			UUID:     "mac-enrolled-wp",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}

	hostByUUID := make(map[string]testhost)
	hostsByID := make(map[uint]testhost)
	for _, h := range []testhost{
		winEnrolled,
		macEnrolled,
		linuxEnrolled,
		macNotEnrolled,
		winNotEnrolled,
		macPending,
		winPending,
		winEnrolledUP,
		winEnrolledLP,
		macEnrolledLP,
		winEnrolledWP,
		macEnrolledWP,
	} {
		hostByUUID[h.host.UUID] = h
		hostsByID[h.host.ID] = h
	}

	unlockPending := map[uint]testhost{
		winEnrolledUP.host.ID: winEnrolledUP,
	}

	lockPending := map[uint]testhost{
		winEnrolledLP.host.ID: winEnrolledLP,
		macEnrolledLP.host.ID: macEnrolledLP,
	}

	wipePending := map[uint]testhost{
		winEnrolledWP.host.ID: winEnrolledWP,
		macEnrolledWP.host.ID: macEnrolledWP,
	}

	ds := setupTestServer(t)
	setupDSMocks(ds, hostByUUID, hostsByID)

	// custom ds mocks for these tests
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *mdmlab.Host) (*mdmlab.HostLockWipeStatus, error) {
		mdmlabPlatform := host.MDMlabPlatform()

		var status mdmlab.HostLockWipeStatus
		status.HostMDMlabPlatform = mdmlabPlatform

		if _, ok := unlockPending[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				return nil, errors.New("apple devices do not have an unlock pending state")
			}

			status.UnlockScript = &mdmlab.HostScriptResult{}
		}

		if _, ok := lockPending[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				status.LockMDMCommand = &mdmlab.MDMCommand{}
				return &status, nil
			}

			status.LockScript = &mdmlab.HostScriptResult{}
		}

		if _, ok := wipePending[host.ID]; ok {
			if mdmlabPlatform == "linux" {
				status.WipeScript = &mdmlab.HostScriptResult{ExitCode: nil}
				return &status, nil
			}

			status.WipeMDMCommand = &mdmlab.MDMCommand{}
			status.WipeMDMCommandResult = nil
			return &status, nil
		}

		return &status, nil
	}
	ds.LockHostViaScriptFunc = func(ctx context.Context, request *mdmlab.HostScriptRequestPayload, platform string) error {
		return nil
	}

	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok || h.mdmInfo == nil {
			return nil, &notFoundError{}
		}

		return h.mdmInfo, nil
	}

	ds.GetHostOrbitInfoFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostOrbitInfo, error) {
		hostIDMod := hostID % 3
		switch hostIDMod {
		case 0:
			return nil, &notFoundError{}
		case 1:
			return &mdmlab.HostOrbitInfo{}, nil
		case 2:
			return &mdmlab.HostOrbitInfo{ScriptsEnabled: ptr.Bool(true)}, nil
		default:
			t.Errorf("unexpected hostIDMod %v", hostIDMod)
			return nil, nil
		}
	}

	ds.IsHostConnectedToMDMlabMDMFunc = func(ctx context.Context, host *mdmlab.Host) (bool, error) {
		mdmInfo := hostsByID[host.ID].mdmInfo
		return mdmInfo != nil && mdmInfo.Enrolled == true && mdmInfo.Name == mdmlab.WellKnownMDMMDMlab, nil
	}

	ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hid uint) ([]*mdmlab.HostMaintenanceWindow, error) {
		return nil, nil
	}
	appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM := setupAppConigs()

	successfulOutput := func(ident string) string {
		return fmt.Sprintf(`
The host will lock when it comes online.

Copy and run this command to see lock status:

mdmlabctl get host %s

When you're ready to unlock the host, copy and run this command:

mdmlabctl mdm unlock --host=%s

`, ident, ident)
	}

	cases := []struct {
		appCfg  *mdmlab.AppConfig
		desc    string
		flags   []string
		wantErr string
	}{
		{appCfgAllMDM, "no flags", nil, `Required flag "host" not set`},
		{appCfgAllMDM, "host flag empty", []string{"--host", ""}, `No host targeted. Please provide --host.`},
		{appCfgAllMDM, "lock non-existent host", []string{"--host", "notfound"}, mdmlab.HostNotFoundErrMsg},
		{appCfgMacMDM, "valid windows but only macos mdm", []string{"--host", winEnrolled.host.UUID}, `Windows MDM isn't turned on.`},
		{appCfgWinMDM, "valid macos but only windows mdm", []string{"--host", macEnrolled.host.UUID}, `macOS MDM isn't turned on.`},
		{appCfgAllMDM, "valid windows", []string{"--host", winEnrolled.host.UUID}, ""},
		{appCfgAllMDM, "valid macos", []string{"--host", macEnrolled.host.UUID}, ""},
		{appCfgNoMDM, "valid linux", []string{"--host", linuxEnrolled.host.UUID}, ""},
		{appCfgNoMDM, "valid windows but no mdm", []string{"--host", winEnrolled.host.UUID}, `Windows MDM isn't turned on.`},
		{appCfgNoMDM, "valid macos but no mdm", []string{"--host", macEnrolled.host.UUID}, `macOS MDM isn't turned on.`},
		{appCfgMacMDM, "valid macos but not enrolled", []string{"--host", macNotEnrolled.host.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but not enrolled", []string{"--host", winNotEnrolled.host.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but pending ", []string{"--host", winPending.host.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgMacMDM, "valid macos but pending", []string{"--host", macPending.host.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgAllMDM, "valid windows but pending unlock", []string{"--host", winEnrolledUP.host.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid windows but pending lock", []string{"--host", winEnrolledLP.host.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid macos but pending lock", []string{"--host", macEnrolledLP.host.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid windows but pending wipe", []string{"--host", winEnrolledWP.host.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid macos but pending wipe", []string{"--host", macEnrolledWP.host.UUID}, "Host has pending wipe request."},
	}

	runTestCases(t, ds, "lock", successfulOutput, cases)
}

func TestMDMUnlockCommand(t *testing.T) {
	macEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       1,
			UUID:     "mac-enrolled",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       2,
			UUID:     "win-enrolled",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	linuxEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       3,
			UUID:     "linux-enrolled",
			Platform: "linux",
		},
	}
	winNotEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       4,
			UUID:     "win-not-enrolled",
			Platform: "windows",
		},
	}
	macNotEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       5,
			UUID:     "mac-not-enrolled",
			Platform: "darwin",
		},
	}
	macPending := testhost{
		host: &mdmlab.Host{
			ID:       6,
			UUID:     "mac-pending",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winPending := testhost{
		host: &mdmlab.Host{
			ID:       7,
			UUID:     "win-pending",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledUP := testhost{
		host: &mdmlab.Host{
			ID:       8,
			UUID:     "win-enrolled-up",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledLP := testhost{
		host: &mdmlab.Host{
			ID:       10,
			UUID:     "win-enrolled-lp",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledLP := testhost{
		host: &mdmlab.Host{
			ID:       11,
			UUID:     "mac-enrolled-lp",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledWP := testhost{
		host: &mdmlab.Host{
			ID:       12,
			UUID:     "win-enrolled-wp",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledWP := testhost{
		host: &mdmlab.Host{
			ID:       13,
			UUID:     "mac-enrolled-wp",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}

	hostByUUID := make(map[string]testhost)
	hostsByID := make(map[uint]testhost)
	for _, h := range []testhost{
		winEnrolled,
		macEnrolled,
		linuxEnrolled,
		macNotEnrolled,
		winNotEnrolled,
		macPending,
		winPending,
		winEnrolledUP,
		winEnrolledLP,
		macEnrolledLP,
		winEnrolledWP,
		macEnrolledWP,
	} {
		hostByUUID[h.host.UUID] = h
		hostsByID[h.host.ID] = h
	}

	locked := map[uint]testhost{
		winEnrolled.host.ID:   winEnrolled,
		macEnrolled.host.ID:   macEnrolled,
		linuxEnrolled.host.ID: linuxEnrolled,
	}

	unlockPending := map[uint]testhost{
		winEnrolledUP.host.ID: winEnrolledUP,
	}

	lockPending := map[uint]testhost{
		winEnrolledLP.host.ID: winEnrolledLP,
		macEnrolledLP.host.ID: macEnrolledLP,
	}

	wipePending := map[uint]testhost{
		winEnrolledWP.host.ID: winEnrolledWP,
		macEnrolledWP.host.ID: macEnrolledWP,
	}

	ds := setupTestServer(t)
	setupDSMocks(ds, hostByUUID, hostsByID)

	// custom mocks for these test
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *mdmlab.Host) (*mdmlab.HostLockWipeStatus, error) {
		mdmlabPlatform := host.MDMlabPlatform()

		var status mdmlab.HostLockWipeStatus
		status.HostMDMlabPlatform = mdmlabPlatform
		if _, ok := locked[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				status.LockMDMCommand = &mdmlab.MDMCommand{}
				status.LockMDMCommandResult = &mdmlab.MDMCommandResult{Status: mdmlab.MDMAppleStatusAcknowledged}
				return &status, nil
			}

			status.LockScript = &mdmlab.HostScriptResult{ExitCode: ptr.Int64(0)}
		}

		if _, ok := unlockPending[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				return nil, errors.New("apple devices do not have an unlock pending state")
			}

			status.UnlockScript = &mdmlab.HostScriptResult{}
		}

		if _, ok := lockPending[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				status.LockMDMCommand = &mdmlab.MDMCommand{}
				return &status, nil
			}

			status.LockScript = &mdmlab.HostScriptResult{}
		}

		if _, ok := wipePending[host.ID]; ok {
			if mdmlabPlatform == "linux" {
				status.WipeScript = &mdmlab.HostScriptResult{ExitCode: nil}
				return &status, nil
			}

			status.WipeMDMCommand = &mdmlab.MDMCommand{}
			status.WipeMDMCommandResult = nil
			return &status, nil
		}

		return &status, nil
	}
	ds.UnlockHostViaScriptFunc = func(ctx context.Context, request *mdmlab.HostScriptRequestPayload, platform string) error {
		return nil
	}
	ds.UnlockHostManuallyFunc = func(ctx context.Context, hostID uint, platform string, ts time.Time) error {
		return nil
	}

	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok || h.mdmInfo == nil {
			return nil, &notFoundError{}
		}

		return h.mdmInfo, nil
	}

	ds.GetHostOrbitInfoFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostOrbitInfo, error) {
		hostIDMod := hostID % 3
		switch hostIDMod {
		case 0:
			return nil, &notFoundError{}
		case 1:
			return &mdmlab.HostOrbitInfo{}, nil
		case 2:
			return &mdmlab.HostOrbitInfo{ScriptsEnabled: ptr.Bool(true)}, nil
		default:
			t.Errorf("unexpected hostIDMod %v", hostIDMod)
			return nil, nil
		}
	}
	ds.IsHostConnectedToMDMlabMDMFunc = func(ctx context.Context, host *mdmlab.Host) (bool, error) {
		return host.MDM.ConnectedToMDMlab != nil && *host.MDM.ConnectedToMDMlab, nil
	}
	ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hid uint) ([]*mdmlab.HostMaintenanceWindow, error) {
		return nil, nil
	}
	appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM := setupAppConigs()

	successfulOutput := func(ident string) string {
		h := hostByUUID[ident]
		if h.host.Platform == "darwin" {
			return `Use this 6 digit PIN to unlock the host:`
		}
		return fmt.Sprintf(`
The host will unlock when it comes online.

Copy and run this command to see results:

mdmlabctl get host %s

`, ident)
	}

	cases := []struct {
		appCfg  *mdmlab.AppConfig
		desc    string
		flags   []string
		wantErr string
	}{
		{appCfgAllMDM, "no flags", nil, `Required flag "host" not set`},
		{appCfgAllMDM, "host flag empty", []string{"--host", ""}, `No host targeted. Please provide --host.`},
		{appCfgAllMDM, "unlock non-existent host", []string{"--host", "notfound"}, mdmlab.HostNotFoundErrMsg},
		{appCfgMacMDM, "valid windows but only macos mdm", []string{"--host", winEnrolled.host.UUID}, `Windows MDM isn't turned on.`},
		{appCfgAllMDM, "valid windows", []string{"--host", winEnrolled.host.UUID}, ""},
		{appCfgAllMDM, "valid macos", []string{"--host", macEnrolled.host.UUID}, ""},
		{appCfgNoMDM, "valid linux", []string{"--host", linuxEnrolled.host.UUID}, ""},
		{appCfgNoMDM, "valid windows but no mdm", []string{"--host", winEnrolled.host.UUID}, `Windows MDM isn't turned on.`},
		// TODO: should we error here?
		// {appCfgNoMDM, "valid macos but no mdm", []string{"--host", macEnrolled.host.UUID}, `macOS MDM isn't turned on.`},
		{appCfgMacMDM, "valid macos but not enrolled", []string{"--host", macNotEnrolled.host.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but not enrolled", []string{"--host", winNotEnrolled.host.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but pending mdm enroll", []string{"--host", winPending.host.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgMacMDM, "valid macos but pending mdm enroll", []string{"--host", macPending.host.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgAllMDM, "valid windows but pending unlock", []string{"--host", winEnrolledUP.host.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid windows but pending lock", []string{"--host", winEnrolledLP.host.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid macos but pending lock", []string{"--host", macEnrolledLP.host.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid windows but pending wipe", []string{"--host", winEnrolledWP.host.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid macos but pending wipe", []string{"--host", macEnrolledWP.host.UUID}, "Host has pending wipe request."},
	}

	runTestCases(t, ds, "unlock", successfulOutput, cases)
}

func TestMDMWipeCommand(t *testing.T) {
	macEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       1,
			UUID:     "mac-enrolled",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       2,
			UUID:     "win-enrolled",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winNotEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       4,
			UUID:     "win-not-enrolled",
			Platform: "windows",
		},
	}
	macNotEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       5,
			UUID:     "mac-not-enrolled",
			Platform: "darwin",
		},
	}
	macPending := testhost{
		host: &mdmlab.Host{
			ID:       6,
			UUID:     "mac-pending",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winPending := testhost{
		host: &mdmlab.Host{
			ID:       7,
			UUID:     "win-pending",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("Pending"), ConnectedToMDMlab: ptr.Bool(false)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: false, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledUP := testhost{
		host: &mdmlab.Host{
			ID:       8,
			UUID:     "win-enrolled-up",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledLP := testhost{
		host: &mdmlab.Host{
			ID:       10,
			UUID:     "win-enrolled-lp",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledLP := testhost{
		host: &mdmlab.Host{
			ID:       11,
			UUID:     "mac-enrolled-lp",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledWP := testhost{
		host: &mdmlab.Host{
			ID:       12,
			UUID:     "win-enrolled-wp",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledWP := testhost{
		host: &mdmlab.Host{
			ID:       13,
			UUID:     "mac-enrolled-wp",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledWiped := testhost{
		host: &mdmlab.Host{
			ID:       14,
			UUID:     "win-enrolled-wiped",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledWiped := testhost{
		host: &mdmlab.Host{
			ID:       15,
			UUID:     "mac-enrolled-wiped",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual)"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	winEnrolledLocked := testhost{
		host: &mdmlab.Host{
			ID:       16,
			UUID:     "win-enrolled-locked",
			Platform: "windows",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	macEnrolledLocked := testhost{
		host: &mdmlab.Host{
			ID:       17,
			UUID:     "mac-enrolled-locked",
			Platform: "darwin",
			MDM:      mdmlab.MDMHostData{Name: mdmlab.WellKnownMDMMDMlab, EnrollmentStatus: ptr.String("On (manual"), ConnectedToMDMlab: ptr.Bool(true)},
		},
		mdmInfo: &mdmlab.HostMDM{Enrolled: true, Name: mdmlab.WellKnownMDMMDMlab},
	}
	linuxEnrolled := testhost{
		host: &mdmlab.Host{
			ID:       18,
			UUID:     "linux-enrolled",
			Platform: "linux",
		},
	}
	linuxEnrolled2 := testhost{
		host: &mdmlab.Host{
			ID:       19,
			UUID:     "linux-enrolled",
			Platform: "linux",
		},
	}
	linuxEnrolled3 := testhost{
		host: &mdmlab.Host{
			ID:       20,
			UUID:     "linux-enrolled",
			Platform: "linux",
		},
	}

	linuxHostIDs := []uint{linuxEnrolled.host.ID, linuxEnrolled2.host.ID, linuxEnrolled3.host.ID}

	hostByUUID := make(map[string]testhost)
	hostsByID := make(map[uint]testhost)
	for _, h := range []testhost{
		winEnrolled,
		macEnrolled,
		linuxEnrolled,
		linuxEnrolled2,
		linuxEnrolled3,
		macNotEnrolled,
		winNotEnrolled,
		macPending,
		winPending,
		winEnrolledUP,
		winEnrolledLP,
		macEnrolledLP,
		winEnrolledWP,
		macEnrolledWP,
		winEnrolledWiped,
		macEnrolledWiped,
		winEnrolledLocked,
		macEnrolledLocked,
	} {
		hostByUUID[h.host.UUID] = h
		hostsByID[h.host.ID] = h
	}

	locked := map[uint]testhost{
		winEnrolledLocked.host.ID: winEnrolledLocked,
		macEnrolledLocked.host.ID: macEnrolledLocked,
	}

	unlockPending := map[uint]testhost{
		winEnrolledUP.host.ID: winEnrolledUP,
	}

	lockPending := map[uint]testhost{
		winEnrolledLP.host.ID: winEnrolledLP,
		macEnrolledLP.host.ID: macEnrolledLP,
	}

	wipePending := map[uint]testhost{
		winEnrolledWP.host.ID: winEnrolledWP,
		macEnrolledWP.host.ID: macEnrolledWP,
	}

	wiped := map[uint]testhost{
		winEnrolledWiped.host.ID: winEnrolledWiped,
		macEnrolledWiped.host.ID: macEnrolledWiped,
	}

	ds := setupTestServer(t)
	setupDSMocks(ds, hostByUUID, hostsByID)

	// TODO: custom ds mocks for these tests
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *mdmlab.Host) (*mdmlab.HostLockWipeStatus, error) {
		mdmlabPlatform := host.MDMlabPlatform()

		var status mdmlab.HostLockWipeStatus
		status.HostMDMlabPlatform = mdmlabPlatform
		if _, ok := locked[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				status.LockMDMCommand = &mdmlab.MDMCommand{}
				status.LockMDMCommandResult = &mdmlab.MDMCommandResult{Status: mdmlab.MDMAppleStatusAcknowledged}
				return &status, nil
			}

			status.LockScript = &mdmlab.HostScriptResult{ExitCode: ptr.Int64(0)}
		}

		if _, ok := unlockPending[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				return nil, errors.New("apple devices do not have an unlock pending state")
			}

			status.UnlockScript = &mdmlab.HostScriptResult{}
		}

		if _, ok := lockPending[host.ID]; ok {
			if mdmlabPlatform == "darwin" {
				status.LockMDMCommand = &mdmlab.MDMCommand{}
				return &status, nil
			}

			status.LockScript = &mdmlab.HostScriptResult{}
		}

		if _, ok := wipePending[host.ID]; ok {
			if mdmlabPlatform == "linux" {
				status.WipeScript = &mdmlab.HostScriptResult{ExitCode: nil}
				return &status, nil
			}

			status.WipeMDMCommand = &mdmlab.MDMCommand{}
			status.WipeMDMCommandResult = nil
			return &status, nil
		}

		if _, ok := wiped[host.ID]; ok {
			if mdmlabPlatform == "linux" {
				status.WipeScript = &mdmlab.HostScriptResult{ExitCode: ptr.Int64(0)}
			}

			if mdmlabPlatform == "darwin" {
				status.WipeMDMCommand = &mdmlab.MDMCommand{}
				status.WipeMDMCommandResult = &mdmlab.MDMCommandResult{
					Status: mdmlab.MDMAppleStatusAcknowledged,
				}
			}

			if mdmlabPlatform == "windows" {
				status.WipeMDMCommand = &mdmlab.MDMCommand{}
				status.WipeMDMCommandResult = &mdmlab.MDMCommandResult{
					Status: "200",
				}
			}

			return &status, nil
		}

		return &status, nil
	}
	ds.UnlockHostViaScriptFunc = func(ctx context.Context, request *mdmlab.HostScriptRequestPayload, hostMDMlabPlatform string) error {
		return nil
	}
	ds.UnlockHostManuallyFunc = func(ctx context.Context, hostID uint, hostMDMlabPlatform string, ts time.Time) error {
		return nil
	}
	ds.WipeHostViaWindowsMDMFunc = func(ctx context.Context, host *mdmlab.Host, cmd *mdmlab.MDMWindowsCommand) error {
		return nil
	}

	ds.WipeHostViaScriptFunc = func(ctx context.Context, request *mdmlab.HostScriptRequestPayload, hostMDMlabPlatform string) error {
		return nil
	}

	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok || h.mdmInfo == nil {
			return nil, &notFoundError{}
		}

		return h.mdmInfo, nil
	}
	ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hid uint) ([]*mdmlab.HostMaintenanceWindow, error) {
		return nil, nil
	}

	// This function should only run on linux
	ds.GetHostOrbitInfoFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostOrbitInfo, error) {
		if !slices.Contains(linuxHostIDs, hostID) {
			t.Errorf("GetHostOrbitInfo should not be called for non-linux host %v", hostID)
			return nil, nil
		}
		hostIDMod := hostID % 3
		switch hostIDMod {
		case 0:
			return nil, &notFoundError{}
		case 1:
			return &mdmlab.HostOrbitInfo{}, nil
		case 2:
			return &mdmlab.HostOrbitInfo{ScriptsEnabled: ptr.Bool(true)}, nil
		default:
			t.Errorf("unexpected hostIDMod %v", hostIDMod)
			return nil, nil
		}
	}
	ds.IsHostConnectedToMDMlabMDMFunc = func(ctx context.Context, host *mdmlab.Host) (bool, error) {
		return host.MDM.ConnectedToMDMlab != nil && *host.MDM.ConnectedToMDMlab, nil
	}

	appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM := setupAppConigs()
	appCfgScriptsDisabled := &mdmlab.AppConfig{ServerSettings: mdmlab.ServerSettings{ScriptsDisabled: true}}

	cases := []struct {
		appCfg  *mdmlab.AppConfig
		desc    string
		flags   []string
		wantErr string
	}{
		{appCfgAllMDM, "no flags", nil, `Required flag "host" not set`},
		{appCfgAllMDM, "host flag empty", []string{"--host", ""}, `No host targeted. Please provide --host.`},
		{appCfgAllMDM, "wipe non-existent host", []string{"--host", "notfound"}, mdmlab.HostNotFoundErrMsg},
		{appCfgMacMDM, "valid windows but only macos mdm", []string{"--host", winEnrolled.host.UUID}, `Windows MDM isn't turned on.`},
		{appCfgAllMDM, "valid windows", []string{"--host", winEnrolled.host.UUID}, ""},
		{appCfgAllMDM, "valid macos", []string{"--host", macEnrolled.host.UUID}, ""},
		{appCfgNoMDM, "valid linux", []string{"--host", linuxEnrolled.host.UUID}, ""},
		{appCfgNoMDM, "valid linux 2", []string{"--host", linuxEnrolled2.host.UUID}, ""},
		{appCfgNoMDM, "valid linux 3", []string{"--host", linuxEnrolled3.host.UUID}, ""},
		{appCfgNoMDM, "valid windows but no mdm", []string{"--host", winEnrolled.host.UUID}, `Windows MDM isn't turned on.`},
		{appCfgMacMDM, "valid macos but not enrolled", []string{"--host", macNotEnrolled.host.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but not enrolled", []string{"--host", winNotEnrolled.host.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but pending mdm enroll", []string{"--host", winPending.host.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgMacMDM, "valid macos but pending mdm enroll", []string{"--host", macPending.host.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgAllMDM, "valid windows but pending unlock", []string{"--host", winEnrolledUP.host.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid windows but pending lock", []string{"--host", winEnrolledLP.host.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid macos but pending lock", []string{"--host", macEnrolledLP.host.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid windows but pending wipe", []string{"--host", winEnrolledWP.host.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid macos but pending wipe", []string{"--host", macEnrolledWP.host.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid windows but host wiped", []string{"--host", winEnrolledWiped.host.UUID}, "Host is already wiped."},
		{appCfgAllMDM, "valid macos but host wiped", []string{"--host", macEnrolledWiped.host.UUID}, "Host is already wiped."},
		{appCfgAllMDM, "valid windows but host is locked", []string{"--host", winEnrolledLocked.host.UUID}, "Host cannot be wiped until it is unlocked."},
		{appCfgAllMDM, "valid macos but host is locked", []string{"--host", macEnrolledLocked.host.UUID}, "Host cannot be wiped until it is unlocked."},
		{appCfgAllMDM, "valid macos but host is locked", []string{"--host", macEnrolledLocked.host.UUID}, "Host cannot be wiped until it is unlocked."},
		{appCfgScriptsDisabled, "valid linux and scripts are disabled", []string{"--host", linuxEnrolled.host.UUID}, ""},
	}

	successfulOutput := func(ident string) string {
		return fmt.Sprintf(`
The host will wipe when it comes online.

Copy and run this command to see results:

mdmlabctl get host %s`, ident)
	}

	runTestCases(t, ds, "wipe", successfulOutput, cases)
}

func writeTmpAppleMDMCmd(t *testing.T, commandName string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
      <key>RequestType</key>
      <string>%s</string>
    </dict>
  </dict>
</plist>`, uuid.New().String(), commandName))
	require.NoError(t, err)
	return tmpFile.Name()
}

func writeTmpWindowsMDMCmd(t *testing.T, commandName string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(fmt.Sprintf(`<Exec>
	<CmdID>11</CmdID>
	<Item>
		<Target>
			<LocURI>%s</LocURI>
		</Target>
		<Meta>
			<Format xmlns="syncml:metinf">chr</Format>
			<Type xmlns="syncml:metinf">text/plain</Type>
		</Meta>
		<Data>NamedValuesList=MinPasswordLength,8;</Data>
	</Item>
</Exec>`, commandName))
	require.NoError(t, err)
	return tmpFile.Name()
}

func writeTmpMobileconfig(t *testing.T, name string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.mobileconfig")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(string(mobileconfigForTest(name, uuid.New().String())))
	require.NoError(t, err)
	return tmpFile.Name()
}

// sets up the test server with the mock datastore and returns the mock datastore
func setupTestServer(t *testing.T) *mock.Store {
	enqueuer := new(mdmmock.MDMAppleStore)
	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	enqueuer.EnqueueDeviceLockCommandFunc = func(ctx context.Context, host *mdmlab.Host, cmd *mdm.Command, pin string) error {
		return nil
	}

	enqueuer.EnqueueDeviceWipeCommandFunc = func(ctx context.Context, host *mdmlab.Host, cmd *mdm.Command) error {
		return nil
	}

	_, ds := runServerWithMockedDS(t, &service.TestServerOpts{
		MDMStorage:       enqueuer,
		MDMPusher:        mockPusher{},
		License:          license,
		NoCacheDatastore: true,
	})

	return ds
}

// sets up common data store mocks that are needed for the tests.
func setupDSMocks(ds *mock.Store, hostByUUID map[string]testhost, hostsByID map[uint]testhost) {
	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*mdmlab.Host, error) {
		h, ok := hostByUUID[identifier]
		if !ok {
			return nil, &notFoundError{}
		}
		return h.host, nil
	}
	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *mdmlab.Host, includeCVEScores bool) error {
		return nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*mdmlab.Pack, err error) {
		return nil, nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, id uint) ([]*mdmlab.HostBattery, error) {
		return nil, nil
	}
	ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hid uint) ([]*mdmlab.HostMaintenanceWindow, error) {
		return nil, nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *mdmlab.Host) ([]*mdmlab.HostPolicy, error) {
		return nil, nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*mdmlab.Label, error) {
		return nil, nil
	}
	ds.GetHostMDMAppleProfilesFunc = func(ctx context.Context, hostUUID string) ([]mdmlab.HostMDMAppleProfile, error) {
		return nil, nil
	}
	ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hostUUID string) ([]mdmlab.HostMDMWindowsProfile, error) {
		return nil, nil
	}
	ds.GetHostMDMMacOSSetupFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDMMacOSSetup, error) {
		return nil, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*mdmlab.Host, error) {
		h, ok := hostsByID[hostID]
		if !ok {
			return nil, &notFoundError{}
		}

		return h.host, nil
	}
	ds.GetMDMWindowsBitLockerStatusFunc = func(ctx context.Context, host *mdmlab.Host) (*mdmlab.HostMDMDiskEncryption, error) {
		return nil, nil
	}
	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok {
			return nil, &notFoundError{}
		}

		return h.mdmInfo, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
}

// sets up the various app configs for the tests. These app configs reflect the various
// states of the MDM configuration.
func setupAppConigs() (*mdmlab.AppConfig, *mdmlab.AppConfig, *mdmlab.AppConfig, *mdmlab.AppConfig) {
	appCfgAllMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{EnabledAndConfigured: true, WindowsEnabledAndConfigured: true}}
	appCfgWinMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{WindowsEnabledAndConfigured: true}}
	appCfgMacMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{EnabledAndConfigured: true}}
	appCfgNoMDM := &mdmlab.AppConfig{MDM: mdmlab.MDM{}}

	return appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM
}

func runTestCases(t *testing.T, ds *mock.Store, actionType string, successfulOutput func(ident string) string, cases []struct {
	appCfg  *mdmlab.AppConfig
	desc    string
	flags   []string
	wantErr string
},
) {
	for _, c := range cases {
		ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
			return c.appCfg, nil
		}
		buf, err := runAppNoChecks(append([]string{"mdm", actionType}, c.flags...))
		if c.wantErr != "" {
			require.Error(t, err, c.desc)
			require.ErrorContains(t, err, c.wantErr, c.desc)
		} else {
			require.NoError(t, err, c.desc)
			require.Contains(t, buf.String(), successfulOutput(c.flags[1]), c.desc)
		}
	}
}
