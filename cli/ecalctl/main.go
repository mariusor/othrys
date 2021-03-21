package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"

	"github.com/mariusor/esports-calendar/cmd"
)

var version = "(unknown)"

func main() {
	var err error

	ctl := cli.App{
		Name:    "ecalctl",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "The path for storage",
				Value: ".",
			},
		},
		Commands: []cli.Command{
			cmd.ShowTypes,
			cmd.Fetch,
			cmd.List,
			cmd.Toot,
		},
	}

	err = ctl.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
