package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/authz"
	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com:it-laborato/MDM_Lab/server/test"
	"github.com/stretchr/testify/require"
)

func TestHostRunScript(t *testing.T) {
	ds := new(mock.Store)
	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	// use a custom implementation of checkAuthErr as the service call will fail
	// with a not found error for unknown host in case of authorization success,
	// and the package-wide checkAuthErr requires no error.
	checkAuthErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
		} else if err != nil {
			require.NotEqual(t, (&authz.Forbidden{}).Error(), err.Error())
		}
	}

	teamHost := &mdmlab.Host{ID: 1, Hostname: "host-team", TeamID: ptr.Uint(1), SeenTime: time.Now(), OrbitNodeKey: ptr.String("abc")}
	noTeamHost := &mdmlab.Host{ID: 2, Hostname: "host-no-team", TeamID: nil, SeenTime: time.Now(), OrbitNodeKey: ptr.String("def")}
	nonExistingHost := &mdmlab.Host{ID: 3, Hostname: "no-such-host", TeamID: nil}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.HostFunc = func(ctx context.Context, hostID uint) (*mdmlab.Host, error) {
		if hostID == 1 {
			return teamHost, nil
		}
		if hostID == 2 {
			return noTeamHost, nil
		}
		return nil, newNotFoundError()
	}
	ds.NewHostScriptExecutionRequestFunc = func(ctx context.Context, request *mdmlab.HostScriptRequestPayload) (*mdmlab.HostScriptResult, error) {
		return &mdmlab.HostScriptResult{HostID: request.HostID, ScriptContents: request.ScriptContents, ExecutionID: "abc"}, nil
	}
	ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*mdmlab.HostScriptResult, error) {
		return nil, nil
	}
	ds.ScriptFunc = func(ctx context.Context, id uint) (*mdmlab.Script, error) {
		return &mdmlab.Script{ID: id}, nil
	}
	ds.GetScriptContentsFunc = func(ctx context.Context, id uint) ([]byte, error) {
		return []byte("echo"), nil
	}
	ds.IsExecutionPendingForHostFunc = func(ctx context.Context, hostID, scriptID uint) (bool, error) { return false, nil }
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}

	t.Run("authorization checks", func(t *testing.T) {
		testCases := []struct {
			name                  string
			user                  *mdmlab.User
			scriptID              *uint
			shouldFailTeamWrite   bool
			shouldFailGlobalWrite bool
		}{
			{
				name:                  "global admin",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: false,
			},
			{
				name:                  "global admin saved",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: false,
			},
			{
				name:                  "global maintainer",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: false,
			},
			{
				name:                  "global maintainer saved",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: false,
			},
			{
				name:                  "global observer",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "global observer saved",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "global observer+",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "global observer+ saved",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "global gitops",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "global gitops saved",
				user:                  &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team admin, belongs to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team admin, belongs to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team maintainer, belongs to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team maintainer, belongs to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer, belongs to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer, belongs to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer+, belongs to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer+, belongs to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team gitops, belongs to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team gitops, belongs to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team admin, DOES NOT belong to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleAdmin}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team admin, DOES NOT belong to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleAdmin}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team maintainer, DOES NOT belong to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleMaintainer}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team maintainer, DOES NOT belong to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleMaintainer}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer, DOES NOT belong to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserver}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer, DOES NOT belong to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserver}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer+, DOES NOT belong to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserverPlus}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer+, DOES NOT belong to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserverPlus}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team gitops, DOES NOT belong to team",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleGitOps}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team gitops, DOES NOT belong to team, saved",
				user:                  &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleGitOps}}},
				scriptID:              ptr.Uint(1),
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

				contents := "abc"
				if tt.scriptID != nil {
					contents = ""
				}
				_, err := svc.RunHostScript(ctx, &mdmlab.HostScriptRequestPayload{HostID: noTeamHost.ID, ScriptContents: contents, ScriptID: tt.scriptID}, 0)
				checkAuthErr(t, tt.shouldFailGlobalWrite, err)
				_, err = svc.RunHostScript(ctx, &mdmlab.HostScriptRequestPayload{HostID: teamHost.ID, ScriptContents: contents, ScriptID: tt.scriptID}, 0)
				checkAuthErr(t, tt.shouldFailTeamWrite, err)

				if tt.scriptID == nil {
					// a non-existing host is authorized as for global write (because we can't know what team it belongs to)
					_, err = svc.RunHostScript(ctx, &mdmlab.HostScriptRequestPayload{HostID: nonExistingHost.ID, ScriptContents: "abc"}, 0)
					checkAuthErr(t, tt.shouldFailGlobalWrite, err)
				}

				// test auth for run sync saved script by name
				if tt.scriptID != nil {
					ds.GetScriptIDByNameFunc = func(ctx context.Context, name string, teamID *uint) (uint, error) {
						return *tt.scriptID, nil
					}
					_, err = svc.RunHostScript(ctx, &mdmlab.HostScriptRequestPayload{HostID: noTeamHost.ID, ScriptContents: "", ScriptID: nil, ScriptName: "Foo", TeamID: 1}, 1)
					checkAuthErr(t, tt.shouldFailGlobalWrite, err)
					_, err = svc.RunHostScript(ctx, &mdmlab.HostScriptRequestPayload{HostID: teamHost.ID, ScriptContents: "", ScriptID: nil, ScriptName: "Foo", TeamID: 1}, 1)
					checkAuthErr(t, tt.shouldFailTeamWrite, err)
				}
			})
		}
	})

	t.Run("script contents validation", func(t *testing.T) {
		testCases := []struct {
			name    string
			script  string
			wantErr string
		}{
			{"empty script", "", "One of 'script_id', 'script_contents', or 'script_name' is required."},
			{"overly long script", strings.Repeat("a", mdmlab.UnsavedScriptMaxRuneLen+1), "Script is too large."},
			{"large script", strings.Repeat("a", mdmlab.UnsavedScriptMaxRuneLen), ""},
			{"invalid utf8", "\xff\xfa", "Wrong data format."},
			{"valid without hashbang", "echo 'a'", ""},
			{"valid with posix hashbang", "#!/bin/sh\necho 'a'", ""},
			{"valid with usr zsh hashbang", "#!/usr/bin/zsh\necho 'a'", ""},
			{"valid with zsh hashbang", "#!/bin/zsh\necho 'a'", ""},
			{"valid with zsh hashbang and arguments", "#!/bin/zsh -x\necho 'a'", ""},
			{"valid with hashbang and spacing", "#! /bin/sh  \necho 'a'", ""},
			{"valid with hashbang and Windows newline", "#! /bin/sh  \r\necho 'a'", ""},
			{"invalid hashbang", "#!/bin/bash\necho 'a'", "Interpreter not supported."},
		}

		ctx = viewer.NewContext(ctx, viewer.Viewer{User: test.UserAdmin})
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				_, err := svc.RunHostScript(ctx, &mdmlab.HostScriptRequestPayload{HostID: noTeamHost.ID, ScriptContents: tt.script}, 0)
				if tt.wantErr != "" {
					require.ErrorContains(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestGetScriptResult(t *testing.T) {
	ds := new(mock.Store)
	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	const (
		noTeamHostExecID      = "no-team-host"
		teamHostExecID        = "team-host"
		nonExistingHostExecID = "non-existing-host"
	)

	checkAuthErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
		} else if err != nil {
			require.NotEqual(t, (&authz.Forbidden{}).Error(), err.Error())
		}
	}

	teamHost := &mdmlab.Host{ID: 1, Hostname: "host-team", TeamID: ptr.Uint(1), SeenTime: time.Now()}
	noTeamHost := &mdmlab.Host{ID: 2, Hostname: "host-no-team", TeamID: nil, SeenTime: time.Now()}
	nonExistingHost := &mdmlab.Host{ID: 3, Hostname: "no-such-host", TeamID: nil}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.GetHostScriptExecutionResultFunc = func(ctx context.Context, executionID string) (*mdmlab.HostScriptResult, error) {
		switch executionID {
		case noTeamHostExecID:
			return &mdmlab.HostScriptResult{HostID: noTeamHost.ID, ScriptContents: "abc", ExecutionID: executionID}, nil
		case teamHostExecID:
			return &mdmlab.HostScriptResult{HostID: teamHost.ID, ScriptContents: "abc", ExecutionID: executionID}, nil
		case nonExistingHostExecID:
			return &mdmlab.HostScriptResult{HostID: nonExistingHost.ID, ScriptContents: "abc", ExecutionID: executionID}, nil
		default:
			return nil, newNotFoundError()
		}
	}
	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*mdmlab.Host, error) {
		if hostID == 1 {
			return teamHost, nil
		}
		if hostID == 2 {
			return noTeamHost, nil
		}
		return nil, newNotFoundError()
	}

	testCases := []struct {
		name                 string
		user                 *mdmlab.User
		shouldFailTeamRead   bool
		shouldFailGlobalRead bool
	}{
		{
			name:                 "global admin",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global maintainer",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global observer",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global observer+",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global gitops",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team admin, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team maintainer, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer+, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team gitops, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team admin, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleAdmin}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team maintainer, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleMaintainer}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserver}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer+, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserverPlus}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team gitops, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleGitOps}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetScriptResult(ctx, noTeamHostExecID)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)
			_, err = svc.GetScriptResult(ctx, teamHostExecID)
			checkAuthErr(t, tt.shouldFailTeamRead, err)

			// a non-existing host is authorized as for global write (because we can't know what team it belongs to)
			_, err = svc.GetScriptResult(ctx, nonExistingHostExecID)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)
		})
	}
}

