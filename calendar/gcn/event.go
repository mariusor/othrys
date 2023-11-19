package gcn

import "time"

type event struct {
	CalID          int64
	StartTime      time.Time
	EndTime        time.Time
	LastModified   time.Time
	Name           string
	Discipline     string
	Classification string
	Content        string
	Links          []string
	Canceled       bool
	TagNames       []string
}

type events []event
