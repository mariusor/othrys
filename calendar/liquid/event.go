package liquid

import (
	"sort"
	"time"
)

type event struct {
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

type events []event

func (e event) isValid() bool {
	return !e.StartTime.IsZero() && e.CalID > 0
}

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

func (e event) equals(other event) bool {
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

func (e events) contains(inc event) bool {
	for _, ev := range e {
		if ev.equals(inc) {
			return true
		}
	}
	return false
}
