package post

import (
	"log"
	"time"
)

const dateFmt = "2006-01-02 15:04"

func PostToStdout(groupped map[time.Time][]release) error {
	f := log.Flags()
	log.SetFlags(0)
	for date, releases := range groupped {
		Info("%s\n", date.Format(dateFmt))
		for i, rel := range releases {
			Info("#%d %s", i, rel)
		}
	}
	log.SetFlags(f)
	return nil
}
