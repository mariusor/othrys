package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/urfave/cli"

	"github.com/mariusor/esports-calendar/calendar"
	"github.com/mariusor/esports-calendar/storage/boltdb"
)

var now = time.Now()

var (
	defaultDuration  = time.Hour * 336 // 2 weeks
	defaultStartTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
)

const (
	AppName    = "othrys"
	AppVersion = "(unknown)"
)

var (
	AppWebsite = "https://github.com/mariusor/ecal-server"
	AppScopes  = []string{"read+write+follow"}
)

func MkDirIfNotExists(p string) error {
	fi, err := os.Stat(p)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(p, os.ModeDir|os.ModePerm|0700)
	}
	if err != nil {
		return err
	}
	fi, err = os.Stat(p)
	if err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("path exists, and is not a folder %s", p)
	}
	return nil
}

func CachePath() string {
	xdgCachePath, _ := os.UserCacheDir()
	return filepath.Join(xdgCachePath, AppName)
}

func DataPath() string {
	homeDir, _ := os.UserHomeDir()
	xdgDataPath := filepath.Join(homeDir, ".local", "share")
	appPath := filepath.Join(xdgDataPath, AppName)

	if _, err := os.Stat(appPath); err != nil && errors.Is(err, os.ErrNotExist) {
		err := MkDirIfNotExists(appPath)
		if err != nil {
			log.Fatalf("Error: %s", err.Error())
		}
	}
	return appPath
}

var FetchCmd = cli.Command{
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
			Value: defaultStartTime.Format("2006-01-02"),
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
	debug  bool
	Types  []string
	weekly bool
	err    logFn
	log    logFn
}

func New(debug bool, types ...string) (*cal, error) {
	logFn := func(s string, args ...interface{}) {
		fmt.Printf(s, args...)
		fmt.Println()
	}
	errFn := func(s string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, s, args...)
		fmt.Fprintln(os.Stderr)
	}
	return &cal{
		debug:  debug,
		Types:  calendar.GetTypes(types),
		weekly: true,
		log:    logFn,
		err:    errFn,
	}, nil
}

type logFn func(string, ...interface{})

type toLoad struct {
	d time.Time
	t string
}

func (c cal) Load(startDate time.Time) (calendar.Events, error) {
	events := make(calendar.Events, 0)
	urls := make([]toLoad, 0)
	date := startDate
	for _, typ := range c.Types {
		urls = append(urls, toLoad{d: date, t: typ})
	}

	for _, l := range urls {
		if c.debug {
			u, _ := calendar.GetCalendarURL(l.t, l.d, c.weekly)
			c.log("Loading [%s]: %s", l.t, u)
		}
		ev, err := calendar.LoadEvents(l.t, l.d)
		if err != nil {
			c.err("Unable to parse page URI for type %s: %s", l.t, err)
			continue
		}
		events = append(events, ev...)
		if c.debug {
			c.log("%d events", len(ev))
		}
	}

	return events, nil
}

const durationStep = 7 * 24 * time.Hour

func fetchCalendars(c *cli.Context) error {
	types := c.StringSlice("calendar")

	start := time.Now().Add(-1 * defaultDuration)
	if sf := c.String("start"); len(sf) > 0 {
		if sfp, err := time.Parse("2006-01-02", sf); err == nil {
			start = sfp
		}
	}
	duration := c.Duration("end")
	debug := c.Bool("debug") || c.GlobalBool("debug")

	f, err := New(debug, types...)
	if err != nil {
		return err
	}

	if len(f.Types) == 0 {
		return fmt.Errorf("no valid calendars have been passed: %s", types)
	}
	date := start
	endDate := start.Add(duration - time.Second)
	st := boltdb.New(boltdb.Config{
		Path:  path.Join(c.GlobalString("path"), boltdb.DefaultFile),
		LogFn: nil,
		ErrFn: nil,
	})

	var events calendar.Events
	for {
		duration := durationStep - time.Second
		if debug {
			f.log("Loading events for period: %s - %s", date.Format("2006-01-02 Mon, 15:04"), date.Add(duration).Format("2006-01-02 Mon, 15:04"))
		}
		events, err = f.Load(date)
		for _, e := range events {
			if debug {
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
			old := st.LoadEvent(e.Type, e.StartTime, e.CalID)
			if old.IsValid() {
				fmt.Printf("%v", old)
			}
			if !old.Equals(e) {
				err := st.SaveEvent(e)
				if err != nil {
					f.err("Error saving %d: %s", e.CalID, err)
				}
			}
		}
		if endDate.Sub(date) <= durationStep {
			break
		}
		date = date.Add(duration)
	}
	return err
}
