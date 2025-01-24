package service

import (
	"context"
	"testing"

	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchTargets(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	user := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	hosts := []*mdmlab.Host{
		{Hostname: "foo.local"},
	}
	labels := []*mdmlab.Label{
		{
			Name:  "label foo",
			Query: "query foo",
		},
	}
	teams := []*mdmlab.Team{
		{Name: "team1"},
	}

	ds.SearchHostsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, query string, omit ...uint) ([]*mdmlab.Host, error) {
		assert.Equal(t, user, filter.User)
		return hosts, nil
	}
	ds.SearchLabelsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, query string, omit ...uint) ([]*mdmlab.Label, error) {
		assert.Equal(t, user, filter.User)
		return labels, nil
	}
	ds.SearchTeamsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, query string, omit ...uint) ([]*mdmlab.Team, error) {
		assert.Equal(t, user, filter.User)
		return teams, nil
	}

	results, err := svc.SearchTargets(ctx, "foo", nil, mdmlab.HostTargets{})
	require.NoError(t, err)
	assert.Equal(t, hosts[0], results.Hosts[0])
	assert.Equal(t, labels[0], results.Labels[0])
	assert.Equal(t, teams[0], results.Teams[0])
}

func TestSearchWithOmit(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	user := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	ds.SearchHostsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, query string, omit ...uint) ([]*mdmlab.Host, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{1, 2}, omit)
		return nil, nil
	}
	ds.SearchLabelsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, query string, omit ...uint) ([]*mdmlab.Label, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{3, 4}, omit)
		return nil, nil
	}
	ds.SearchTeamsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, query string, omit ...uint) ([]*mdmlab.Team, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{5, 6}, omit)
		return nil, nil
	}

	_, err := svc.SearchTargets(ctx, "foo", nil, mdmlab.HostTargets{HostIDs: []uint{1, 2}, LabelIDs: []uint{3, 4}, TeamIDs: []uint{5, 6}})
	require.NoError(t, err)
}
