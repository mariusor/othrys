package calendar

import (
	"fmt"
	"time"
)

type Event struct {
	CalId int
	StartTime time.Time
	Duration time.Duration
	LastModified time.Time
	Type string
	Category string
	Stage string
	Content string
	MatchCount int
	Links []string
	Canceled bool
}

func (e Event) IsValid() bool {
	return false
}

func (e Event) Equals(inc Event) bool {
	return false
}

func (e Event) String() string {
	if len(e.Category) == 0 || len(e.Stage)== 0 {
		return fmt.Sprintf("<%s//%s>", e.StartTime, e.Duration)
	}
	return  fmt.Sprintf("<[%s:%s] @ %s//%s>", e.Category, e.Stage, e.StartTime, e.Duration)
}
