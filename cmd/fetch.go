package cmd

import (
	"github.com/mariusor/esports-calendar/calendar"
	"github.com/urfave/cli"
	"time"
)

var now = time.Now()

var defaultCalendars = calendar.DefaultValues // all
var defaultDuration = time.Hour * 336                                                 // 2 weeks
var defaultStartTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

var Fetch = cli.Command{
	Name:  "fetch",
	Usage: "Fetches calendar events",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "calendar",
			Usage: "Which calendars to load",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Output debug messages",
		},
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Don't persist events",
		},
		&cli.StringFlag{
			Name:  "start",
			Usage: "Date at which to start",
			Value: now.Format("2006-01-02"),
		},
		&cli.DurationFlag{
			Name:  "end",
			Usage: "Date interval to check",
			Value: defaultDuration,
		},
	},
	Action: fetchCalendars,
}

func fetchCalendars(c *cli.Context) error {
	types := calendar.GetTypes(c.StringSlice("calendar")...)
	f := calendar.New(types...)
	start := time.Now().Add(-1 * defaultDuration)
	if sf := c.String("start"); len(sf) > 0 {
		if sfp, err := time.Parse("2006-01-02", sf); err == nil {
			start = sfp
		} 
	}
	f.Load(start, defaultDuration)
	return nil
}
