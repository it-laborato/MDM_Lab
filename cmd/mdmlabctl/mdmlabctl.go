package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path"
	"runtime"
	"time"

	eemdmlabctl "github.com/it-laborato/MDM_Lab/ee/mdmlabctl"
	"github.com/it-laborato/MDM_Lab/server/version"
	"github.com/urfave/cli/v2"
)

const (
	defaultFileMode = 0o600
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	app := createApp(os.Stdin, os.Stdout, os.Stderr, exitErrHandler)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stdout, "Error: %+v\n", err)
		os.Exit(1)
	}
}

// exitErrHandler implements cli.ExitErrHandlerFunc. If there is an error, prints it to stderr and exits with status 1.
func exitErrHandler(c *cli.Context, err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(c.App.ErrWriter, "Error: %+v\n", err)

	if errors.Is(err, fs.ErrPermission) {
		switch runtime.GOOS {
		case "darwin", "linux":
			fmt.Fprintf(c.App.ErrWriter, "\nThis error can usually be resolved by fixing the permissions on the %s directory, or re-running this command with sudo.\n", path.Dir(c.String("config")))
		case "windows":
			fmt.Fprintf(c.App.ErrWriter, "\nThis error can usually be resolved by fixing the permissions on the %s directory, or re-running this command with 'Run as administrator'.\n", path.Dir(c.String("config")))
		}
	}
	cli.OsExiter(1)
}

func createApp(
	reader io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	exitErrHandler cli.ExitErrHandlerFunc,
) *cli.App {
	app := cli.NewApp()
	app.Name = "mdmlabctl"
	app.Usage = "CLI for operating MDMlab"
	app.Version = version.Version().Version
	app.ExitErrHandler = exitErrHandler
	cli.VersionPrinter = func(c *cli.Context) {
		version.PrintFull()
	}
	app.Reader = reader
	app.Writer = stdout
	app.ErrWriter = stderr

	app.Commands = []*cli.Command{
		apiCommand(),
		applyCommand(),
		deleteCommand(),
		setupCommand(),
		loginCommand(),
		logoutCommand(),
		queryCommand(),
		getCommand(),
		{
			Name:  "config",
			Usage: "Modify MDMlab server connection settings",
			Subcommands: []*cli.Command{
				configSetCommand(),
				configGetCommand(),
			},
		},
		convertCommand(),
		goqueryCommand(),
		userCommand(),
		debugCommand(),
		previewCommand(),
		eemdmlabctl.UpdatesCommand(),
		hostsCommand(),
		vulnerabilityDataStreamCommand(),
		packageCommand(),
		generateCommand(),
		{
			// It's become common for folks to unintentionally install mdmlabctl when they actually
			// need the MDMlab server. This is hopefully a more helpful error message.
			Name:  "prepare",
			Usage: "This is not the binary you're looking for. Please use the mdmlab server binary for prepare commands.",
			Action: func(c *cli.Context) error {
				return errors.New("This is not the binary you're looking for. Please use the mdmlab server binary for prepare commands.")
			},
		},
		triggerCommand(),
		mdmCommand(),
		upgradePacksCommand(),
		runScriptCommand(),
		gitopsCommand(),
	}
	return app
}
