package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:      "logout",
		Usage:     "Log out of MDMlab",
		UsageText: `mdmlabctl logout [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			mdmlab, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			if err := mdmlab.Logout(); err != nil {
				return fmt.Errorf("error logging out: %w", err)
			}

			configPath, context := c.String("config"), c.String("context")

			if err := setConfigValue(configPath, context, "token", ""); err != nil {
				return fmt.Errorf("error setting token for the current context: %w", err)
			}

			fmt.Printf("[+] MDMlab logout successful and local token cleared!\n")

			return nil
		},
	}
}
