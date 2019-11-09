package cmd

import (
	"fmt"
	"github.com/mariusor/esports-calendar/storage"
	"github.com/mariusor/esports-calendar/storage/boltdb"
	"github.com/urfave/cli"
	"time"
)

var List = cli.Command{
	Name:  "list",
	Usage: "Lists already saved calendar events",
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
			Value: now.Format("2006-01-02"),
		},
		&cli.DurationFlag{
			Name:  "end",
			Usage: "Date interval to check",
			Value: defaultDuration,
		},
	},
	Action: listCalendars,
}

func listCalendars(c *cli.Context) error {
	types := c.StringSlice("calendar")

	start := time.Now().Add(-1 * defaultDuration)
	if sf := c.String("start"); len(sf) > 0 {
		if sfp, err := time.Parse("2006-01-02", sf); err == nil {
			start = sfp
		}
	}
	duration := c.Duration("end")

	f, err := New(true, types...)
	if err != nil {
		return err
	}

	date := start
	st := boltdb.New(boltdb.Config{
		Path:  "./calendar.bdb",
		LogFn: nil,
		ErrFn: nil,
	})

	f.log("Loading events for period: %s - %s", date.Format("2006-01-02 Mon, 15:04"), date.Add(duration).Format("2006-01-02 Mon, 15:04"))
	events, err := st.LoadEvents(storage.DateCursor{T: start, D: duration}, types...)
	if err != nil {
		return fmt.Errorf("unable to load events: %w", err)
	}
	if len(events) == 0 {
		fmt.Printf("nothing found")
		return nil
	}
	for _, e := range events {
		fmtTime := e.StartTime.Format("2006-01-02 15:04 MST")
		cat := ""
		stg := ""
		fm := "%s%s%s"
		if len(e.Category) > 0 {
			cat = e.Category
			fm = "%s:%s%s"

		}
		if len(e.Stage) > 0 {
			stg = e.Stage
			fm = "%s:%s:%s"
		}
		f.log("[%d] "+fm+" @ %s//%s", e.CalID, e.Type, cat, stg, fmtTime, e.Duration)
		if e.Content != "" {
			f.log("%v", e.Content)
		}
	}
	return err
}
