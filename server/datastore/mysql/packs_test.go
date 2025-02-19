package mysql

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/mixer/clock"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPacks(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Delete", testPacksDelete},
		{"Save", testPacksSave},
		{"GetByName", testPacksGetByName},
		{"List", testPacksList},
		{"ApplySpecRoundtrip", testPacksApplySpecRoundtrip},
		{"GetSpec", testPacksGetSpec},
		{"ApplySpecMissingQueries", testPacksApplySpecMissingQueries},
		{"ApplySpecMissingName", testPacksApplySpecMissingName},
		{"ListForHost", testPacksListForHost},
		{"ApplySpecFailsOnTargetIDNull", testPacksApplySpecFailsOnTargetIDNull},
		{"ApplyStatsNotLocking", testPacksApplyStatsNotLocking},
		{"ApplyStatsNotLockingTryTwo", testPacksApplyStatsNotLockingTryTwo},
		{"ListForHostIncludesOnlyUserPacks", testListForHostIncludesOnlyUserPacks},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPacksDelete(t *testing.T, ds *Datastore) {
	pack := test.NewPack(t, ds, "foo")
	assert.NotEqual(t, uint(0), pack.ID)

	pack, err := ds.Pack(context.Background(), pack.ID)
	require.Nil(t, err)

	err = ds.DeletePack(context.Background(), pack.Name)
	assert.Nil(t, err)

	assert.NotEqual(t, uint(0), pack.ID)
	_, err = ds.Pack(context.Background(), pack.ID)
	assert.NotNil(t, err)
}

func testPacksSave(t *testing.T, ds *Datastore) {
	expectedPack := &mdmlab.Pack{
		Name:     "foo",
		Hosts:    []mdmlab.Target{{TargetID: 1, Type: mdmlab.TargetHost}},
		HostIDs:  []uint{1},
		Labels:   []mdmlab.Target{{TargetID: 1, Type: mdmlab.TargetLabel}},
		LabelIDs: []uint{1},
		Teams:    []mdmlab.Target{{TargetID: 1, Type: mdmlab.TargetTeam}},
		TeamIDs:  []uint{1},
	}

	pack, err := ds.NewPack(context.Background(), expectedPack)
	require.NoError(t, err)
	assert.NotEqual(t, uint(0), pack.ID)
	test.EqualSkipTimestampsID(t, expectedPack, pack)

	pack, err = ds.Pack(context.Background(), pack.ID)
	require.NoError(t, err)
	test.EqualSkipTimestampsID(t, expectedPack, pack)

	expectedPack = &mdmlab.Pack{
		ID:       pack.ID,
		Name:     "bar",
		Hosts:    []mdmlab.Target{{TargetID: 3, Type: mdmlab.TargetHost}},
		HostIDs:  []uint{3},
		Labels:   []mdmlab.Target{{TargetID: 4, Type: mdmlab.TargetLabel}, {TargetID: 6, Type: mdmlab.TargetLabel}},
		LabelIDs: []uint{4, 6},
		Teams:    []mdmlab.Target{},
		TeamIDs:  []uint{},
	}

	err = ds.SavePack(context.Background(), expectedPack)
	require.NoError(t, err)

	pack, err = ds.Pack(context.Background(), pack.ID)
	require.NoError(t, err)
	assert.Equal(t, "bar", pack.Name)
	test.EqualSkipTimestampsID(t, expectedPack, pack)
}

func testPacksGetByName(t *testing.T, ds *Datastore) {
	pack := test.NewPack(t, ds, "foo")
	assert.NotEqual(t, uint(0), pack.ID)

	pack, ok, err := ds.PackByName(context.Background(), pack.Name)
	require.Nil(t, err)
	assert.True(t, ok)
	assert.NotNil(t, pack)
	assert.Equal(t, "foo", pack.Name)

	pack, ok, err = ds.PackByName(context.Background(), "bar")
	require.Nil(t, err)
	assert.False(t, ok)
	assert.Nil(t, pack)
}

