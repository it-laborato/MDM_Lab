package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/pkg/optjson"
	"github.com/it-laborato/MDM_Lab/server/datastore/mysql"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	apple_mdm "github.com/it-laborato/MDM_Lab/server/mdm/apple"
	nanomdm_push "github.com/it-laborato/MDM_Lab/server/mdm/nanomdm/push"
	mock "github.com/it-laborato/MDM_Lab/server/mock/mdm"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPusher struct {
	response *nanomdm_push.Response
	err      error
}

func (m mockPusher) Push(context.Context, []string) (map[string]*nanomdm_push.Response, error) {
	var res map[string]*nanomdm_push.Response
	if m.response != nil {
		res = map[string]*nanomdm_push.Response{
			m.response.Id: m.response,
		}
	}
	return res, m.err
}

func TestAppleMDM(t *testing.T) {
	ctx := context.Background()

	// use a real mysql datastore so that the test does not rely so much on
	// specific internals (sequence and number of calls, etc.). The MDM storage
	// and pusher are mocks.
	ds := mysql.CreateMySQLDS(t)
	// call TruncateTables immediately as a DB migation may have created jobs
	mysql.TruncateTables(t, ds)

	mdmStorage, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	// nopLog := kitlog.NewNopLogger()
	// use this to debug/verify details of calls
	nopLog := kitlog.NewJSONLogger(os.Stdout)

	testOrgName := "mdmlab-test"

	createEnrolledHost := func(t *testing.T, i int, teamID *uint, depAssignedToMDMlab bool) *mdmlab.Host {
		// create the host
		h, err := ds.NewHost(ctx, &mdmlab.Host{
			Hostname:       fmt.Sprintf("test-host%d-name", i),
			OsqueryHostID:  ptr.String(fmt.Sprintf("osquery-%d", i)),
			NodeKey:        ptr.String(fmt.Sprintf("nodekey-%d", i)),
			UUID:           uuid.New().String(),
			Platform:       "darwin",
			HardwareSerial: fmt.Sprintf("serial-%d", i),
			TeamID:         teamID,
		})
		require.NoError(t, err)

		// create the nano_device and enrollment
		var abmTokenID uint
		mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_devices (id, serial_number, authenticate) VALUES (?, ?, ?)`, h.UUID, h.HardwareSerial, "test")
			if err != nil {
				return err
			}
			_, err = q.ExecContext(ctx, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex)
				VALUES (?, ?, ?, ?, ?, ?)`, h.UUID, h.UUID, "device", "topic", "push_magic", "token_hex")
			if err != nil {
				return err
			}

			encTok := uuid.NewString()
			abmToken, err := ds.InsertABMToken(ctx, &mdmlab.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
			abmTokenID = abmToken.ID

			return err
		})
		if depAssignedToMDMlab {
			err := ds.UpsertMDMAppleHostDEPAssignments(ctx, []mdmlab.Host{*h}, abmTokenID)
			require.NoError(t, err)
		}
		err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "http://example.com", depAssignedToMDMlab, mdmlab.WellKnownMDMMDMlab, "")
		require.NoError(t, err)
		return h
	}

	getEnqueuedCommandTypes := func(t *testing.T) []string {
		var commands []string
		mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &commands, "SELECT request_type FROM nano_commands")
		})
		return commands
	}

	enableManualRelease := func(t *testing.T, teamID *uint) {
		if teamID == nil {
			enableAppCfg := func(enable bool) {
				ac, err := ds.AppConfig(ctx)
				require.NoError(t, err)
				ac.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(enable)
				err = ds.SaveAppConfig(ctx, ac)
				require.NoError(t, err)
			}

			enableAppCfg(true)
			t.Cleanup(func() { enableAppCfg(false) })
		} else {
			enableTm := func(enable bool) {
				tm, err := ds.Team(ctx, *teamID)
				require.NoError(t, err)
				tm.Config.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(enable)
				_, err = ds.SaveTeam(ctx, tm)
				require.NoError(t, err)
			}

			enableTm(true)
			t.Cleanup(func() { enableTm(false) })
		}
	}

	t.Run("no-op with nil commander", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		// create a host and enqueue the job
		h := createEnrolledHost(t, 1, nil, true)
		err := QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", nil, "", false)
		require.NoError(t, err)

		// run the worker, should mark the job as done
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)
		require.Empty(t, jobs)
	})

	t.Run("fails with unknown task", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		// create a host and enqueue the job
		h := createEnrolledHost(t, 1, nil, true)
		err := QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMTask("no-such-task"), h.UUID, "darwin", nil, "", false)
		require.NoError(t, err)

		// run the worker, should mark the job as failed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Time{})
		require.NoError(t, err)
		require.Len(t, jobs, 1)
		require.Contains(t, jobs[0].Error, "unknown task: no-such-task")
		require.Equal(t, mdmlab.JobStateQueued, jobs[0].State)
		require.Equal(t, 1, jobs[0].Retries)
	})

	t.Run("installs default manifest", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		h := createEnrolledHost(t, 1, nil, true)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		// use "" instead of "darwin" as platform to test a queued job after the upgrade to iOS/iPadOS support.
		err := QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "", nil, "", false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)

		// there is no post-DEP release device job anymore
		require.Len(t, jobs, 0)

		require.ElementsMatch(t, []string{"InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))
	})

	t.Run("installs default manifest, manual release", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		t.Cleanup(func() { mysql.TruncateTables(t, ds) })

		h := createEnrolledHost(t, 1, nil, true)
		enableManualRelease(t, nil)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err = QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", nil, "", false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)

		// there is no post-DEP release device job pending
		require.Empty(t, jobs)

		require.ElementsMatch(t, []string{"InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))
	})

	t.Run("installs custom bootstrap manifest", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		h := createEnrolledHost(t, 1, nil, true)
		err := ds.InsertMDMAppleBootstrapPackage(ctx, &mdmlab.MDMAppleBootstrapPackage{
			Name:   "custom-bootstrap",
			TeamID: 0, // no-team
			Bytes:  []byte("test"),
			Sha256: []byte("test"),
			Token:  "token",
		}, nil)
		require.NoError(t, err)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err = QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", nil, "", false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)

		// the post-DEP release device job is not queued anymore
		require.Len(t, jobs, 0)

		require.ElementsMatch(t, []string{"InstallEnterpriseApplication", "InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))

		ms, err := ds.GetHostMDMMacOSSetup(ctx, h.ID)
		require.NoError(t, err)
		require.Equal(t, "custom-bootstrap", ms.BootstrapPackageName)
	})

	t.Run("installs custom bootstrap manifest of a team", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		tm, err := ds.NewTeam(ctx, &mdmlab.Team{Name: "test"})
		require.NoError(t, err)

		h := createEnrolledHost(t, 1, &tm.ID, true)
		err = ds.InsertMDMAppleBootstrapPackage(ctx, &mdmlab.MDMAppleBootstrapPackage{
			Name:   "custom-team-bootstrap",
			TeamID: tm.ID,
			Bytes:  []byte("test"),
			Sha256: []byte("test"),
			Token:  "token",
		}, nil)
		require.NoError(t, err)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err = QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", &tm.ID, "", false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)

		// the post-DEP release device job is not queued anymore
		require.Len(t, jobs, 0)

		require.ElementsMatch(t, []string{"InstallEnterpriseApplication", "InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))

		ms, err := ds.GetHostMDMMacOSSetup(ctx, h.ID)
		require.NoError(t, err)
		require.Equal(t, "custom-team-bootstrap", ms.BootstrapPackageName)
	})

	t.Run("installs custom bootstrap manifest of a team, manual release", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		t.Cleanup(func() { mysql.TruncateTables(t, ds) })

		tm, err := ds.NewTeam(ctx, &mdmlab.Team{Name: "test"})
		require.NoError(t, err)
		enableManualRelease(t, &tm.ID)

		h := createEnrolledHost(t, 1, &tm.ID, true)
		err = ds.InsertMDMAppleBootstrapPackage(ctx, &mdmlab.MDMAppleBootstrapPackage{
			Name:   "custom-team-bootstrap",
			TeamID: tm.ID,
			Bytes:  []byte("test"),
			Sha256: []byte("test"),
			Token:  "token",
		}, nil)
		require.NoError(t, err)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err = QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", &tm.ID, "", false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)

		// there is no post-DEP release device job pending
		require.Empty(t, jobs)

		require.ElementsMatch(t, []string{"InstallEnterpriseApplication", "InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))

		ms, err := ds.GetHostMDMMacOSSetup(ctx, h.ID)
		require.NoError(t, err)
		require.Equal(t, "custom-team-bootstrap", ms.BootstrapPackageName)
	})

	t.Run("unknown enroll reference", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		h := createEnrolledHost(t, 1, nil, true)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err := QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", nil, "abcd", false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Time{})
		require.NoError(t, err)
		require.Len(t, jobs, 1)
		require.Contains(t, jobs[0].Error, "MDMIdPAccount with uuid abcd was not found")
		require.Equal(t, mdmlab.JobStateQueued, jobs[0].State)
		require.Equal(t, 1, jobs[0].Retries)
	})

	t.Run("enroll reference but SSO disabled", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		err := ds.InsertMDMIdPAccount(ctx, &mdmlab.MDMIdPAccount{
			Username: "test",
			Fullname: "test",
			Email:    "test@example.com",
		})
		require.NoError(t, err)

		idpAcc, err := ds.GetMDMIdPAccountByEmail(ctx, "test@example.com")
		require.NoError(t, err)

		h := createEnrolledHost(t, 1, nil, true)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err = QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", nil, idpAcc.UUID, false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)

		// the post-DEP release device job is not queued anymore
		require.Len(t, jobs, 0)

		// confirm that AccountConfiguration command was not enqueued
		require.ElementsMatch(t, []string{"InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))
	})

	t.Run("enroll reference with SSO enabled", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		err := ds.InsertMDMIdPAccount(ctx, &mdmlab.MDMIdPAccount{
			Username: "test",
			Fullname: "test",
			Email:    "test@example.com",
		})
		require.NoError(t, err)

		idpAcc, err := ds.GetMDMIdPAccountByEmail(ctx, "test@example.com")
		require.NoError(t, err)

		tm, err := ds.NewTeam(ctx, &mdmlab.Team{Name: "test"})
		require.NoError(t, err)
		tm, err = ds.Team(ctx, tm.ID)
		require.NoError(t, err)
		tm.Config.MDM.MacOSSetup.EnableEndUserAuthentication = true
		_, err = ds.SaveTeam(ctx, tm)
		require.NoError(t, err)

		h := createEnrolledHost(t, 1, &tm.ID, true)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err = QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", &tm.ID, idpAcc.UUID, false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)

		// the post-DEP release device job is not queued anymore
		require.Len(t, jobs, 0)

		require.ElementsMatch(t, []string{"InstallEnterpriseApplication", "AccountConfiguration"}, getEnqueuedCommandTypes(t))
	})

	t.Run("installs mdmlabd for manual enrollments", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		h := createEnrolledHost(t, 1, nil, true)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err := QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostManualEnrollmentTask, h.UUID, "darwin", nil, "", false)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
		require.NoError(t, err)
		require.Empty(t, jobs)
		require.ElementsMatch(t, []string{"InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))
	})

	t.Run("use worker for automatic release", func(t *testing.T) {
		mysql.SetTestABMAssets(t, ds, testOrgName)
		defer mysql.TruncateTables(t, ds)

		h := createEnrolledHost(t, 1, nil, true)

		mdmWorker := &AppleMDM{
			Datastore: ds,
			Log:       nopLog,
			Commander: apple_mdm.NewMDMAppleCommander(mdmStorage, mockPusher{}),
		}
		w := NewWorker(ds, nopLog)
		w.Register(mdmWorker)

		err := QueueAppleMDMJob(ctx, ds, nopLog, AppleMDMPostDEPEnrollmentTask, h.UUID, "darwin", nil, "", true)
		require.NoError(t, err)

		// run the worker, should succeed
		err = w.ProcessJobs(ctx)
		require.NoError(t, err)

		// ensure the job's not_before allows it to be returned if it were to run
		// again
		time.Sleep(time.Second)

		require.ElementsMatch(t, []string{"InstallEnterpriseApplication"}, getEnqueuedCommandTypes(t))

		// the release device job got enqueued
		jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().Add(time.Minute)) // release job is always added with a delay
		require.NoError(t, err)
		require.Len(t, jobs, 1)
		require.Equal(t, mdmlab.JobStateQueued, jobs[0].State)
		require.Equal(t, appleMDMJobName, jobs[0].Name)
		require.Contains(t, string(*jobs[0].Args), AppleMDMPostDEPReleaseDeviceTask)
	})
}

func TestGetSignedURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	meta := &mdmlab.MDMAppleBootstrapPackage{
		Sha256: []byte{1, 2, 3},
	}

	var data []byte
	buf := bytes.NewBuffer(data)
	logger := kitlog.NewLogfmtLogger(buf)
	a := &AppleMDM{Log: logger}

	// S3 not configured
	assert.Empty(t, a.getSignedURL(ctx, meta))
	assert.Empty(t, buf.String())

	// Signer not configured
	mockStore := &mock.MDMBootstrapPackageStore{}
	a.BootstrapPackageStore = mockStore
	mockStore.SignFunc = func(ctx context.Context, fileID string) (string, error) {
		return "bozo", mdmlab.ErrNotConfigured
	}
	assert.Empty(t, a.getSignedURL(ctx, meta))
	assert.Empty(t, buf.String())

	// Test happy path
	mockStore.SignFunc = func(ctx context.Context, fileID string) (string, error) {
		return "signed", nil
	}
	mockStore.ExistsFunc = func(ctx context.Context, packageID string) (bool, error) {
		assert.Equal(t, "010203", packageID)
		return true, nil
	}
	assert.Equal(t, "signed", a.getSignedURL(ctx, meta))
	assert.Empty(t, buf.String())
	assert.True(t, mockStore.SignFuncInvoked)
	assert.True(t, mockStore.ExistsFuncInvoked)
	mockStore.SignFuncInvoked = false
	mockStore.ExistsFuncInvoked = false

	// Test error -- sign failed
	mockStore.SignFunc = func(ctx context.Context, fileID string) (string, error) {
		return "", errors.New("test error")
	}
	assert.Empty(t, a.getSignedURL(ctx, meta))
	assert.Contains(t, buf.String(), "test error")
	assert.True(t, mockStore.SignFuncInvoked)
	assert.False(t, mockStore.ExistsFuncInvoked)

}
