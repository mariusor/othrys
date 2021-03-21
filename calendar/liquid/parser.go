package liquid

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const defaultMatchDuration = 45 * time.Minute

func LoadEvents(url *url.URL, date time.Time) (events, error) {
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

	events := make(events, 0)
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
			ev := event{}
			loadEvent(&ev, day, s)
			if ev.isValid() && !events.contains(ev) {
				events = append(events, ev)
			}
		})
	})

	return events, nil
}

const calID = 100000000

func loadEvent(e *event, date time.Time, s *goquery.Selection) {
	e.MatchCount = 1
	e.Category = LabelUnknown
	s.Find("div.ev-match").Each(func(i int, s *goquery.Selection) {
		rawContent := s.Text()
		lines := strings.Split(rawContent, "\n")
		newLines := make([]string, 0)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 0 {
				newLines = append(newLines, line)
			}
		}
		e.Content = strings.Join(newLines, "\n")

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
	e.Duration = time.Duration(e.MatchCount) * defaultMatchDuration
	s.Find("div.ev-stage").Each(func(i int, s *goquery.Selection) {
		e.Stage = s.Text()
	})
	var perTypID int64 = 0
	if style, exists := s.Find("span.league-sprite-small").Attr("style"); exists {
		r := regexp.MustCompile(`\d+`)
		if m := r.FindSubmatch([]byte(style)); m != nil {
			if typID, err := strconv.ParseInt(string(m[0]), 10, 32); err == nil {
				e.Type = getType(typID)
				perTypID = calID / 100 * typID
			}
		}
	}
	s.Find("div.ev-ctrl").Each(func(i int, s *goquery.Selection) {
		ss := s.Find("span")
		e.Category = ss.Text()
		if attrID, ok := ss.Attr("data-event-id"); ok {
			if id, err := strconv.ParseInt(attrID, 10, 32); err == nil {
				e.CalID = id + calID + perTypID
			}
		}
	})
}

func getType(key int64) string {
	for lbl, id := range calendarType {
		if int64(id) == key {
			return lbl
		}
	}
	return LabelUnknown
}
