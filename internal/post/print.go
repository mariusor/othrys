package post

import (
	"github.com/mariusor/esports-calendar/calendar"
	"log"
	"time"
)

const dateFmt = "2006-01-02 15:04"

func PostToStdout(groups map[time.Time]calendar.Events) error {
	f := log.Flags()
	log.SetFlags(0)
	for date, releases := range groups {
		log.Printf("%s\n", date.Format(dateFmt))
		for i, rel := range releases {
			log.Printf("#%d %s", i, rel)
		}
	}
	log.SetFlags(f)
	return nil
}
