package service

import (
	"context"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/datastore/mysql"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPack(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.PackFunc = func(ctx context.Context, id uint) (*mdmlab.Pack, error) {
		return &mdmlab.Pack{
			ID:      1,
			TeamIDs: []uint{1},
		}, nil
	}

	pack, err := svc.GetPack(test.UserContext(ctx, test.UserAdmin), 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), pack.ID)

	_, err = svc.GetPack(test.UserContext(ctx, test.UserNoRoles), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestNewPackSavesTargets(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewPackFunc = func(ctx context.Context, pack *mdmlab.Pack, opts ...mdmlab.OptionalArg) (*mdmlab.Pack, error) {
		return pack, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}

	packPayload := mdmlab.PackPayload{
		Name:     ptr.String("foo"),
		HostIDs:  &[]uint{123},
		LabelIDs: &[]uint{456},
		TeamIDs:  &[]uint{789},
	}
	pack, err := svc.NewPack(test.UserContext(ctx, test.UserAdmin), packPayload)
	require.NoError(t, err)

	require.Len(t, pack.HostIDs, 1)
	require.Len(t, pack.LabelIDs, 1)
	require.Len(t, pack.TeamIDs, 1)
	assert.Equal(t, uint(123), pack.HostIDs[0])
	assert.Equal(t, uint(456), pack.LabelIDs[0])
	assert.Equal(t, uint(789), pack.TeamIDs[0])
	assert.True(t, ds.NewPackFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)
}

func TestPacksWithDS(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *mysql.Datastore)
	}{
		{"ListPacks", testPacksListPacks},
		{"DeletePack", testPacksDeletePack},
		{"DeletePackByID", testPacksDeletePackByID},
		{"ApplyPackSpecs", testPacksApplyPackSpecs},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPacksListPacks(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)

	queries, err := svc.ListPacks(test.UserContext(ctx, test.UserAdmin), mdmlab.PackListOptions{IncludeSystemPacks: false})
	require.NoError(t, err)
	assert.Len(t, queries, 0)

	_, err = ds.NewPack(ctx, &mdmlab.Pack{
		Name: "foo",
	})
	require.NoError(t, err)

	queries, err = svc.ListPacks(test.UserContext(ctx, test.UserAdmin), mdmlab.PackListOptions{IncludeSystemPacks: false})
	require.NoError(t, err)
	assert.Len(t, queries, 1)
}

func testPacksDeletePack(t *testing.T, ds *mysql.Datastore) {
	test.AddAllHostsLabel(t, ds)

	users := createTestUsers(t, ds)
	user := users["admin1@example.com"]

	type args struct {
		ctx  context.Context
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "delete pack that doesn't exist",
			args: args{
				ctx:  test.UserContext(context.Background(), &user),
				name: "foo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t, ds, nil, nil)
			if err := svc.DeletePack(tt.args.ctx, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("DeletePack() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testPacksDeletePackByID(t *testing.T, ds *mysql.Datastore) {
	test.AddAllHostsLabel(t, ds)

	type args struct {
		ctx context.Context
		id  uint
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "cannot delete pack that doesn't exists",
			args: args{
				ctx: test.UserContext(context.Background(), test.UserAdmin),
				id:  123456,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t, ds, nil, nil)
			if err := svc.DeletePackByID(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("DeletePackByID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testPacksApplyPackSpecs(t *testing.T, ds *mysql.Datastore) {
	test.AddAllHostsLabel(t, ds)

	users := createTestUsers(t, ds)
	user := users["admin1@example.com"]

	type args struct {
		ctx   context.Context
		specs []*mdmlab.PackSpec
	}
	tests := []struct {
		name    string
		args    args
		want    []*mdmlab.PackSpec
		wantErr bool
	}{
		{
			name: "cannot modify global pack",
			args: args{
				ctx: test.UserContext(context.Background(), &user),
				specs: []*mdmlab.PackSpec{
					{Name: "Foo Pack", Description: "Foo Desc", Platform: "MacOS"},
					{Name: "Bar Pack", Description: "Bar Desc", Platform: "MacOS"},
				},
			},
			want: []*mdmlab.PackSpec{
				{Name: "Foo Pack", Description: "Foo Desc", Platform: "MacOS"},
				{Name: "Bar Pack", Description: "Bar Desc", Platform: "MacOS"},
			},
			wantErr: false,
		},
		{
			name: "cannot modify team pack",
			args: args{
				ctx: test.UserContext(context.Background(), &user),
				specs: []*mdmlab.PackSpec{
					{Name: "Test", Description: "Test Desc", Platform: "linux"},
				},
			},
			want: []*mdmlab.PackSpec{
				{Name: "Test", Description: "Test Desc", Platform: "linux"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t, ds, nil, nil)
			got, err := svc.ApplyPackSpecs(tt.args.ctx, tt.args.specs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPackSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUserIsGitOpsOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		user       *mdmlab.User
		expectedFn func(value bool, err error) bool
	}{
		{
			name: "missing user in context",
			user: nil,
			expectedFn: func(value bool, err error) bool {
				return err != nil && !value
			},
		},
		{
			name: "no roles",
			user: &mdmlab.User{},
			expectedFn: func(value bool, err error) bool {
				return err != nil && !value
			},
		},
		{
			name: "global gitops",
			user: &mdmlab.User{
				GlobalRole: ptr.String(mdmlab.RoleGitOps),
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "global non-gitops",
			user: &mdmlab.User{
				GlobalRole: ptr.String(mdmlab.RoleObserver),
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
		{
			name: "team gitops",
			user: &mdmlab.User{
				GlobalRole: nil,
				Teams: []mdmlab.UserTeam{
					{
						Team: mdmlab.Team{ID: 1},
						Role: mdmlab.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "multiple team gitops",
			user: &mdmlab.User{
				GlobalRole: nil,
				Teams: []mdmlab.UserTeam{
					{
						Team: mdmlab.Team{ID: 1},
						Role: mdmlab.RoleGitOps,
					},
					{
						Team: mdmlab.Team{ID: 2},
						Role: mdmlab.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "multiple teams, not all gitops",
			user: &mdmlab.User{
				GlobalRole: nil,
				Teams: []mdmlab.UserTeam{
					{
						Team: mdmlab.Team{ID: 1},
						Role: mdmlab.RoleObserver,
					},
					{
						Team: mdmlab.Team{ID: 2},
						Role: mdmlab.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
		{
			name: "multiple teams, none gitops",
			user: &mdmlab.User{
				GlobalRole: nil,
				Teams: []mdmlab.UserTeam{
					{
						Team: mdmlab.Team{ID: 1},
						Role: mdmlab.RoleObserver,
					},
					{
						Team: mdmlab.Team{ID: 2},
						Role: mdmlab.RoleMaintainer,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := userIsGitOpsOnly(viewer.NewContext(context.Background(), viewer.Viewer{User: tc.user}))
			require.True(t, tc.expectedFn(actual, err))
		})
	}
}
