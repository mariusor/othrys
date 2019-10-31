package main

import (
	"github.com/mariusor/esports-calendar/cmd"
	"github.com/urfave/cli"
	"os"
)

var version = "(unknown)"

func main() {
	var err error
	app := cli.App{}
	app.Name = "cal-ctl"
	app.Version = version
	app.Before = func(c *cli.Context) error {
		return nil
	}

	app.Commands = []cli.Command{
		cmd.ShowTypes,
		cmd.Fetch,
	}
	err = app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
