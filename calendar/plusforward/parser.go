package plusforward

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/mariusor/esports-calendar/calendar"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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
	doc.Find("td.cal_day").Each(func(i int, s *goquery.Selection) {
		len := len(s.Nodes)
		fmt.Sprintf("%d\n", len)
		var day time.Time
		dateVal := s.Find("div.cal_date").Text()
		if wdmv, err := time.Parse("MondayJanuary _2", dateVal); err == nil {
			day = time.Date(date.Year(), date.Month(), wdmv.Day(), date.Hour(), date.Minute(), date.Second(), 0, date.Location())
		} else if wdv, err := time.Parse("Monday _2", dateVal); err == nil {
			day = time.Date(date.Year(), date.Month(), wdv.Day(), date.Hour(), date.Minute(), date.Second(), 0, date.Location())
		} else {
			day = date
		}
		// regular events
		s.Find("div.cal_event").Each(func(i int, s *goquery.Selection) {
			ev := calendar.Event{}
			loadEvent(&ev, day, s)
			if ev.IsValid() && !events.Contains(ev) {
				events = append(events, ev)
			}
		})
		// full day events
		s.Find("a.cal_event").Each(func(i int, s *goquery.Selection) {
			ev := calendar.Event{}
			loadOngoingEvent(&ev, s)
			if ev.IsValid() && !events.Contains(ev) {
				events = append(events, ev)
			}
		})
	})

	return events, nil
}

func loadOngoingEvent(e *calendar.Event, s *goquery.Selection) {
	e.MatchCount = 1
	e.Type = LabelUnknown
	//category_div = event_block.find("div", class_="cal_e_title")
	var perTypID int64 = 0
	if class, ok := s.Attr("class"); ok {
		e.Type = getTypeFromClass(class)
		perTypID = calID / 200 * int64(calendarType[e.Type])
	}
	if href, ok := s.Attr("href"); ok {
		e.CalID = getCalIDFromHref(href) + calID + perTypID
	}
	if tit, ok := s.Attr("title"); ok {
		elems := strings.Split(tit, "|")
		if len(elems) > 0 {
			dates := strings.Split(elems[1], " -> ")
			if len(dates) > 0 {
				start, _ := time.Parse(" 02 Jan 2006 15:04 MST", dates[0])
				end, _ := time.Parse("02 Jan 2006 15:04 MST", dates[1])
				if !start.IsZero() {
					e.StartTime = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
					if !end.IsZero() {
						e.Duration = end.Sub(start)
					}
				}
			}
		}
	}
	e.Category = s.Find("div.cal_title").Text()
}

func loadEvent(e *calendar.Event, date time.Time, s *goquery.Selection) {
	e.MatchCount = 1
	e.Type = LabelUnknown

	var perTypID int64 = 0
	//category_div = event_block.find("div", class_="cal_e_title")
	s.Find("div.cal_e_title").Each(func(i int, s *goquery.Selection) {
		//title_div = category_div.find("div", class_="cal_title")
		if cat, ok := s.Find("div.cal_title").Find("a").Attr("title"); ok {
			e.Category = cat
		}
		//subtitle_div = event_block.find("div", class_="cal_e_subtitle")
		e.Stage = s.Find("div.cal_e_subtitle").Text()
		if evTime, err := time.Parse("15:04", s.Find("div.cal_time").Text()); err == nil {
			e.StartTime = time.Date(date.Year(), date.Month(), date.Day(), evTime.Hour(), evTime.Minute(), 0, 0, date.Location())
			e.Duration = 45 * time.Minute
		}
		if class, exists := s.Find("div.cal_cat").Find("i.pfcat").Attr("class"); exists {
			e.Type = getTypeFromClass(class)
			perTypID = calID / 200 * int64(calendarType[e.Type])
		}
		//matches_container_div = event_block.find("div", class_="cal_matches")
		s.Find("div.cal_matches").Each(func(i int, s *goquery.Selection) {
		})
		if href, ok := s.Find("a").Attr("href"); ok {
			e.CalID = getCalIDFromHref(href) + calID + perTypID
		}
	})
}
const calID = 200000000
func getCalIDFromHref(href string) int64 {
	r := regexp.MustCompile(`post/(\d+)`)
	m := r.FindSubmatch([]byte(href))
	if len(m) > 1 {
		if id, err := strconv.ParseInt(string(m[1]), 10, 32); err == nil {
			return id
		}
	}
	return -1
}

func getTypeFromClass(class string) string {
	r := regexp.MustCompile(`cat-?(\d+)`)
	if m := r.FindSubmatch([]byte(class)); len(m) > 1 {
		if typID, err := strconv.ParseInt(string(m[1]), 10, 32); err == nil {
			return getType(typID)
		}
	}
	return LabelUnknown
}

func getType(key int64) string {
	for lbl, id := range calendarType {
		if int64(id) == key {
			return lbl
		}
	}
	return LabelUnknown
}
