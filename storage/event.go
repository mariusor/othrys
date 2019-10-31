package storage

import (
	"fmt"
	"sort"
	"time"
)

type Event struct {
	CalID        int
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
	return false
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
	if len(e.Category) == 0 || len(e.Stage) == 0 {
		return fmt.Sprintf("<%s//%s>", e.StartTime, e.Duration)
	}
	return fmt.Sprintf("<[%s:%s] @ %s//%s>", e.Category, e.Stage, e.StartTime, e.Duration)
}
