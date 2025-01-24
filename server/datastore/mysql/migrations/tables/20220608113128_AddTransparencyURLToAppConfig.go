package tables

import (
	"database/sql"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220608113128, Down_20220608113128)
}

func Up_20220608113128(tx *sql.Tx) error {
	err := updateAppConfigJSON(tx, func(config *mdmlab.AppConfig) error {
		if config.MDMlabDesktop.TransparencyURL != "" {
			return errors.New("unexpected transparency_url value in app_config_json")
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func Down_20220608113128(tx *sql.Tx) error {
	return nil
}
