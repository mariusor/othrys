package cmd

import (
	"fmt"
	"path"
	"time"

	"github.com/urfave/cli"

	"git.sr.ht/~mariusor/othrys/calendar"
	"git.sr.ht/~mariusor/othrys/internal/post"
	"git.sr.ht/~mariusor/othrys/storage"
	"git.sr.ht/~mariusor/othrys/storage/boltdb"
)

var PostCmd = cli.Command{
	Name:  "post",
	Usage: "Post events to the Fediverse",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "calendar",
			Usage: "Which calendars to post",
			Value: (*cli.StringSlice)(&calendar.DefaultCalendars),
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Output debug messages",
		},
		&cli.StringFlag{
			Name:  "date",
			Usage: "Date at which to start",
			Value: defaultStartTime.Format("2006-01-02"),
		},
		&cli.StringSliceFlag{
			Name:  "instance",
			Usage: "The instances to post to",
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: "The type of the instance: Mastodon, FedBOX, oni",
			Value: "oni",
		},
	},
	Action: Post(ResolutionDay),
}

type PostConfig struct {
	Path       string
	DryRun     bool
	Date       time.Time
	Resolution time.Duration
	PostFns    []post.PosterFn
	infFn      logFn
	errFn      logFn
}

func parseStartDate(s string) time.Time {
	d := time.Now().UTC()
	if s != "" {
		if parsed, err := time.Parse("2006-01-02", s); err == nil {
			d = parsed
		}
	}
	return d.Truncate(24 * time.Hour)
}

func stringValue(c *cli.Context, p string) string {
	for {
		if c.IsSet(p) {
			if val := c.String(p); val != "" {
				return val
			}
		}
		if c = c.Parent(); c == nil {
			break
		}
	}
	return ""
}

const (
	TypeMastodon = "mastodon"
	TypeONI      = "oni"
	TypeFedBOX   = "fedbox"
)

func stringSliceValues(c *cli.Context, p string) []string {
	for {
		if c.IsSet(p) {
			if values := c.StringSlice(p); values != nil {
				return values
			}
		}
		if c = c.Parent(); c == nil {
			break
		}
	}
	return nil
}

func Post(resolution time.Duration) cli.ActionFunc {
	return func(c *cli.Context) error {
		conf := PostConfig{
			DryRun:     c.GlobalBool("dry-run"),
			Date:       parseStartDate(stringValue(c, "date")),
			Resolution: resolution,
			Path:       c.GlobalString("path"),
		}

		calendars := stringSliceValues(c, "calendar")
		calendars = calendar.GetTypes(calendars)

		if !conf.DryRun {
			creds, err := post.LoadCredentials(DataPath())
			if err != nil {
				return fmt.Errorf("unable to load credentials for the client: %w", err)
			}
			for _, cred := range creds {
				if cred.Valid(c) {
					conf.PostFns = append(conf.PostFns, cred.Post())
				}
			}
		}
		if len(conf.PostFns) == 0 {
			conf.PostFns = append(conf.PostFns, post.ToStdout)
		}
		return LoadAndPost(conf, calendars...)
	}
}

func LoadAndPost(c PostConfig, types ...string) error {
	if c.Resolution == 0 {
		c.Resolution = ResolutionDay
	}

	f, err := New(true, types...)
	if err != nil {
		return err
	}

	if len(f.Types) == 0 {
		return fmt.Errorf("no valid calendars have been passed: %s", types)
	}

	repo := boltdb.New(boltdb.Config{
		Path:  path.Join(c.Path, boltdb.DefaultFile),
		LogFn: boltdb.LoggerFn(c.infFn),
		ErrFn: boltdb.LoggerFn(c.errFn),
	})

	releases, err := repo.LoadEvents(storage.Cursor(c.Date, c.Resolution), types...)
	if err != nil {
		return fmt.Errorf("unable to load releases from storage: %w", err)
	}
	releases = getEventsForTimeAndResolution(releases, c.Date, c.Resolution)

	if len(releases) == 0 {
		info("No releases for the period: %s %s", c.Date.Format("Monday, _2 January 2006"), FormatDuration(c.Resolution))
		return nil
	}

	toPostReleases := make(map[time.Time]calendar.Events)
	for _, r := range releases {
		day := r.StartTime.Truncate(ResolutionDay)
		toPostReleases[day] = append(toPostReleases[day], r)
	}

	for _, postFn := range c.PostFns {
		if err := postFn(toPostReleases); err != nil {
			info("Error trying to post: %s", err)
		}
	}
	return nil
}

func getEventsForTimeAndResolution(rel calendar.Events, when time.Time, resolution time.Duration) calendar.Events {
	periodRel := make([]calendar.Event, 0)

	for _, r := range rel {
		relativeStart := r.StartTime.Truncate(resolution).Sub(when)
		relativeEnd := r.StartTime.Add(r.Duration).Truncate(resolution).Sub(when)
		if relativeStart == 0 || relativeEnd == 0 {
			periodRel = append(periodRel, r)
		}
	}
	return periodRel
}

func FormatDuration(d time.Duration) string {
	label := "hour"
	val := float64(d) / float64(time.Hour)
	if d > ResolutionDay {
		label = "day"
		val = float64(d) / float64(ResolutionDay)
	}
	if d > ResolutionWeek {
		label = "week"
		val = float64(d) / float64(ResolutionWeek)
	}
	if d > ResolutionMonthish {
		label = "month"
		val = float64(d) / float64(ResolutionMonthish)
	}
	if d > ResolutionYearish {
		label = "year"
		val = float64(d) / float64(ResolutionYearish)
	}
	if val != 1.0 && val != -1.0 {
		label = label + "s"
	}
	return fmt.Sprintf("%+.2g%s", val, label)
}

const (
	ResolutionDay      = 24 * time.Hour
	ResolutionWeek     = 7 * ResolutionDay
	ResolutionMonthish = 31 * ResolutionDay
	ResolutionYearish  = 365 * ResolutionDay
)

func PostEverything(dataPath string, resolution time.Duration) error {
	creds, err := post.LoadCredentials(dataPath)
	if err != nil {
		return fmt.Errorf("no credentials found: %w", err)
	}
	conf := PostConfig{
		Date:       time.Now().UTC(),
		Resolution: resolution,
		Path:       dataPath,
	}
	for _, cred := range creds {
		conf.PostFns = append(conf.PostFns, cred.Post())
	}

	if err = LoadAndPost(conf, calendar.GetTypes(nil)...); err != nil {
		return fmt.Errorf("unable to post: %w", err)
	}

	return nil
}
