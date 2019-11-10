package main

import (
	"fmt"
	"github.com/mariusor/esports-calendar/cmd"
	"github.com/urfave/cli"
	"os"
)

var version = "(unknown)"

func main() {
	var err error

	ctl := cli.App{
		Name:    "ecalsrv",
		Version: version,
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
