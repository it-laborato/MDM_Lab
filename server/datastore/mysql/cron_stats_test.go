package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

type testCronStats struct {
	mdmlab.CronStats
	// Errors is a JSON string containing any errors encountered during the run.
	Errors sql.NullString `db:"errors"`
}

func TestInsertUpdateCronStats(t *testing.T) {
	const (
		scheduleName = "test_sched"
		instanceID   = "test_instance"
	)
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	id, err := ds.InsertCronStats(ctx, mdmlab.CronStatsTypeScheduled, scheduleName, instanceID, mdmlab.CronStatsStatusPending)
	require.NoError(t, err)

	res, err := ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, id, res[0].ID)
	require.Equal(t, mdmlab.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, mdmlab.CronStatsStatusPending, res[0].Status)

	err = ds.UpdateCronStats(ctx, id, mdmlab.CronStatsStatusCompleted, &mdmlab.CronScheduleErrors{
		"some_job":       errors.New("some error"),
		"some_other_job": errors.New("some other error"),
	})
	require.NoError(t, err)

	res, err = ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, id, res[0].ID)
	require.Equal(t, mdmlab.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, mdmlab.CronStatsStatusCompleted, res[0].Status)

	var stats []testCronStats
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	// Make sure we got valid JSON back.
	var actualMap map[string]string
	err = json.Unmarshal([]byte(stats[0].Errors.String), &actualMap)
	require.NoError(t, err)

	// Compare the error JSON with the expected object.
	expectedJSON := `{"some_job": "some error", "some_other_job": "some other error"}`
	var expectedMap map[string]string
	err = json.Unmarshal([]byte(expectedJSON), &expectedMap)
	require.NoError(t, err)
	require.Equal(t, actualMap, expectedMap)
}

func TestGetLatestCronStats(t *testing.T) {
	const (
		scheduleName = "test_sched"
		instanceID   = "test_instance"
	)
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	insertTestCS := func(name string, statsType mdmlab.CronStatsType, status mdmlab.CronStatsStatus, createdAt time.Time) {
		stmt := `INSERT INTO cron_stats (stats_type, name, instance, status, created_at) VALUES (?, ?, ?, ?, ?)`
		_, err := ds.writer(ctx).ExecContext(ctx, stmt, statsType, name, instanceID, status, createdAt)
		require.NoError(t, err)
	}

	then := time.Now().UTC().Truncate(time.Second).Add(-24 * time.Hour)

	// insert two "scheduled" stats
	insertTestCS(scheduleName, mdmlab.CronStatsTypeScheduled, mdmlab.CronStatsStatusPending, then.Add(2*time.Minute))
	insertTestCS(scheduleName, mdmlab.CronStatsTypeScheduled, mdmlab.CronStatsStatusCompleted, then.Add(1*time.Minute))

	// most recent record is returned for "scheduled" stats type
	res, err := ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, mdmlab.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, mdmlab.CronStatsStatusPending, res[0].Status)
	require.Equal(t, then.Add(2*time.Minute), res[0].CreatedAt)

	// insert two "triggered" stats
	insertTestCS(scheduleName, mdmlab.CronStatsTypeTriggered, mdmlab.CronStatsStatusCompleted, then.Add(2*time.Hour))
	insertTestCS(scheduleName, mdmlab.CronStatsTypeTriggered, mdmlab.CronStatsStatusCompleted, then.Add(1*time.Hour))

	// most recent record is returned for both "scheduled" stats type and "triggered" stats type
	res, err = ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, mdmlab.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, mdmlab.CronStatsStatusPending, res[0].Status)
	require.Equal(t, then.Add(2*time.Minute), res[0].CreatedAt)
	require.Equal(t, mdmlab.CronStatsTypeTriggered, res[1].StatsType)
	require.Equal(t, mdmlab.CronStatsStatusCompleted, res[1].Status)
	require.Equal(t, then.Add(2*time.Hour), res[1].CreatedAt)

	// insert some other stats that shouldn't be returned
	insertTestCS(scheduleName, mdmlab.CronStatsTypeScheduled, mdmlab.CronStatsStatusExpired, then.Add(3*time.Hour))    // expired status shouldn't be returned
	insertTestCS(scheduleName, mdmlab.CronStatsTypeTriggered, mdmlab.CronStatsStatusExpired, then.Add(3*time.Hour))    // expired status shouldn't be returned
	insertTestCS(scheduleName, mdmlab.CronStatsTypeScheduled, mdmlab.CronStatsStatusCanceled, then.Add(4*time.Hour))   // canceled status shouldn't be returned
	insertTestCS(scheduleName, mdmlab.CronStatsTypeTriggered, mdmlab.CronStatsStatusCanceled, then.Add(4*time.Hour))   // canceled status shouldn't be returned
	insertTestCS("schedule_1337", mdmlab.CronStatsTypeTriggered, mdmlab.CronStatsStatusPending, then.Add(5*time.Hour)) // different name shouldn't be returned

	// most recent record is returned for both "scheduled" stats type and "triggered" stats type
	res, err = ds.GetLatestCronStats(ctx, scheduleName)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, mdmlab.CronStatsTypeScheduled, res[0].StatsType)
	require.Equal(t, mdmlab.CronStatsStatusPending, res[0].Status)
	require.Equal(t, then.Add(2*time.Minute), res[0].CreatedAt)
	require.Equal(t, mdmlab.CronStatsTypeTriggered, res[1].StatsType)
	require.Equal(t, mdmlab.CronStatsStatusCompleted, res[1].Status)
	require.Equal(t, then.Add(2*time.Hour), res[1].CreatedAt)
}

