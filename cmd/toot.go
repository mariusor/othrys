package cmd

import (
	"github.com/urfave/cli"
)

var Toot = cli.Command{
	Name:  "toot",
	Usage: "Post events to mastodon",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "calendar",
			Usage: "Which calendars to list",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Output debug messages",
		},
		&cli.StringFlag{
			Name:  "start",
			Usage: "Date at which to start",
			Value: defaultStartTime.Format("2006-01-02"),
		},
		&cli.DurationFlag{
			Name:  "end",
			Usage: "Date interval to check",
			Value: defaultDuration,
		},
	},
	Action: listCalendars,
}
