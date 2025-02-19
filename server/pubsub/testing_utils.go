package pubsub

import (
	"testing"

	"github.com/it-laborato/MDM_Lab/server/datastore/redis/redistest"
	"github.com/go-kit/log"
)

func SetupRedisForTest(t *testing.T, cluster, readReplica bool) *redisQueryResults {
	const dupResults = false
	pool := redistest.SetupRedis(t, "zz", cluster, false, readReplica)
	return NewRedisQueryResults(pool, dupResults, log.NewNopLogger())
}
