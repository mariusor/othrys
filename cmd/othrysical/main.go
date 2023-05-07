package main

import (
	"fmt"
	"github.com/mariusor/esports-calendar/internal/cmd"
	"os"

	"github.com/urfave/cli"
)

var version = "(unknown)"

func main() {
	var err error

	ctl := cli.App{
		Name:    fmt.Sprintf("%sical", cmd.AppName),
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "Set storage path",
				Value: cmd.DataPath(),
			},
		},
		Commands: []cli.Command{
			cmd.Server,
		},
	}

	err = ctl.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
