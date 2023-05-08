package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"

	"github.com/mariusor/esports-calendar/internal/cmd"
)

func main() {
	var err error

	ctl := cli.App{
		Name:    fmt.Sprintf("%sctl", cmd.AppName),
		Version: cmd.AppVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "The path for storage",
				Value: cmd.DataPath(),
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Output debug messages",
			},
		},
		Commands: []cli.Command{
			cmd.ShowTypesCmd,
			cmd.FetchCmd,
			cmd.ListCmd,
			cmd.AuthorizeCmd,
			cmd.PostCmd,
		},
	}

	err = ctl.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