func testPacksList(t *testing.T, ds *Datastore) {
	p1 := &mdmlab.PackSpec{
		ID:   1,
		Name: "foo_pack",
	}
	p2 := &mdmlab.PackSpec{
		ID:   2,
		Name: "bar_pack",
	}
	err := ds.ApplyPackSpecs(context.Background(), []*mdmlab.PackSpec{p1})
	require.Nil(t, err)

	packs, err := ds.ListPacks(context.Background(), mdmlab.PackListOptions{IncludeSystemPacks: false})
	require.Nil(t, err)
	assert.Len(t, packs, 1)

	err = ds.ApplyPackSpecs(context.Background(), []*mdmlab.PackSpec{p1, p2})
	require.Nil(t, err)

	packs, err = ds.ListPacks(context.Background(), mdmlab.PackListOptions{IncludeSystemPacks: false})
	require.Nil(t, err)
	assert.Len(t, packs, 2)
}

func setupPackSpecsTest(t *testing.T, ds mdmlab.Datastore) []*mdmlab.PackSpec {
	zwass := test.NewUser(t, ds, "Zach", "zwass@example.com", true)
	queries := []*mdmlab.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo", Logging: mdmlab.LoggingSnapshot},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar", Logging: mdmlab.LoggingSnapshot},
	}
	// Zach creates some queries
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries, nil)
	require.NoError(t, err)

	labels := []*mdmlab.LabelSpec{
		{
			Name:  "foo",
			Query: "select * from foo",
		},
		{
			Name:  "bar",
			Query: "select * from bar",
		},
		{
			Name:  "bing",
			Query: "select * from bing",
		},
	}
	err = ds.ApplyLabelSpecs(context.Background(), labels)
	require.Nil(t, err)

	// create some teams
	teams := []*mdmlab.Team{
		{
			Name: "team1",
		},
		{
			Name: "team2",
		},
		{
			Name: "team3",
		},
	}
	for _, team := range teams {
		_, err := ds.NewTeam(context.Background(), team)
		require.NoError(t, err)
	}

	expectedSpecs := []*mdmlab.PackSpec{
		{
			ID:   1,
			Name: "test_pack",
			Targets: mdmlab.PackSpecTargets{
				Labels: []string{
					"foo",
					"bar",
					"bing",
				},
				Teams: []string{
					"team1",
					"team2",
				},
			},
			Queries: []mdmlab.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Name:        "q0",
					Description: "test_foo",
					Interval:    42,
				},
				{
					QueryName: queries[0].Name,
					Name:      "foo_snapshot",
					Interval:  600,
					Snapshot:  ptr.Bool(true),
					Denylist:  ptr.Bool(false),
				},
				{
					Name:      "q2",
					QueryName: queries[1].Name,
					Interval:  600,
					Removed:   ptr.Bool(false),
					Shard:     ptr.Uint(73),
					Version:   ptr.String("0.0.0.0.0.1"),
					Denylist:  ptr.Bool(true),
				},
			},
		},
		{
			ID:       2,
			Name:     "test_pack_disabled",
			Disabled: true,
			Targets: mdmlab.PackSpecTargets{
				Labels: []string{
					"foo",
					"bar",
					"bing",
				},
			},
			Queries: []mdmlab.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Name:        "q0",
					Description: "test_foo",
					Interval:    42,
				},
				{
					QueryName: queries[0].Name,
					Name:      "foo_snapshot",
					Interval:  600,
					Snapshot:  ptr.Bool(true),
				},
				{
					Name:      "q2",
					QueryName: queries[1].Name,
					Interval:  600,
					Removed:   ptr.Bool(false),
					Shard:     ptr.Uint(73),
					Version:   ptr.String("0.0.0.0.0.1"),
				},
			},
		},
	}

	err = ds.ApplyPackSpecs(context.Background(), expectedSpecs)
	require.Nil(t, err)
	return expectedSpecs
}

func testPacksApplySpecRoundtrip(t *testing.T, ds *Datastore) {
	expectedSpecs := setupPackSpecsTest(t, ds)

	gotSpec, err := ds.GetPackSpecs(context.Background())
	require.Nil(t, err)
	assert.EqualValues(t, expectedSpecs, gotSpec)
}

func testPacksGetSpec(t *testing.T, ds *Datastore) {
	expectedSpecs := setupPackSpecsTest(t, ds)

	for _, s := range expectedSpecs {
		spec, err := ds.GetPackSpec(context.Background(), s.Name)
		require.Nil(t, err)
		assert.Equal(t, s, spec)
	}
}

