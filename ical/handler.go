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

var colors = map[string]string{
	liquid.LabelSC2:                 "99:99:99",
	liquid.LabelSCRemastered:        "99:99:99",
	liquid.LabelBW:                  "99:99:99",
	liquid.LabelCSGO:                "99:99:99",
	liquid.LabelHOTS:                "99:99:99",
	liquid.LabelSmash:               "99:99:99",
	liquid.LabelHearthstone:         "99:99:99",
	liquid.LabelDota:                "99:99:99",
	liquid.LabelLOL:                 "99:99:99",
	liquid.LabelOverwatch:           "99:99:99",
	plusforward.LabelQuakeLive:      "99:99:99",
	plusforward.LabelQuakeIV:        "99:99:99",
	plusforward.LabelQuakeIII:       "99:99:99",
	plusforward.LabelQuakeII:        "99:99:99",
	plusforward.LabelQuakeWorld:     "99:99:99",
	plusforward.LabelDiabotical:     "99:99:99",
	plusforward.LabelDoom:           "99:99:99",
	plusforward.LabelReflex:         "99:99:99",
	plusforward.LabelGG:             "99:99:99",
	plusforward.LabelUnreal:         "99:99:99",
	plusforward.LabelWarsow:         "99:99:99",
	plusforward.LabelDbmb:           "99:99:99",
	plusforward.LabelXonotic:        "99:99:99",
	plusforward.LabelQuakeChampions: "99:99:99",
	plusforward.LabelQuakeCPMA:      "99:99:99",
}

var labels = map[string]string{
	liquid.LabelSC2:                 "StarCraft 2",
	liquid.LabelSCRemastered:        "StarCraft Remastered",
	liquid.LabelBW:                  "BroodWar",
	liquid.LabelCSGO:                "Counterstrike: Go",
	liquid.LabelHOTS:                "Heroes of the Storm",
	liquid.LabelSmash:               "Smash",
	liquid.LabelHearthstone:         "Hearthstone",
	liquid.LabelDota:                "DotA",
	liquid.LabelLOL:                 "League of Legends",
	liquid.LabelOverwatch:           "Overwatch",
	plusforward.LabelQuakeLive:      "Quake Live",
	plusforward.LabelQuakeIV:        "Quake IV",
	plusforward.LabelQuakeIII:       "Quake III",
	plusforward.LabelQuakeII:        "Quake II",
	plusforward.LabelQuakeWorld:     "Quake World",
	plusforward.LabelDiabotical:     "Diabotical",
	plusforward.LabelDoom:           "DOOM",
	plusforward.LabelReflex:         "Reflex",
	plusforward.LabelGG:             "GG",
	plusforward.LabelUnreal:         "Unreal",
	plusforward.LabelWarsow:         "Warsow",
	plusforward.LabelDbmb:           "DBMB",
	plusforward.LabelXonotic:        "Xonotic",
	plusforward.LabelQuakeChampions: "Quake Champions",
	plusforward.LabelQuakeCPMA:      "Quake CPMA",
}

func (c cal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	// /{type}/{year}/{month}/{day}
	typ := strings.ToLower(chi.URLParam(r, "type"))

	yearURL := strings.ToLower(chi.URLParam(r, "year"))
	if len(yearURL) == 0 {
		yearURL = fmt.Sprintf("%4d", now.Year())
	}
	dateURL := fmt.Sprintf("%s-01-01 00:00:00", yearURL)

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
	date, _ = time.Parse("2006-01-02 15:04:05", dateURL)
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
	cal.PRODID = fmt.Sprintf("-//TL//ESPORTS-CAL//EN/%s", c.Version)

	cal.VERSION = "2.0"
	cal.URL = fmt.Sprintf("https://calendar.littr.me/%s/%d", typ, date.Year())

	name := "EsportsCalendar"
	description := name

	cal.NAME = name
	cal.X_WR_CALNAME = name
	if label, ok := labels[typ]; ok {
		description = fmt.Sprintf("EsportsCalendar, events for %s", label)
	}
	cal.DESCRIPTION = description
	cal.X_WR_CALDESC = description

	tz := date.Location().String()
	cal.TIMEZONE_ID = tz
	cal.X_WR_TIMEZONE = tz

	cal.REFRESH_INTERVAL = "PT1H"
	cal.X_PUBLISHED_TTL = "PT1H"

	if col, ok := colors[typ]; ok {
		cal.COLOR = col
	}
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

	w.Write(b.Bytes())
	w.WriteHeader(http.StatusOK)
}
