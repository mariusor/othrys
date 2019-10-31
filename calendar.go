package calendar

import (
	"fmt"
	"github.com/anaskhan96/soup"
	"github.com/mariusor/esports-calendar/app/liquid"
	"github.com/mariusor/esports-calendar/app/plusforward"
	"net/url"
	"os"
	"time"
)

type Fetcher interface {
	Load(startDate time.Time, period time.Duration) ([]Event, error)
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

type cal struct {
	types  []string
	weekly bool
	err    logFn
	log    logFn
}

func New(types ...string) *cal {
	logFn := func(s string, args ...interface{}) {
		fmt.Printf(s, args...)
	}
	errFn := func(s string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, s, args...)
	}
	return &cal{
		types:  types,
		weekly: true,
		log:    logFn,
		err:    errFn,
	}
}

func (c cal) Load(startDate time.Time, period time.Duration)  ([]Event, error) {
	events := make([]Event, 0)
	urls := make([]*url.URL, 0)
	for _, typ := range c.types {
		validType := false
		if plusforward.ValidType(typ) {
			url, err := plusforward.GetCalendarURL(startDate, typ, true)
			if err != nil {
				//c.err("unable to get calendar URI for %s: %s", typ, err)
				continue
			}
			validType = true
			urls = append(urls, url)
		}
		if liquid.ValidType(typ) {
			url, err := liquid.GetCalendarURL(startDate, typ, true)
			if err != nil {
				//c.err("unable to get calendar URI for %s: %s", typ, err)
				continue
			}
			validType = true
			urls = append(urls, url)
		}
		if !validType {
			c.err("invalid type %s", typ)
		}
	}
	for _, url := range urls {
		r, err := loadURL(url)
		if err != nil {
			c.err("Unable to load URI %s: %s", url, err)
			continue
		}
		c.log("%v\n", r)
	}
	return events, nil
}

func loadURL(u *url.URL) (*soup.Root, error) {
	resp, err := soup.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to load calendar page: %w", err)
	}
	r := soup.HTMLParse(resp)
	return &r, nil
}
