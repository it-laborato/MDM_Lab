package data

import (
	"database/sql"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

func init() {
	MigrationClient.AddMigration(Up_20230525175650, Down_20230525175650)
}

func Up_20230525175650(tx *sql.Tx) error {
	label := mdmlab.Label{
		Name:        "chrome",
		Query:       "select 1 from os_version where platform = 'chrome';",
		Description: "All Chrome hosts",
		LabelType:   mdmlab.LabelTypeBuiltIn,
	}

	sql := `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES (?, ?, ?, ?, ?)
`
	_, err := tx.Exec(sql, label.Name, label.Description, label.Query, label.Platform, label.LabelType)
	if err != nil {
		return err
	}
	return nil
}

func Down_20230525175650(tx *sql.Tx) error {
	return nil
}