func TestCleanupCronStats(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)
	now := time.Now().UTC().Truncate(time.Second)
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
	name := "test_sched"
	instance := "test_instance"

	cases := []struct {
		createdAt               time.Time
		status                  mdmlab.CronStatsStatus
		shouldCleanupMaxPending bool
		shouldCleanupMaxAge     bool
	}{
		{
			createdAt:               now,
			status:                  mdmlab.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now,
			status:                  mdmlab.CronStatsStatusPending,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-1 * time.Hour),
			status:                  mdmlab.CronStatsStatusPending,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-2 * time.Hour),
			status:                  mdmlab.CronStatsStatusExpired,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-3 * time.Hour),
			status:                  mdmlab.CronStatsStatusPending,
			shouldCleanupMaxPending: true,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               now.Add(-3 * time.Hour),
			status:                  mdmlab.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               twoDaysAgo.Add(1 * time.Hour),
			status:                  mdmlab.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     false,
		},
		{
			createdAt:               twoDaysAgo.Add(-1 * time.Hour),
			status:                  mdmlab.CronStatsStatusCompleted,
			shouldCleanupMaxPending: false,
			shouldCleanupMaxAge:     true,
		},
	}

	for _, c := range cases {
		stmt := `INSERT INTO cron_stats (stats_type, name, instance, status, created_at) VALUES (?, ?, ?, ?, ?)`
		_, err := ds.writer(ctx).ExecContext(ctx, stmt, mdmlab.CronStatsTypeScheduled, name, instance, c.status, c.createdAt)
		require.NoError(t, err)
	}

	var stats []testCronStats
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases))
	for i, s := range stats {
		require.Equal(t, cases[i].createdAt, s.CreatedAt)
		require.Equal(t, cases[i].status, s.Status)
	}

	err = ds.CleanupCronStats(ctx)
	require.NoError(t, err)

	stats = []testCronStats{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases)-1) // case[7] was deleted because it exceeded max age
	for i, c := range cases {
		if i >= len(stats) {
			require.True(t, c.shouldCleanupMaxAge)
			break
		}
		if c.shouldCleanupMaxPending {
			require.Equal(t, mdmlab.CronStatsStatusExpired, stats[i].Status)
		} else {
			require.Equal(t, c.status, stats[i].Status)
		}
	}
}

func TestUpdateAllCronStatsForInstance(t *testing.T) {
	ctx := context.Background()
	ds := CreateMySQLDS(t)

	cases := []struct {
		instance     string
		schedName    string
		status       mdmlab.CronStatsStatus
		shouldUpdate bool
	}{
		{
			instance:     "inst1",
			schedName:    "sched1",
			status:       mdmlab.CronStatsStatusCompleted,
			shouldUpdate: false,
		},
		{
			instance:     "inst1",
			schedName:    "sched1",
			status:       mdmlab.CronStatsStatusPending,
			shouldUpdate: true,
		},
		{
			instance:     "inst1",
			schedName:    "sched2",
			status:       mdmlab.CronStatsStatusExpired,
			shouldUpdate: false,
		},
		{
			instance:     "inst1",
			schedName:    "sched2",
			status:       mdmlab.CronStatsStatusPending,
			shouldUpdate: true,
		},
		{
			instance:     "inst2",
			schedName:    "sched1",
			status:       mdmlab.CronStatsStatusPending,
			shouldUpdate: false,
		},
		{
			instance:     "inst2",
			schedName:    "sched2",
			status:       mdmlab.CronStatsStatusPending,
			shouldUpdate: false,
		},
	}

	for _, c := range cases {
		stmt := `INSERT INTO cron_stats (stats_type, name, instance, status) VALUES (?, ?, ?, ?)`
		_, err := ds.writer(ctx).ExecContext(ctx, stmt, mdmlab.CronStatsTypeScheduled, c.schedName, c.instance, c.status)
		require.NoError(t, err)
	}

	var stats []testCronStats
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases))
	for i, s := range stats {
		require.Equal(t, cases[i].schedName, s.Name)
		require.Equal(t, cases[i].instance, s.Instance)
		require.Equal(t, cases[i].status, s.Status)
	}

	err = ds.UpdateAllCronStatsForInstance(ctx, "inst1", mdmlab.CronStatsStatusPending, mdmlab.CronStatsStatusCanceled)
	require.NoError(t, err)

	stats = []testCronStats{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &stats, `SELECT * FROM cron_stats ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, stats, len(cases))
	for i, c := range cases {
		s := stats[i]
		require.Equal(t, c.instance, s.Instance)
		require.Equal(t, c.schedName, s.Name)
		if c.shouldUpdate {
			require.Equal(t, mdmlab.CronStatsStatusCanceled, s.Status)
		} else {
			require.Equal(t, c.status, s.Status)
		}
	}
}
