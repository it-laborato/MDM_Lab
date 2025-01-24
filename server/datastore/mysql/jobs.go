package mysql

import (
	"context"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewJob(ctx context.Context, job *mdmlab.Job) (*mdmlab.Job, error) {
	query := `
INSERT INTO jobs (
    name,
    args,
    state,
    retries,
    error,
    not_before
)
VALUES (?, ?, ?, ?, ?, COALESCE(?, NOW()))
`
	var notBefore *time.Time
	if !job.NotBefore.IsZero() {
		notBefore = &job.NotBefore
	}
	result, err := ds.writer(ctx).ExecContext(ctx, query, job.Name, job.Args, job.State, job.Retries, job.Error, notBefore)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	job.ID = uint(id) //nolint:gosec // dismiss G115

	return job, nil
}

func (ds *Datastore) GetQueuedJobs(ctx context.Context, maxNumJobs int, now time.Time) ([]*mdmlab.Job, error) {
	query := `
SELECT
    id, created_at, updated_at, name, args, state, retries, error, not_before
FROM
    jobs
WHERE
    state = ? AND
    not_before <= ?
ORDER BY
    updated_at ASC
LIMIT ?
`

	if now.IsZero() {
		now = time.Now().UTC()
	}

	var jobs []*mdmlab.Job
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &jobs, query, mdmlab.JobStateQueued, now, maxNumJobs)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (ds *Datastore) UpdateJob(ctx context.Context, id uint, job *mdmlab.Job) (*mdmlab.Job, error) {
	query := `
UPDATE jobs
SET
    state = ?,
    retries = ?,
    error = ?,
    not_before = COALESCE(?, NOW())
WHERE
    id = ?
`
	var notBefore *time.Time
	if !job.NotBefore.IsZero() {
		notBefore = &job.NotBefore
	}
	_, err := ds.writer(ctx).ExecContext(ctx, query, job.State, job.Retries, job.Error, notBefore, id)
	if err != nil {
		return nil, err
	}

	return job, nil
}