func testPacksApplySpecMissingQueries(t *testing.T, ds *Datastore) {
	// Do not define queries mentioned in spec
	specs := []*mdmlab.PackSpec{
		{
			ID:   1,
			Name: "test_pack",
			Targets: mdmlab.PackSpecTargets{
				Labels: []string{},
			},
			Queries: []mdmlab.PackSpecQuery{
				{
					QueryName: "bar",
					Interval:  600,
				},
			},
		},
	}

	// Should error due to unknown query
	err := ds.ApplyPackSpecs(context.Background(), specs)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "unknown query 'bar'")
	}
}

func testPacksApplySpecMissingName(t *testing.T, ds *Datastore) {
	setupPackSpecsTest(t, ds)

	specs := []*mdmlab.PackSpec{
		{
			Name: "test2",
			Targets: mdmlab.PackSpecTargets{
				Labels: []string{},
			},
			Queries: []mdmlab.PackSpecQuery{
				{
					QueryName: "foo",
					Interval:  600,
				},
			},
		},
	}
	err := ds.ApplyPackSpecs(context.Background(), specs)
	require.NoError(t, err)

	// Query name should have been copied into name field
	spec, err := ds.GetPackSpec(context.Background(), "test2")
	require.NoError(t, err)
	assert.Equal(t, "foo", spec.Queries[0].Name)
}

func testPacksListForHost(t *testing.T, ds *Datastore) {
	mockClock := clock.NewMockClock()

	l1 := &mdmlab.LabelSpec{
		ID:   1,
		Name: "foo",
	}
	l2 := &mdmlab.LabelSpec{
		ID:   2,
		Name: "bar",
	}
	err := ds.ApplyLabelSpecs(context.Background(), []*mdmlab.LabelSpec{l1, l2})
	require.Nil(t, err)

	p1 := &mdmlab.PackSpec{
		ID:   1,
		Name: "foo_pack",
		Targets: mdmlab.PackSpecTargets{
			Labels: []string{
				l1.Name,
				l2.Name,
			},
		},
	}
	p2 := &mdmlab.PackSpec{
		ID:   2,
		Name: "shmoo_pack",
		Targets: mdmlab.PackSpecTargets{
			Labels: []string{
				l2.Name,
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), []*mdmlab.PackSpec{p1, p2})
	require.Nil(t, err)

	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", mockClock.Now())

	packs, err := ds.ListPacksForHost(context.Background(), h1.ID)
	require.Nil(t, err)
	require.Len(t, packs, 0)

	err = ds.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{l1.ID: ptr.Bool(true)}, mockClock.Now(), false)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(context.Background(), h1.ID)
	require.Nil(t, err)
	if assert.Len(t, packs, 1) {
		assert.Equal(t, "foo_pack", packs[0].Name)
	}

	err = ds.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{l1.ID: ptr.Bool(false), l2.ID: ptr.Bool(true)}, mockClock.Now(), false)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(context.Background(), h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)

	err = ds.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{l1.ID: ptr.Bool(true), l2.ID: ptr.Bool(true)}, mockClock.Now(), false)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(context.Background(), h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)

	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", mockClock.Now())

	err = ds.RecordLabelQueryExecutions(context.Background(), h2, map[uint]*bool{l2.ID: ptr.Bool(true)}, mockClock.Now(), false)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(context.Background(), h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)

	err = ds.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{l2.ID: ptr.Bool(false)}, mockClock.Now(), false)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(context.Background(), h1.ID)
	require.Nil(t, err)
	if assert.Len(t, packs, 1) {
		assert.Equal(t, "foo_pack", packs[0].Name)
	}
}

func testPacksApplySpecFailsOnTargetIDNull(t *testing.T, ds *Datastore) {
	// Do not define queries mentioned in spec
	specs := []*mdmlab.PackSpec{
		{
			ID:   1,
			Name: "test_pack",
			Targets: mdmlab.PackSpecTargets{
				Labels: []string{"UnexistentLabel"},
			},
		},
	}

	// Should error due to unkown label target id
	err := ds.ApplyPackSpecs(context.Background(), specs)
	require.Error(t, err)
}

