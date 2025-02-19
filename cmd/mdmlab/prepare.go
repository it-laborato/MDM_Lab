package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/datastore/mysql"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/mixer/clock"
	"github.com/spf13/cobra"
)

func createPrepareCmd(configManager config.Manager) *cobra.Command {
	prepareCmd := &cobra.Command{
		Use:   "prepare",
		Short: "Subcommands for initializing MDMlab infrastructure",
		Long: `
Subcommands for initializing MDMlab infrastructure

To setup MDMlab infrastructure, use one of the available commands.
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help() //nolint:errcheck
		},
	}

	noPrompt := false
	// Whether to enable developer options
	dev := false

	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Given correct database configurations, prepare the databases for use",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if dev {
				applyDevFlags(&config)
				noPrompt = true
			}

			ds, err := mysql.New(config.Mysql, clock.C)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			status, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			switch status.StatusCode {
			case mdmlab.NoMigrationsCompleted:
				// OK
			case mdmlab.AllMigrationsCompleted:
				fmt.Println("Migrations already completed. Nothing to do.")
				return
			case mdmlab.SomeMigrationsCompleted:
				if !noPrompt {
					fmt.Printf("################################################################################\n"+
						"# WARNING:\n"+
						"#   This will perform MDMlab database migrations. Please back up your data before\n"+
						"#   continuing.\n"+
						"#\n"+
						"#   Missing migrations: tables=%v, data=%v.\n"+
						"#\n"+
						"#   Press Enter to continue, or Control-c to exit.\n"+
						"################################################################################\n",
						status.MissingTable, status.MissingData)
					bufio.NewScanner(os.Stdin).Scan()
				}
			case mdmlab.UnknownMigrations:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your MDMlab database has unrecognized migrations. This could happen when\n"+
					"#   running an older version of MDMlab on a newer migrated database.\n"+
					"#\n"+
					"#   Unknown migrations: tables=%v, data=%v.\n"+
					"################################################################################\n",
					status.UnknownTable, status.UnknownData)
				if dev {
					os.Exit(1)
				}
			}

			if err := ds.MigrateTables(cmd.Context()); err != nil {
				initFatal(err, "migrating db schema")
			}

			if err := ds.MigrateData(cmd.Context()); err != nil {
				initFatal(err, "migrating builtin data")
			}

			fmt.Println("Migrations completed.")
		},
	}

	dbCmd.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "disable prompting before migrations (for use in scripts)")
	dbCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

	prepareCmd.AddCommand(dbCmd)
	return prepareCmd
}
