package ical

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/mariusor/esports-calendar/calendar/liquid"
	"github.com/mariusor/esports-calendar/calendar/plusforward"
	"github.com/mariusor/esports-calendar/storage"
	"github.com/mariusor/esports-calendar/storage/boltdb"
	"net/http"
	"strings"
	"time"
)

type ical struct{}

func (i ical) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	b, _ := json.Marshal(events)

	w.Write(b)
	w.WriteHeader(http.StatusOK)
}