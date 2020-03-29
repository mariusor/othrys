package ical

import (
	"bytes"
	"fmt"
	"github.com/mariusor/esports-calendar/calendar"
	"github.com/mariusor/esports-calendar/storage"
	"github.com/mariusor/esports-calendar/storage/boltdb"
	"github.com/soh335/ical"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type cal struct {
	Version string
}

func NewHandler() *cal { return new(cal) }

func parsePath (u *url.URL) ([]string, int) {
	year := int64(time.Now().Year())
	if u == nil {
		return calendar.GetTypes(nil), int(year)
	}
	p := u.Path
	types := make([]string, 0)
	
	yearS, typesS := path.Split(p)
	year, _ = strconv.ParseInt(strings.Replace(yearS, "/", "", -1), 10, 32)
	
	maybeTypes := strings.Split(typesS, "+")
	types = calendar.GetTypes(maybeTypes)

	return types, int(year)
}

func (c *cal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	types, yearURL := parsePath(r.URL)
	dateURL := fmt.Sprintf("%d-01-01 00:00:00", yearURL)

	var date time.Time
	var err error
	date, _ = time.Parse("2006-01-02 15:04:05", dateURL)
	st := boltdb.New(boltdb.Config{
		Path:  "./calendar.bdb",
		LogFn: nil,
		ErrFn: nil,
	})
	// use one year
	duration := 8759*time.Hour + 59*time.Minute + 59*time.Second
	if !date.IsZero() {
		date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	}

	events, err := st.LoadEvents(storage.DateCursor{T: date, D: duration}, types...)

	cal := ical.NewBasicVCalendar()
	cal.PRODID = fmt.Sprintf("-//TL//ESPORTS-CAL//EN/%s", c.Version)

	cal.VERSION = "2.0"
	cal.URL = fmt.Sprintf("https://calendar.littr.me%s", r.URL.String())

	name := "EsportsCalendar"
	description := name

	cal.NAME = name
	cal.X_WR_CALNAME = name
	lbls := make([]string, 0)
	for _, typ := range types {
		if label, ok := calendar.Labels[typ]; ok {
			lbls = append(lbls, label)
		}
		if col, ok := calendar.Colors[typ]; ok {
			cal.COLOR = col
		}
	}
	if len(lbls) > 0 {
		description = fmt.Sprintf("EsportsCalendar, events for %s", strings.Join(lbls, ", "))
	}
	cal.DESCRIPTION = description
	cal.X_WR_CALDESC = description

	tz := date.Location().String()
	cal.TIMEZONE_ID = tz
	cal.X_WR_TIMEZONE = tz

	cal.REFRESH_INTERVAL = "PT1H"
	cal.X_PUBLISHED_TTL = "PT1H"

	cal.CALSCALE = "GREGORIAN"
	cal.METHOD = "PUBLISH"
	for _, ev := range events {
		summary := ev.Stage
		if ev.Category != "" {
			summary = fmt.Sprintf("[%s] %s: %s", ev.Type, ev.Category, summary)
		}

		e := &ical.VEvent{
			UID:         fmt.Sprintf("%d", ev.CalID),
			DTSTAMP:     ev.LastModified,
			DTSTART:     ev.StartTime,
			DTEND:       ev.StartTime.Add(ev.Duration),
			SUMMARY:     summary,
			DESCRIPTION: ev.Content,
			TZID:        tz,
			AllDay:      ev.Duration > 24*time.Hour,
		}
		cal.VComponent = append(cal.VComponent, e)
	}

	b := &bytes.Buffer{}
	err = cal.Encode(b)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%s", err)))
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Write(b.Bytes())
	w.WriteHeader(http.StatusOK)
}
