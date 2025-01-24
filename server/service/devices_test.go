package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/pkg/optjson"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com:it-laborato/MDM_Lab/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMDMlabDesktopSummary(t *testing.T) {
	t.Run("free implementation", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		sum, err := svc.GetMDMlabDesktopSummary(ctx)
		require.ErrorIs(t, err, mdmlab.ErrMissingLicense)
		require.Empty(t, sum)
	})

	t.Run("different app config values for managed host", func(t *testing.T) {
		ds := new(mock.Store)
		license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		ds.FailingPoliciesCountFunc = func(ctx context.Context, host *mdmlab.Host) (uint, error) {
			return uint(1), nil
		}
		const expectedPlatform = "darwin"
		ds.HasSelfServiceSoftwareInstallersFunc = func(ctx context.Context, platform string, teamID *uint) (bool, error) {
			assert.Equal(t, expectedPlatform, platform)
			return true, nil
		}

		cases := []struct {
			mdm         mdmlab.MDM
			depAssigned bool
			out         mdmlab.DesktopNotifications
		}{
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: true,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: true,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      true,
					RenewEnrollmentProfile: false,
				},
			},
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: true,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: true,
					},
				},
				depAssigned: false,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: false,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: true,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: true,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: false,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: false,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: false,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
		}

		for _, c := range cases {
			c := c
			ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
				appCfg := mdmlab.AppConfig{}
				appCfg.MDM = c.mdm
				return &appCfg, nil
			}

			ds.IsHostConnectedToMDMlabMDMFunc = func(ctx context.Context, host *mdmlab.Host) (bool, error) {
				return false, nil
			}
			ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
				return &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMIntune,
					DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseSuccess)),
				}, nil
			}

			ctx := test.HostContext(ctx, &mdmlab.Host{
				OsqueryHostID:      ptr.String("test"),
				DEPAssignedToMDMlab: &c.depAssigned,
				Platform:           expectedPlatform,
			})
			sum, err := svc.GetMDMlabDesktopSummary(ctx)
			require.NoError(t, err)
			require.Equal(t, c.out, sum.Notifications, fmt.Sprintf("enabled_and_configured: %t | macos_migration.enable: %t", c.mdm.EnabledAndConfigured, c.mdm.MacOSMigration.Enable))
			require.EqualValues(t, 1, *sum.FailingPolicies)
			assert.Equal(t, ptr.Bool(true), sum.SelfService)
		}

	})

	t.Run("different app config values for unmanaged host", func(t *testing.T) {
		ds := new(mock.Store)
		license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		ds.FailingPoliciesCountFunc = func(ctx context.Context, host *mdmlab.Host) (uint, error) {
			return uint(1), nil
		}
		ds.HasSelfServiceSoftwareInstallersFunc = func(ctx context.Context, platform string, teamID *uint) (bool, error) {
			return true, nil
		}
		cases := []struct {
			mdm         mdmlab.MDM
			depAssigned bool
			out         mdmlab.DesktopNotifications
		}{
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: true,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: true,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: true,
				},
			},
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: false,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: true,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: true,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: false,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				mdm: mdmlab.MDM{
					EnabledAndConfigured: false,
					MacOSMigration: mdmlab.MacOSMigration{
						Enable: false,
					},
				},
				depAssigned: true,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
		}

		mdmInfo := &mdmlab.HostMDM{
			IsServer:               false,
			InstalledFromDep:       true,
			Enrolled:               false,
			Name:                   mdmlab.WellKnownMDMMDMlab,
			DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseSuccess)),
		}

		for _, c := range cases {
			c := c
			ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
				appCfg := mdmlab.AppConfig{}
				appCfg.MDM = c.mdm
				return &appCfg, nil
			}
			ds.IsHostConnectedToMDMlabMDMFunc = func(ctx context.Context, host *mdmlab.Host) (bool, error) {
				return false, nil
			}

			ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
				return mdmInfo, nil
			}

			ctx = test.HostContext(ctx, &mdmlab.Host{
				OsqueryHostID:      ptr.String("test"),
				DEPAssignedToMDMlab: &c.depAssigned,
			})
			sum, err := svc.GetMDMlabDesktopSummary(ctx)
			require.NoError(t, err)
			require.Equal(t, c.out, sum.Notifications, fmt.Sprintf("enabled_and_configured: %t | macos_migration.enable: %t", c.mdm.EnabledAndConfigured, c.mdm.MacOSMigration.Enable))
			require.EqualValues(t, 1, *sum.FailingPolicies)
		}

	})

	t.Run("different host attributes", func(t *testing.T) {
		ds := new(mock.Store)
		license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

		// context without a host
		sum, err := svc.GetMDMlabDesktopSummary(ctx)
		require.Empty(t, sum)
		var authErr *mdmlab.AuthRequiredError
		require.ErrorAs(t, err, &authErr)

		ds.FailingPoliciesCountFunc = func(ctx context.Context, host *mdmlab.Host) (uint, error) {
			return uint(1), nil
		}

		ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
			appCfg := mdmlab.AppConfig{}
			appCfg.MDM.EnabledAndConfigured = true
			appCfg.MDM.MacOSMigration.Enable = true
			return &appCfg, nil
		}

		ds.HasSelfServiceSoftwareInstallersFunc = func(ctx context.Context, platform string, teamID *uint) (bool, error) {
			return false, nil
		}

		cases := []struct {
			name    string
			host    *mdmlab.Host
			hostMDM *mdmlab.HostMDM
			err     error
			out     mdmlab.DesktopNotifications
		}{
			{
				name: "not enrolled into osquery",
				host: &mdmlab.Host{OsqueryHostID: nil},
				err:  nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				name: "manually enrolled into another MDM",
				host: &mdmlab.Host{
					OsqueryHostID:      ptr.String("test"),
					DEPAssignedToMDMlab: ptr.Bool(false),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       false,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMIntune,
					DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseSuccess)),
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				name: "DEP capable, but already unenrolled",
				host: &mdmlab.Host{
					DEPAssignedToMDMlab: ptr.Bool(true),
					OsqueryHostID:      ptr.String("test"),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               false,
					Name:                   mdmlab.WellKnownMDMMDMlab,
					DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseSuccess)),
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: true,
				},
			},
			{
				name: "DEP capable, but enrolled into MDMlab",
				host: &mdmlab.Host{
					DEPAssignedToMDMlab: ptr.Bool(true),
					OsqueryHostID:      ptr.String("test"),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMMDMlab,
					DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseSuccess)),
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				name: "failed ADE assignment status",
				host: &mdmlab.Host{
					DEPAssignedToMDMlab: ptr.Bool(true),
					OsqueryHostID:      ptr.String("test"),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMIntune,
					DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseFailed)),
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				name: "not accessible ADE assignment status",
				host: &mdmlab.Host{
					DEPAssignedToMDMlab: ptr.Bool(true),
					OsqueryHostID:      ptr.String("test"),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMIntune,
					DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseNotAccessible)),
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				name: "empty ADE assignment status",
				host: &mdmlab.Host{
					DEPAssignedToMDMlab: ptr.Bool(true),
					OsqueryHostID:      ptr.String("test"),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMIntune,
					DEPProfileAssignStatus: ptr.String(""),
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				name: "nil ADE assignment status",
				host: &mdmlab.Host{
					DEPAssignedToMDMlab: ptr.Bool(true),
					OsqueryHostID:      ptr.String("test"),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMIntune,
					DEPProfileAssignStatus: nil,
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      false,
					RenewEnrollmentProfile: false,
				},
			},
			{
				name: "all conditions met",
				host: &mdmlab.Host{
					DEPAssignedToMDMlab: ptr.Bool(true),
					OsqueryHostID:      ptr.String("test"),
				},
				hostMDM: &mdmlab.HostMDM{
					IsServer:               false,
					InstalledFromDep:       true,
					Enrolled:               true,
					Name:                   mdmlab.WellKnownMDMIntune,
					DEPProfileAssignStatus: ptr.String(string(mdmlab.DEPAssignProfileResponseSuccess)),
				},
				err: nil,
				out: mdmlab.DesktopNotifications{
					NeedsMDMMigration:      true,
					RenewEnrollmentProfile: false,
				},
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				ctx = test.HostContext(ctx, c.host)

				ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostMDM, error) {
					if c.hostMDM == nil {
						return nil, sql.ErrNoRows
					}
					return c.hostMDM, nil
				}

				ds.IsHostConnectedToMDMlabMDMFunc = func(ctx context.Context, host *mdmlab.Host) (bool, error) {
					return c.hostMDM != nil && c.hostMDM.Enrolled == true && c.hostMDM.Name == mdmlab.WellKnownMDMMDMlab, nil
				}

				sum, err := svc.GetMDMlabDesktopSummary(ctx)

				if c.err != nil {
					require.ErrorIs(t, err, c.err)
					require.Empty(t, sum)
				} else {

					require.NoError(t, err)
					require.Equal(t, c.out, sum.Notifications)
					require.EqualValues(t, 1, *sum.FailingPolicies)
				}
			})
		}

	})
}