func TestSavedScripts(t *testing.T) {
	ds := new(mock.Store)
	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	withLFContents := "echo\necho"
	withCRLFContents := "echo\r\necho"

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.NewScriptFunc = func(ctx context.Context, script *mdmlab.Script) (*mdmlab.Script, error) {
		require.Equal(t, withLFContents, script.ScriptContents)
		newScript := *script
		newScript.ID = 1
		return &newScript, nil
	}
	const (
		team1ScriptID  = 1
		noTeamScriptID = 2
	)
	ds.ScriptFunc = func(ctx context.Context, id uint) (*mdmlab.Script, error) {
		switch id {
		case team1ScriptID:
			return &mdmlab.Script{ID: id, TeamID: ptr.Uint(1)}, nil
		default:
			return &mdmlab.Script{ID: id}, nil
		}
	}
	ds.GetScriptContentsFunc = func(ctx context.Context, id uint) ([]byte, error) {
		return []byte("echo"), nil
	}
	ds.DeleteScriptFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.ListScriptsFunc = func(ctx context.Context, teamID *uint, opt mdmlab.ListOptions) ([]*mdmlab.Script, *mdmlab.PaginationMetadata, error) {
		return nil, &mdmlab.PaginationMetadata{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: 0}, nil
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

			_, err := svc.NewScript(ctx, nil, "test.ps1", strings.NewReader(withCRLFContents))
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)
			err = svc.DeleteScript(ctx, noTeamScriptID)
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)
			_, _, err = svc.ListScripts(ctx, nil, mdmlab.ListOptions{})
			checkAuthErr(t, tt.shouldFailGlobalRead, err)
			_, _, err = svc.GetScript(ctx, noTeamScriptID, false)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)
			_, _, err = svc.GetScript(ctx, noTeamScriptID, true)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.NewScript(ctx, ptr.Uint(1), "test.sh", strings.NewReader(withLFContents))
			checkAuthErr(t, tt.shouldFailTeamWrite, err)
			err = svc.DeleteScript(ctx, team1ScriptID)
			checkAuthErr(t, tt.shouldFailTeamWrite, err)
			_, _, err = svc.ListScripts(ctx, ptr.Uint(1), mdmlab.ListOptions{})
			checkAuthErr(t, tt.shouldFailTeamRead, err)
			_, _, err = svc.GetScript(ctx, team1ScriptID, false)
			checkAuthErr(t, tt.shouldFailTeamRead, err)
			_, _, err = svc.GetScript(ctx, team1ScriptID, true)
			checkAuthErr(t, tt.shouldFailTeamRead, err)
		})
	}
}

