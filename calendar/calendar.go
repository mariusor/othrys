package calendar

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~mariusor/othrys/calendar/gcn"
	"git.sr.ht/~mariusor/othrys/calendar/liquid"
	"git.sr.ht/~mariusor/othrys/calendar/plusforward"
	vocab "github.com/go-ap/activitypub"
)

var DefaultCalendars = []string{liquid.LabelTeamLiquid, plusforward.LabelPlusForward}

type Fetcher interface {
	Load(startDate time.Time) (Events, error)
}

type Source interface {
	Load() error
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
	TagNames     []string
	Tags         vocab.ItemCollection `json:"-"`
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
	return fmt.Sprintf("<[%d] "+f+" @ %s//%s>", e.CalID, e.Type, cat, stg, fmtTime, e.Duration)
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

func inStringList(s string, list []string) bool {
	for _, lss := range list {
		if lss == s {
			return true
		}
	}
	return false
}

func GetTypes(strs []string) []string {
	types := make([]string, 0)
	if len(strs) == 0 {
		return append(append(liquid.ValidTypes[:], plusforward.ValidTypes[:]...), gcn.ValidTypes[:]...)
	}
	for _, typ := range strs {
		if ext := filepath.Ext(typ); ext != "" {
			typ = strings.Replace(typ, ext, "", 1)
		}
		if !liquid.ValidType(typ) && !plusforward.ValidType(typ) && !gcn.ValidType(typ) || inStringList(typ, types) {
			continue
		}
		if typ == liquid.LabelTeamLiquid {
			for _, t := range liquid.ValidTypes {
				if inStringList(t, types) {
					continue
				}
				types = append(types, t)
			}
		}
		if typ == plusforward.LabelPlusForward {
			for _, t := range plusforward.ValidTypes {
				if inStringList(t, types) {
					continue
				}
				types = append(types, t)
			}
		}
		types = append(types, typ)
	}
	return types
}

var Colors = map[string]string{
	liquid.LabelSC2:          "99:99:99",
	liquid.LabelSCRemastered: "99:99:99",
	liquid.LabelBW:           "99:99:99",
	liquid.LabelCSGO:         "99:99:99",
	liquid.LabelHOTS:         "99:99:99",
	liquid.LabelSmash:        "99:99:99",
	//liquid.LabelHearthstone:         "99:99:99",
	liquid.LabelDota:                "99:99:99",
	liquid.LabelLOL:                 "99:99:99",
	liquid.LabelOverwatch:           "99:99:99",
	plusforward.LabelQuakeLive:      "99:99:99",
	plusforward.LabelQuakeIV:        "99:99:99",
	plusforward.LabelQuakeIII:       "99:99:99",
	plusforward.LabelQuakeII:        "99:99:99",
	plusforward.LabelQuakeWorld:     "99:99:99",
	plusforward.LabelDiabotical:     "99:99:99",
	plusforward.LabelDoom:           "99:99:99",
	plusforward.LabelReflex:         "99:99:99",
	plusforward.LabelGG:             "99:99:99",
	plusforward.LabelUnreal:         "99:99:99",
	plusforward.LabelWarsow:         "99:99:99",
	plusforward.LabelDbmb:           "99:99:99",
	plusforward.LabelXonotic:        "99:99:99",
	plusforward.LabelQuakeChampions: "99:99:99",
	plusforward.LabelQuakeCPMA:      "99:99:99",
}

var Labels = map[string]string{
	liquid.LabelTeamLiquid:          "TeamLiquid",
	liquid.LabelSC2:                 "StarCraft 2",
	liquid.LabelSCRemastered:        "StarCraft Remastered",
	liquid.LabelBW:                  "BroodWar",
	liquid.LabelCSGO:                "Counter-Strike: Global Offensive",
	liquid.LabelHOTS:                "Heroes of the Storm",
	liquid.LabelSmash:               "Smash",
	liquid.LabelDota:                "DotA",
	liquid.LabelLOL:                 "League of Legends",
	liquid.LabelOverwatch:           "Overwatch",
	plusforward.LabelPlusForward:    "PlusForward",
	plusforward.LabelQuakeLive:      "Quake Live",
	plusforward.LabelQuakeIV:        "Quake IV",
	plusforward.LabelQuakeIII:       "Quake III",
	plusforward.LabelQuakeII:        "Quake II",
	plusforward.LabelQuakeWorld:     "Quake World",
	plusforward.LabelDiabotical:     "Diabotical",
	plusforward.LabelDoom:           "DOOM",
	plusforward.LabelReflex:         "Reflex",
	plusforward.LabelGG:             "GG",
	plusforward.LabelUnreal:         "Unreal",
	plusforward.LabelWarsow:         "Warsow",
	plusforward.LabelDbmb:           "DBMB",
	plusforward.LabelXonotic:        "Xonotic",
	plusforward.LabelQuakeChampions: "Quake Champions",
	plusforward.LabelQuakeCPMA:      "Quake CPMA",
}

func GetCalendarURL(typ string, date time.Time, byWeek bool) (*url.URL, error) {
	if plusforward.ValidType(typ) {
		return plusforward.GetCalendarURL(typ, date, byWeek)
	} else if liquid.ValidType(typ) {
		return liquid.GetCalendarURL(typ, date, byWeek)
	} else if gcn.ValidType(typ) {
		return gcn.GetCalendarURL(typ, date)
	}
	return nil, fmt.Errorf("invalid type %s", typ)
}

func LoadEvents(typ string, date time.Time) (Events, error) {
	events := make(Events, 0)
	u, err := GetCalendarURL(typ, date, false)
	if err != nil {
		return events, err
	}
	if plusforward.ValidType(typ) {
		e, err := plusforward.LoadEvents(u, date)
		if err != nil {
			return nil, err
		}
		for _, ev := range e {
			events = append(events, Event{
				CalID:        ev.CalID,
				StartTime:    ev.StartTime,
				Duration:     ev.Duration,
				LastModified: ev.LastModified,
				Type:         ev.Type,
				Category:     ev.Category,
				Stage:        ev.Stage,
				Content:      ev.Content,
				MatchCount:   ev.MatchCount,
				Links:        ev.Links,
				Canceled:     ev.Canceled,
				TagNames:     ev.TagNames,
			})
		}
	} else if liquid.ValidType(typ) {
		e, err := liquid.LoadEvents(u, date)
		if err != nil {
			return nil, err
		}
		for _, ev := range e {
			events = append(events, Event{
				CalID:        ev.CalID,
				StartTime:    ev.StartTime,
				Duration:     ev.Duration,
				LastModified: ev.LastModified,
				Type:         ev.Type,
				Category:     ev.Category,
				Stage:        ev.Stage,
				Content:      ev.Content,
				MatchCount:   ev.MatchCount,
				Links:        ev.Links,
				Canceled:     ev.Canceled,
				TagNames:     ev.TagNames,
			})
		}
	} else if gcn.ValidType(typ) {
		e, err := gcn.LoadEvents(u, date)
		if err != nil {
			return nil, err
		}
		for _, ev := range e {
			events = append(events, Event{
				CalID:        ev.CalID,
				StartTime:    ev.StartTime,
				Duration:     ev.EndTime.Sub(ev.StartTime),
				LastModified: ev.LastModified,
				Type:         gcn.Label(ev.Discipline),
				Category:     ev.Classification,
				Stage:        ev.Classification,
				Content:      ev.Content,
				MatchCount:   1,
				Links:        ev.Links,
				Canceled:     ev.Canceled,
				TagNames:     ev.TagNames,
			})
		}
	} else {
		err = fmt.Errorf("invalid type %s", typ)
	}
	return events, err
}
