package service

import (
	"context"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCheckPolicySpecAuthorization(t *testing.T) {
	t.Run("when team not found", func(t *testing.T) {
		ds := new(mock.Store)
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
			return nil, &notFoundError{}
		}

		svc, ctx := newTestService(t, ds, nil, nil)

		req := []*mdmlab.PolicySpec{
			{
				Team: "some_team",
			},
		}

		user := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

		actual := svc.ApplyPolicySpecs(ctx, req)
		var expected mdmlab.NotFoundError

		require.ErrorAs(t, actual, &expected)
	})
}

func TestGlobalPoliciesAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewGlobalPolicyFunc = func(ctx context.Context, authorID *uint, args mdmlab.PolicyPayload) (*mdmlab.Policy, error) {
		return &mdmlab.Policy{}, nil
	}
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts mdmlab.ListOptions) ([]*mdmlab.Policy, error) {
		return nil, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*mdmlab.Policy, error) {
		return nil, nil
	}
	ds.PolicyFunc = func(ctx context.Context, id uint) (*mdmlab.Policy, error) {
		return &mdmlab.Policy{
			PolicyData: mdmlab.PolicyData{
				ID: id,
			},
		}, nil
	}
	ds.DeleteGlobalPoliciesFunc = func(ctx context.Context, ids []uint) ([]uint, error) {
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: 1}, nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*mdmlab.PolicySpec) error {
		return nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.SavePolicyFunc = func(ctx context.Context, p *mdmlab.Policy, shouldDeleteAll bool, removePolicyStats bool) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{
			WebhookSettings: mdmlab.WebhookSettings{
				FailingPoliciesWebhook: mdmlab.FailingPoliciesWebhookSettings{
					Enable: false,
				},
			},
		}, nil
	}

	testCases := []struct {
		name            string
		user            *mdmlab.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			false,
			false,
		},
		{
			"global observer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			true,
			false,
		},
		{
			"team admin",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			true,
			false,
		},
		{
			"team maintainer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team observer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			true,
			false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.NewGlobalPolicy(ctx, mdmlab.PolicyPayload{
				Name:  "query1",
				Query: "select 1;",
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ListGlobalPolicies(ctx, mdmlab.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetPolicyByIDQueries(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyGlobalPolicy(ctx, 1, mdmlab.ModifyPolicyPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.DeleteGlobalPolicies(ctx, []uint{1})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.ApplyPolicySpecs(ctx, []*mdmlab.PolicySpec{
				{
					Name:  "query2",
					Query: "select 1;",
				},
			})
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestRemoveGlobalPoliciesFromWebhookConfig(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{ds: ds}

	var storedAppConfig mdmlab.AppConfig

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &storedAppConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, info *mdmlab.AppConfig) error {
		storedAppConfig = *info
		return nil
	}

	for _, tc := range []struct {
		name     string
		currCfg  []uint
		toDelete []uint
		expCfg   []uint
	}{
		{
			name:     "delete-one",
			currCfg:  []uint{1},
			toDelete: []uint{1},
			expCfg:   []uint{},
		},
		{
			name:     "delete-all-2",
			currCfg:  []uint{1, 2, 3},
			toDelete: []uint{1, 2, 3},
			expCfg:   []uint{},
		},
		{
			name:     "basic",
			currCfg:  []uint{1, 2, 3},
			toDelete: []uint{1, 2},
			expCfg:   []uint{3},
		},
		{
			name:     "empty-cfg",
			currCfg:  []uint{},
			toDelete: []uint{1},
			expCfg:   []uint{},
		},
		{
			name:     "no-deletion-cfg",
			currCfg:  []uint{1},
			toDelete: []uint{2, 3, 4},
			expCfg:   []uint{1},
		},
		{
			name:     "no-deletion-cfg-2",
			currCfg:  []uint{1},
			toDelete: []uint{},
			expCfg:   []uint{1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			storedAppConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs = tc.currCfg
			err := svc.removeGlobalPoliciesFromWebhookConfig(context.Background(), tc.toDelete)
			require.NoError(t, err)
			require.Equal(t, tc.expCfg, storedAppConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
		})
	}
}

// test ApplyPolicySpecsReturnsErrorOnDuplicatePolicyNamesInSpecs
func TestApplyPolicySpecsReturnsErrorOnDuplicatePolicyNamesInSpecs(t *testing.T) {
	ds := new(mock.Store)
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		return nil, &notFoundError{}
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	req := []*mdmlab.PolicySpec{
		{
			Name:     "query1",
			Query:    "select 1;",
			Platform: "windows",
		},
		{
			Name:     "query1",
			Query:    "select 1;",
			Platform: "windows",
		},
	}

	user := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	err := svc.ApplyPolicySpecs(ctx, req)

	badRequestError := &mdmlab.BadRequestError{}
	require.ErrorAs(t, err, &badRequestError)
	require.Equal(t, "duplicate policy names not allowed", badRequestError.Message)
}
