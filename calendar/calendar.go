package calendar

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Fetcher interface {
	Load(startDate time.Time) (Events, error)
}

type Event struct {
	CalID        int64
	StartTime    time.Time
	Duration     time.Duration
	LastModified time.Time
	Type         string
	Category     string
	Stage        string
	Content      string
	MatchCount   int
	Links        []string
	Canceled     bool
}

type Events []Event

func stringArrayEqual(a1, a2 []string) bool {
	if len(a1) != len(a2) {
		return false
	}
	sort.Strings(a1)
	sort.Strings(a2)
	for k, v := range a1 {
		if v != a2[k] {
			return false
		}
	}
	return true
}

func (e Event) IsValid() bool {
	return !e.StartTime.IsZero() && e.CalID > 0
}

func (e Event) Equals(other Event) bool {
	return e.CalID == other.CalID &&
		e.StartTime == other.StartTime &&
		e.Duration == other.Duration &&
		e.Type == other.Type &&
		e.Category == other.Category &&
		e.Stage == other.Stage &&
		e.Content == other.Content &&
		stringArrayEqual(e.Links, other.Links) &&
		e.Canceled == other.Canceled
}

func (e Event) String() string {
	return e.GoString()
}
func (e Event) GoString() string {
	fmtTime := e.StartTime.Format("2006-01-02 15:04 MST")
	cat := ""
	stg := ""
	f := "%s%s%s"
	if len(e.Category) > 0 {
		cat = e.Category
		f = "%s:%s%s"

	}
	if len(e.Stage) > 0 {
		stg = e.Stage
		f = "%s:%s:%s"
	}
	return fmt.Sprintf("<[%d] " + f + " @ %s//%s>", e.CalID, e.Type, cat, stg, fmtTime, e.Duration)
}
func (e Events) String() string {
	return e.GoString()
}

func (e Events) GoString() string {
	ss := make([]string, len(e))
	for i, ev := range e {
		ss[i] = ev.GoString()
	}
	return fmt.Sprintf("Events[%d]:\n\t%s\n", len(e), strings.Join(ss, "\n\t"))
}

func (e Events) Contains(inc Event) bool {
	for _, ev := range e {
		if ev.Equals(inc) {
			return true
		}
	}
	return false
}
