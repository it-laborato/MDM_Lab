package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSetupExperienceAuth(t *testing.T) {
	ds := new(mock.Store)
	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	teamID := uint(1)
	teamScriptID := uint(1)
	noTeamScriptID := uint(2)

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.SetSetupExperienceScriptFunc = func(ctx context.Context, script *mdmlab.Script) error {
		return nil
	}

	ds.GetSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) (*mdmlab.Script, error) {
		if teamID == nil {
			return &mdmlab.Script{ID: noTeamScriptID}, nil
		}
		switch *teamID {
		case uint(1):
			return &mdmlab.Script{ID: teamScriptID, TeamID: teamID}, nil
		default:
			return nil, newNotFoundError()
		}
	}
	ds.GetAnyScriptContentsFunc = func(ctx context.Context, id uint) ([]byte, error) {
		return []byte("echo"), nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		if teamID == nil {
			return nil
		}
		switch *teamID {
		case uint(1):
			return nil
		default:
			return newNotFoundError() // TODO: confirm if we want to return not found on deletes
		}
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: id}, nil
	}
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}
	ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) {
		return document, nil
	}

	testCases := []struct {
		name                  string
		user                  *mdmlab.User
		shouldFailTeamWrite   bool
		shouldFailGlobalWrite bool
		shouldFailTeamRead    bool
		shouldFailGlobalRead  bool
	}{
		{
			name:                  "global admin",
			user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global maintainer",
			user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global observer",
			user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global observer+",
			user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global gitops",
			user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team admin, belongs to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team maintainer, belongs to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer, belongs to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer+, belongs to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team gitops, belongs to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team admin, DOES NOT belong to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleAdmin}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team maintainer, DOES NOT belong to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleMaintainer}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer, DOES NOT belong to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserver}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer+, DOES NOT belong to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserverPlus}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team gitops, DOES NOT belong to team",
			user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleGitOps}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			t.Run("setup experience script", func(t *testing.T) {
				err := svc.SetSetupExperienceScript(ctx, nil, "test.sh", strings.NewReader("echo"))
				checkAuthErr(t, tt.shouldFailGlobalWrite, err)
				err = svc.DeleteSetupExperienceScript(ctx, nil)
				checkAuthErr(t, tt.shouldFailGlobalWrite, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, nil, false)
				checkAuthErr(t, tt.shouldFailGlobalRead, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, nil, true)
				checkAuthErr(t, tt.shouldFailGlobalRead, err)

				err = svc.SetSetupExperienceScript(ctx, &teamID, "test.sh", strings.NewReader("echo"))
				checkAuthErr(t, tt.shouldFailTeamWrite, err)
				err = svc.DeleteSetupExperienceScript(ctx, &teamID)
				checkAuthErr(t, tt.shouldFailTeamWrite, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, &teamID, false)
				checkAuthErr(t, tt.shouldFailTeamRead, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, &teamID, true)
				checkAuthErr(t, tt.shouldFailTeamRead, err)
			})
		})
	}
}

