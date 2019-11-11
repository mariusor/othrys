package ical

import (
	"bytes"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/mariusor/esports-calendar/calendar/liquid"
	"github.com/mariusor/esports-calendar/calendar/plusforward"
	"github.com/mariusor/esports-calendar/storage"
	"github.com/mariusor/esports-calendar/storage/boltdb"
	"github.com/soh335/ical"
	"net/http"
	"strings"
	"time"
)

type cal struct {
	Version string
}

func (c cal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	// /{type}/{year}/{month}/{day}
	typ := strings.ToLower(chi.URLParam(r, "type"))

	yearURL := strings.ToLower(chi.URLParam(r, "year"))
	if len(yearURL) == 0 {
		yearURL = fmt.Sprintf("%4d", now.Year())
	}
	dateURL := fmt.Sprintf("%s-01-01", yearURL)

	types := make([]string, 0)
	if typ != "" {
		if !liquid.ValidType(typ) || plusforward.ValidType(typ) {
			// error wrong type
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("Invalid type %s", typ)))
			return
		}
		types = append(types, typ)
	}
	var date time.Time
	var err error
	if date, err = time.Parse("2006-01-02", dateURL); err != nil {
		if date, err = time.Parse("2006-january-02", dateURL); err != nil {
			date, _ = time.Parse("2006-jan-02", dateURL)
		}
	}
	st := boltdb.New(boltdb.Config{
		Path:  "./calendar.bdb",
		LogFn: nil,
		ErrFn: nil,
	})
	// use one year
	var duration time.Duration = 8759*time.Hour + 59*time.Minute + 59*time.Second
	if !date.IsZero() {
		date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	}

	events, err := st.LoadEvents(storage.DateCursor{T: date, D: duration}, types...)

	cal := ical.NewBasicVCalendar()
	cal.PRODID = "TL-CAL/v2.0"

	cal.VERSION = c.Version
	cal.URL = "https://calendar.littr.me/"

	cal.NAME = "EsportsCalendar"
	cal.X_WR_CALNAME = "EsportsCalendar"
	cal.DESCRIPTION = "EsportsCalendar"
	cal.X_WR_CALDESC = "EsportsCalendar"

	cal.TIMEZONE_ID = "UTC"
	cal.X_WR_TIMEZONE = "UTC"

	cal.REFRESH_INTERVAL = "P1H"
	cal.X_PUBLISHED_TTL = "P1H"

	cal.COLOR = ""
	cal.CALSCALE = "GREGORIAN"
	cal.METHOD = "PUBLISH"
	for _, ev := range events {
		summary := ev.Stage
		if ev.Category != "" {
			summary = fmt.Sprintf("%s: %s", ev.Category, summary)
		}

		e := &ical.VEvent{
			UID:         fmt.Sprintf("%d", ev.CalID),
			DTSTAMP:     ev.LastModified,
			DTSTART:     ev.StartTime,
			DTEND:       ev.StartTime.Add(ev.Duration),
			SUMMARY:     summary,
			DESCRIPTION: ev.Content,
			TZID:        "UTC",
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

	w.Write(b.Bytes())
	w.WriteHeader(http.StatusOK)
}
