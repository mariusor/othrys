package cmd

import (
	calendar "github.com/mariusor/esports-calendar"
	"github.com/mariusor/esports-calendar/app/liquid"
	"github.com/mariusor/esports-calendar/app/plusforward"
	"github.com/urfave/cli"
	"time"
)

var now = time.Now()

var defaultCalendars = cli.StringSlice{liquid.LabelTeamLiquid, plusforward.LabelPlusForward} // all
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
	types := c.StringSlice("calendar")
	fetchTypes := make([]string, 0)
	for _, cal := range types {
		if cal == liquid.LabelTeamLiquid {
			 fetchTypes = append(fetchTypes, liquid.ValidTypes[:]...)
		} else if cal == plusforward.LabelPlusForward {
			 fetchTypes = append(fetchTypes, plusforward.ValidTypes[:]...)
		} else {
			fetchTypes = append(fetchTypes, cal)
		}
	}
	
	f := calendar.New(fetchTypes...)
	start := time.Now().Add(-1 * defaultDuration)
	if sf := c.String("start"); len(sf) > 0 {
		if sfp, err := time.Parse("2006-01-02", sf); err == nil {
			start = sfp
		} 
	}
	f.Load(start, defaultDuration)
	return nil
}
