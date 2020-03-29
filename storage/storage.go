package storage

import (
	"github.com/mariusor/esports-calendar/calendar"
	"time"
)

type DateCursor struct {
	T time.Time
	D time.Duration
}

func Cursor(st time.Time, d time.Duration) DateCursor {
	return DateCursor{
		T: st,
		D: d,
	}
}

type Saver interface {
	SaveEvents(...calendar.Events) error
}

type Loader interface {
	LoadEvents(DateCursor, ...string) (calendar.Events, error)
	LoadEvent(string, time.Time, int64) calendar.Event
}
