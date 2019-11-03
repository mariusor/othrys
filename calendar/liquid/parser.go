package liquid

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/mariusor/esports-calendar/calendar"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

func LoadEvents(url *url.URL, date time.Time) (calendar.Events, error) {
	if url == nil {
		return nil, fmt.Errorf("nil URL received")
	}
	res, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	events := make(calendar.Events, 0)
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	// Find the review items
	doc.Find("div.ev-feed").Each(func(i int, s *goquery.Selection) {
		var day time.Time
		startDay := int64(date.Day())
		if dataDay, exists := s.Attr("data-day"); exists {
			startDay, _ = strconv.ParseInt(dataDay, 10, 32)
			day = time.Date(date.Year(), date.Month(), int(startDay), date.Hour(), date.Minute(), date.Second(), 0, date.Location())
		}

		s.Find("div.ev-block").Each(func(i int, s *goquery.Selection) {
			ev := calendar.Event{}
			loadEvent(&ev, day, s)
			if ev.IsValid() && !events.Contains(ev) {
				events = append(events, ev)
			}
		})
	})

	return events, nil
}

func loadEvent(e *calendar.Event, date time.Time, s *goquery.Selection) {
	e.MatchCount = 1
	e.Category = LabelUnknown
	s.Find("div.ev-match").Each(func(i int, s *goquery.Selection) {
		e.Content = s.Text()
		r := regexp.MustCompile(" vs ")
		m := r.Find([]byte(e.Content))

		if len(m) > 0 {
			e.MatchCount = len(m)
		}
		timeVal := s.Find("span.ev-timer").Text()
		evTime, err := time.Parse("15:04", timeVal)
		if err != nil {
			evTime = date
		}
		e.StartTime = time.Date(date.Year(), date.Month(), date.Day(), evTime.Hour(), evTime.Minute(), 0, 0, date.Location())
	})
	e.Duration = 45 * time.Minute * time.Duration(e.MatchCount)
	s.Find("div.ev-stage").Each(func(i int, s *goquery.Selection) {
		e.Stage = s.Text()
	})
	s.Find("div.ev-ctrl").Each(func(i int, s *goquery.Selection) {
		ss := s.Find("span")
		e.Category = ss.Text()
		if attrID, ok := ss.Attr("data-event-id"); ok {
			if id, err := strconv.ParseInt(attrID, 10, 32); err == nil {
				e.CalID = id
			}
		}
	})
	if style, exists := s.Find("span.league-sprite-small").Attr("style"); exists {
		r := regexp.MustCompile(`\d+`)
		if m := r.FindSubmatch([]byte(style)); m != nil {
			if typID, err := strconv.ParseInt(string(m[0]), 10, 32); err == nil {
				e.Type = getType(typID)
			}
		}
	}
}

func getType(key int64) string {
	for lbl, id := range calendarType {
		if int64(id) == key {
			return lbl
		}
	}
	return LabelUnknown
}
