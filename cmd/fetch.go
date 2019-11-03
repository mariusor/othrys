package cmd

import (
	"fmt"
	"github.com/mariusor/esports-calendar/calendar"
	"github.com/mariusor/esports-calendar/calendar/liquid"
	"github.com/mariusor/esports-calendar/calendar/plusforward"
	"github.com/urfave/cli"
	"net/url"
	"os"
	"time"
)

var now = time.Now()

var (
	defaultCalendars = []string{liquid.LabelTeamLiquid, plusforward.LabelPlusForward} // all
	defaultDuration  = time.Hour * 336                                                // 2 weeks
	defaultStartTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
)
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

type cal struct {
	types  []string
	weekly bool
	err    logFn
	log    logFn
}

func New(types ...string) *cal {
	logFn := func(s string, args ...interface{}) {
		fmt.Printf(s, args...)
		fmt.Println()
	}
	errFn := func(s string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, s , args...)
		fmt.Fprintln(os.Stderr)
	}
	return &cal{
		types:  types,
		weekly: true,
		log:    logFn,
		err:    errFn,
	}
}

func GetTypes(types ...string) []string {
	fetchTypes := make([]string, 0)
	for _, cal := range types {
		if cal == liquid.LabelTeamLiquid {
			//fetchTypes = append(fetchTypes, liquid.ValidTypes[:]...)
			fetchTypes = append(fetchTypes, cal)
		} else if cal == plusforward.LabelPlusForward {
			//fetchTypes = append(fetchTypes, plusforward.ValidTypes[:]...)
			fetchTypes = append(fetchTypes, cal)
		} else {
			fetchTypes = append(fetchTypes, cal)
		}
	}
	return fetchTypes
}

func ValidTypes() []string {
	types := make([]string, 0)
	for _, typ := range liquid.ValidTypes {
		types = append(types, typ)
	}
	for _, typ := range plusforward.ValidTypes {
		types = append(types, typ)
	}
	return types
}

type logFn func(s string, args ...interface{})

type toLoad struct {
	u *url.URL
	d time.Time
	t string
}

func (c cal) Load(startDate time.Time, period time.Duration) (calendar.Events, error) {
	events := make(calendar.Events, 0)
	urls := make([]toLoad, 0)
	date := startDate
	endDate := startDate.Add(period)
	for {
		if endDate.Sub(date) <= 0 {
			break
		}
		for _, typ := range c.types {
			validType := false
			if plusforward.ValidType(typ) {
				url, err := plusforward.GetCalendarURL(date, typ, true)
				if err != nil {
					//c.err("unable to get calendar URI for %s: %s", typ, err)
					continue
				}
				validType = true
				urls = append(urls, toLoad{u: url, d: date, t: typ,})
			}
			if liquid.ValidType(typ) {
				url, err := liquid.GetCalendarURL(date, typ, true)
				if err != nil {
					//c.err("unable to get calendar URI for %s: %s", typ, err)
					continue
				}
				validType = true
				urls = append(urls, toLoad{u: url, d: date, t: typ,})
			}
			if !validType {
				c.err("invalid type %s", typ)
			}
		}
		date = date.Add(7 * 24 * time.Hour)
	}
	for _, l := range urls {
		if plusforward.ValidType(l.t) {
			ev, err := plusforward.LoadEvents(l.u, l.d)
			if err != nil {
				c.err("Unable to parse page URI %s: %s", l.u, err)
				continue
			}
			events = append(events, ev...)
		}
		if liquid.ValidType(l.t) {
			ev, err := liquid.LoadEvents(l.u, l.d)
			if err != nil {
				c.err("Unable to parse page URI %s: %s", l.u, err)
				continue
			}
			events = append(events, ev...)
		}
	}
	return events, nil
}

func fetchCalendars(c *cli.Context) error {
	types := GetTypes(c.StringSlice("calendar")...)
	f := New(types...)
	start := time.Now().Add(-1 * defaultDuration)
	if sf := c.String("start"); len(sf) > 0 {
		if sfp, err := time.Parse("2006-01-02", sf); err == nil {
			start = sfp
		}
	}
	duration := c.Duration("end")
	events, err := f.Load(start, duration)
	fmt.Printf("%#v", events)
	return err
}
