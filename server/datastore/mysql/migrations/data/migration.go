package data

import "github.com:it-laborato/MDM_Lab/server/goose"

var MigrationClient = goose.New("migration_status_data", goose.MySqlDialect{})