func randomPackStatsForHost(packID uint, packName string, packType string, scheduledQueries []*mdmlab.ScheduledQuery, amount int) []mdmlab.PackStats {
	var queryStats []mdmlab.ScheduledQueryStats

	for i := 0; i < amount; i++ {
		sq := scheduledQueries[rand.Intn(len(scheduledQueries))]
		queryStats = append(queryStats, mdmlab.ScheduledQueryStats{
			ScheduledQueryName: sq.Name,
			ScheduledQueryID:   sq.ID,
			QueryName:          sq.QueryName,
			Description:        sq.Description,
			PackID:             packID,
			AverageMemory:      uint64(rand.Intn(100)),
			Denylisted:         false,
			Executions:         uint64(rand.Intn(100)),
			Interval:           rand.Intn(100),
			LastExecuted:       time.Now(),
			OutputSize:         uint64(rand.Intn(1000)),
			SystemTime:         uint64(rand.Intn(1000)),
			UserTime:           uint64(rand.Intn(1000)),
			WallTime:           uint64(rand.Intn(1000)),
		})
	}
	return []mdmlab.PackStats{
		{
			PackName:   packName,
			Type:       packType,
			PackID:     packID,
			QueryStats: queryStats,
		},
	}
}

func testPacksApplyStatsNotLocking(t *testing.T, ds *Datastore) {
	t.Skip("This can be too much for the test db if you're running all tests")

	specs := setupPackSpecsTest(t, ds)

	host, err := ds.NewHost(context.Background(), &mdmlab.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pack, _, err := ds.PackByName(context.Background(), "test_pack")
				require.NoError(t, err)
				schedQueries, err := ds.ListScheduledQueriesInPackWithStats(context.Background(), pack.ID, mdmlab.ListOptions{})
				require.NoError(t, err)

				amount := rand.Intn(5000)
				require.NoError(t, saveHostPackStatsDB(context.Background(), ds.writer(context.Background()), host.TeamID, host.ID, randomPackStatsForHost(pack.ID, pack.Name, *pack.Type, schedQueries, amount)))
			}
		}
	}()

	time.Sleep(1 * time.Second)
	for i := 0; i < 1000; i++ {
		require.NoError(t, ds.ApplyPackSpecs(context.Background(), specs))
		time.Sleep(77 * time.Millisecond)
	}

	cancelFunc()
}

func testPacksApplyStatsNotLockingTryTwo(t *testing.T, ds *Datastore) {
	t.Skip("This can be too much for the test db if you're running all tests")

	setupPackSpecsTest(t, ds)

	host, err := ds.NewHost(context.Background(), &mdmlab.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	ctx, cancelFunc := context.WithCancel(context.Background())
	for i := 0; i < 2; i++ {
		go func() {
			ms := rand.Intn(100)
			if ms == 0 {
				ms = 10
			}
			ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					pack, _, err := ds.PackByName(context.Background(), "test_pack")
					require.NoError(t, err)
					schedQueries, err := ds.ListScheduledQueriesInPackWithStats(context.Background(), pack.ID, mdmlab.ListOptions{})
					require.NoError(t, err)

					amount := rand.Intn(5000)
					require.NoError(t, saveHostPackStatsDB(context.Background(), ds.writer(context.Background()), host.TeamID, host.ID, randomPackStatsForHost(pack.ID, pack.Name, *pack.Type, schedQueries, amount)))
				}
			}
		}()
	}

	time.Sleep(60 * time.Second)

	cancelFunc()
}

func testListForHostIncludesOnlyUserPacks(t *testing.T, ds *Datastore) {
	mockClock := clock.NewMockClock()
	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", mockClock.Now())
	ctx := context.Background()

	label := &mdmlab.LabelSpec{
		ID:   1,
		Name: "All Hosts",
	}
	require.NoError(t, ds.ApplyLabelSpecs(ctx, []*mdmlab.LabelSpec{label}))

	pack := &mdmlab.PackSpec{
		ID:   1,
		Name: "foo_pack",
		Targets: mdmlab.PackSpecTargets{
			Labels: []string{
				label.Name,
			},
		},
	}
	require.NoError(t, ds.ApplyPackSpecs(ctx, []*mdmlab.PackSpec{pack}))
	require.NoError(t, ds.RecordLabelQueryExecutions(ctx, h1, map[uint]*bool{label.ID: ptr.Bool(true)}, mockClock.Now(), false))

	packs, err := ds.ListPacksForHost(ctx, h1.ID)
	require.Nil(t, err)
	if assert.Len(t, packs, 1) {
		assert.Equal(t, "foo_pack", packs[0].Name)
	}
}
