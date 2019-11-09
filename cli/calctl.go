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

	app := cli.App{
		Name:    "cal-ctl",
		Version: version,
		Commands: []cli.Command{
			cmd.ShowTypes,
			cmd.Fetch,
			cmd.List,
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