func TestTriggerLinuxDiskEncryptionEscrow(t *testing.T) {
	t.Run("unavailable in MDMlab Free", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
		err := svc.TriggerLinuxDiskEncryptionEscrow(ctx, &mdmlab.Host{ID: 1})
		require.ErrorIs(t, err, mdmlab.ErrMissingLicense)
	})

	t.Run("no-op on already pending", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}, SkipCreateTestUsers: true})
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return true
		}

		err := svc.TriggerLinuxDiskEncryptionEscrow(ctx, &mdmlab.Host{ID: 1})
		require.NoError(t, err)
		require.True(t, ds.IsHostPendingEscrowFuncInvoked)
	})

	t.Run("validation failures", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}, SkipCreateTestUsers: true})
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}
		var reportedErrors []string
		host := &mdmlab.Host{ID: 1, Platform: "rhel", OSVersion: "Red Hat Enterprise Linux 9.0.0"}
		ds.ReportEscrowErrorFunc = func(ctx context.Context, hostID uint, err string) error {
			require.Equal(t, hostID, host.ID)
			reportedErrors = append(reportedErrors, err)
			return nil
		}

		// invalid platform
		err := svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.ErrorContains(t, err, "MDMlab does not yet support creating LUKS disk encryption keys on this platform.")
		require.True(t, ds.IsHostPendingEscrowFuncInvoked)

		// valid platform, no-team, encryption not enabled
		host.OSVersion = "Fedora 32.0.0"
		appConfig := &mdmlab.AppConfig{MDM: mdmlab.MDM{EnableDiskEncryption: optjson.SetBool(false)}}
		ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
			return appConfig, nil
		}
		err = svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.ErrorContains(t, err, "Disk encryption is not enabled for hosts not assigned to a team.")

		// valid platform, team, encryption not enabled
		host.TeamID = ptr.Uint(1)
		teamConfig := &mdmlab.TeamMDM{}
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*mdmlab.TeamMDM, error) {
			require.Equal(t, uint(1), teamID)
			return teamConfig, nil
		}
		err = svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.ErrorContains(t, err, "Disk encryption is not enabled for this host's team.")

		// valid platform, team, host disk is not encrypted or unknown encryption state
		teamConfig = &mdmlab.TeamMDM{EnableDiskEncryption: true}
		err = svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.ErrorContains(t, err, "Host's disk is not encrypted. Please encrypt your disk first.")
		host.DiskEncryptionEnabled = ptr.Bool(false)
		err = svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.ErrorContains(t, err, "Host's disk is not encrypted. Please encrypt your disk first.")

		// No MDMlab Desktop
		host.DiskEncryptionEnabled = ptr.Bool(true)
		orbitInfo := &mdmlab.HostOrbitInfo{Version: "1.35.1"}
		ds.GetHostOrbitInfoFunc = func(ctx context.Context, id uint) (*mdmlab.HostOrbitInfo, error) {
			return orbitInfo, nil
		}
		err = svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.ErrorContains(t, err, "Your version of mdmlabd does not support creating disk encryption keys on Linux. Please upgrade mdmlabd, then click Refetch, then try again.")

		// Encryption key is already escrowed
		orbitInfo.Version = mdmlab.MinOrbitLUKSVersion
		ds.AssertHasNoEncryptionKeyStoredFunc = func(ctx context.Context, hostID uint) error {
			return errors.New("encryption key is already escrowed")
		}
		err = svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.ErrorContains(t, err, "encryption key is already escrowed")

		require.Len(t, reportedErrors, 7)
	})

	t.Run("validation success", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}, SkipCreateTestUsers: true})
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}
		ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
			return &mdmlab.AppConfig{MDM: mdmlab.MDM{EnableDiskEncryption: optjson.SetBool(true)}}, nil
		}
		ds.GetHostOrbitInfoFunc = func(ctx context.Context, id uint) (*mdmlab.HostOrbitInfo, error) {
			return &mdmlab.HostOrbitInfo{Version: "1.36.0", DesktopVersion: ptr.String("42")}, nil
		}
		ds.AssertHasNoEncryptionKeyStoredFunc = func(ctx context.Context, hostID uint) error {
			return nil
		}
		host := &mdmlab.Host{ID: 1, Platform: "ubuntu", DiskEncryptionEnabled: ptr.Bool(true), OrbitVersion: ptr.String(mdmlab.MinOrbitLUKSVersion)}
		ds.QueueEscrowFunc = func(ctx context.Context, hostID uint) error {
			require.Equal(t, uint(1), hostID)
			return nil
		}

		err := svc.TriggerLinuxDiskEncryptionEscrow(ctx, host)
		require.NoError(t, err)
		require.True(t, ds.QueueEscrowFuncInvoked)
	})
}