func TestHostScriptDetailsAuth(t *testing.T) {
	ds := new(mock.Store)
	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}

	testCases := []struct {
		name                 string
		user                 *mdmlab.User
		shouldFailTeamRead   bool
		shouldFailGlobalRead bool
	}{
		{
			name:                 "global admin",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global maintainer",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global observer",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global observer+",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global gitops",
			user:                 &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team admin, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team maintainer, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer+, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team gitops, belongs to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team admin, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleAdmin}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team maintainer, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleMaintainer}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserver}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer+, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserverPlus}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team gitops, DOES NOT belong to team",
			user:                 &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleGitOps}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			t.Run("no team host script details", func(t *testing.T) {
				ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*mdmlab.Host, error) {
					require.Equal(t, uint(42), hostID)
					return &mdmlab.Host{ID: hostID}, nil
				}
				ds.GetHostScriptDetailsFunc = func(ctx context.Context, hostID uint, teamID *uint, opts mdmlab.ListOptions, hostPlatform string) ([]*mdmlab.HostScriptDetail, *mdmlab.PaginationMetadata, error) {
					require.Nil(t, teamID)
					return []*mdmlab.HostScriptDetail{}, nil, nil
				}
				_, _, err := svc.GetHostScriptDetails(ctx, 42, mdmlab.ListOptions{})
				checkAuthErr(t, tt.shouldFailGlobalRead, err)
			})

			t.Run("team host script details", func(t *testing.T) {
				ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*mdmlab.Host, error) {
					require.Equal(t, uint(42), hostID)
					return &mdmlab.Host{ID: hostID, TeamID: ptr.Uint(1)}, nil
				}
				ds.GetHostScriptDetailsFunc = func(ctx context.Context, hostID uint, teamID *uint, opts mdmlab.ListOptions, hostPlatform string) ([]*mdmlab.HostScriptDetail, *mdmlab.PaginationMetadata, error) {
					require.NotNil(t, teamID)
					require.Equal(t, uint(1), *teamID)
					return []*mdmlab.HostScriptDetail{}, nil, nil
				}
				_, _, err := svc.GetHostScriptDetails(ctx, 42, mdmlab.ListOptions{})
				checkAuthErr(t, tt.shouldFailTeamRead, err)
			})

			t.Run("host not found", func(t *testing.T) {
				ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*mdmlab.Host, error) {
					require.Equal(t, uint(43), hostID)
					return nil, &notFoundError{}
				}
				_, _, err := svc.GetHostScriptDetails(ctx, 43, mdmlab.ListOptions{})
				if tt.shouldFailGlobalRead {
					checkAuthErr(t, tt.shouldFailGlobalRead, err)
				} else {
					require.True(t, mdmlab.IsNotFound(err))
				}
			})
		})
	}
}