func TestMaybeUpdateSetupExperience(t *testing.T) {
	ds := new(mock.Store)
	// _, ctx := newTestService(t, ds, nil, nil, nil)
	ctx := context.Background()

	hostUUID := "host-uuid"
	scriptUUID := "script-uuid"
	softwareUUID := "software-uuid"
	vppUUID := "vpp-uuid"

	t.Run("unsupported result type", func(t *testing.T) {
		_, err := maybeUpdateSetupExperienceStatus(ctx, ds, map[string]interface{}{"key": "value"}, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported result type")
	})

	t.Run("script results", func(t *testing.T) {
		testCases := []struct {
			name          string
			exitCode      int
			expected      mdmlab.SetupExperienceStatusResultStatus
			alwaysUpdated bool
		}{
			{
				name:          "success",
				exitCode:      0,
				expected:      mdmlab.SetupExperienceStatusSuccess,
				alwaysUpdated: true,
			},
			{
				name:          "failure",
				exitCode:      1,
				expected:      mdmlab.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ds.MaybeUpdateSetupExperienceScriptStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status mdmlab.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, executionID, scriptUUID)
					require.Equal(t, tt.expected, status)
					require.True(t, status.IsValid())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceScriptStatusFuncInvoked = false

				result := mdmlab.SetupExperienceScriptResult{
					HostUUID:    hostUUID,
					ExecutionID: scriptUUID,
					ExitCode:    tt.exitCode,
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, true)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceScriptStatusFuncInvoked)
			})
		}
	})

	t.Run("software install results", func(t *testing.T) {
		testCases := []struct {
			name          string
			status        mdmlab.SoftwareInstallerStatus
			expectStatus  mdmlab.SetupExperienceStatusResultStatus
			alwaysUpdated bool
		}{
			{
				name:          "success",
				status:        mdmlab.SoftwareInstalled,
				expectStatus:  mdmlab.SetupExperienceStatusSuccess,
				alwaysUpdated: true,
			},
			{
				name:          "failure",
				status:        mdmlab.SoftwareInstallFailed,
				expectStatus:  mdmlab.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
			{
				name:          "pending",
				status:        mdmlab.SoftwareInstallPending,
				expectStatus:  mdmlab.SetupExperienceStatusPending,
				alwaysUpdated: false,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				requireTerminalStatus := true // when this flag is true, we don't expect pending status to update

				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status mdmlab.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, executionID, softwareUUID)
					require.Equal(t, tt.expectStatus, status)
					require.True(t, status.IsValid())
					require.True(t, status.IsTerminalStatus())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false

				result := mdmlab.SetupExperienceSoftwareInstallResult{
					HostUUID:        hostUUID,
					ExecutionID:     softwareUUID,
					InstallerStatus: tt.status,
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked)

				requireTerminalStatus = false // when this flag is false, we do expect pending status to update

				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status mdmlab.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, executionID, softwareUUID)
					require.Equal(t, tt.expectStatus, status)
					require.True(t, status.IsValid())
					if status.IsTerminalStatus() {
						require.True(t, status == mdmlab.SetupExperienceStatusSuccess || status == mdmlab.SetupExperienceStatusFailure)
					} else {
						require.True(t, status == mdmlab.SetupExperienceStatusPending || status == mdmlab.SetupExperienceStatusRunning)
					}
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false
				updated, err = maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				shouldUpdate := tt.alwaysUpdated
				if tt.expectStatus == mdmlab.SetupExperienceStatusPending || tt.expectStatus == mdmlab.SetupExperienceStatusRunning {
					shouldUpdate = true
				}
				require.Equal(t, shouldUpdate, updated)
				require.Equal(t, shouldUpdate, ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked)
			})
		}
	})

	t.Run("vpp install results", func(t *testing.T) {
		testCases := []struct {
			name          string
			status        string
			expected      mdmlab.SetupExperienceStatusResultStatus
			alwaysUpdated bool
		}{
			{
				name:          "success",
				status:        mdmlab.MDMAppleStatusAcknowledged,
				expected:      mdmlab.SetupExperienceStatusSuccess,
				alwaysUpdated: true,
			},
			{
				name:          "failure",
				status:        mdmlab.MDMAppleStatusError,
				expected:      mdmlab.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
			{
				name:          "format error",
				status:        mdmlab.MDMAppleStatusCommandFormatError,
				expected:      mdmlab.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
			{
				name:          "pending",
				status:        mdmlab.MDMAppleStatusNotNow,
				expected:      mdmlab.SetupExperienceStatusPending,
				alwaysUpdated: false,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				requireTerminalStatus := true // when this flag is true, we don't expect pending status to update

				ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(ctx context.Context, hostUUID string, cmdUUID string, status mdmlab.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, cmdUUID, vppUUID)
					require.Equal(t, tt.expected, status)
					require.True(t, status.IsValid())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked = false

				result := mdmlab.SetupExperienceVPPInstallResult{
					HostUUID:      hostUUID,
					CommandUUID:   vppUUID,
					CommandStatus: tt.status,
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked)

				requireTerminalStatus = false // when this flag is false, we do expect pending status to update

				ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(ctx context.Context, hostUUID string, cmdUUID string, status mdmlab.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, cmdUUID, vppUUID)
					require.Equal(t, tt.expected, status)
					require.True(t, status.IsValid())
					if status.IsTerminalStatus() {
						require.True(t, status == mdmlab.SetupExperienceStatusSuccess || status == mdmlab.SetupExperienceStatusFailure)
					} else {
						require.True(t, status == mdmlab.SetupExperienceStatusPending || status == mdmlab.SetupExperienceStatusRunning)
					}
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked = false

				updated, err = maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				shouldUpdate := tt.alwaysUpdated
				if tt.expected == mdmlab.SetupExperienceStatusPending || tt.expected == mdmlab.SetupExperienceStatusRunning {
					shouldUpdate = true
				}
				require.Equal(t, shouldUpdate, updated)
				require.Equal(t, shouldUpdate, ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked)
			})
		}
	})
}
