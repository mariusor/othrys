package liquid

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const LabelSC2 = "sc2"
const LabelSCRemastered = "scrm"
const LabelBW = "bw"
const LabelCSGO = "csgo"
const LabelHOTS = "hots"
const LabelSmash = "smash"

const LabelDota = "dota"
const LabelLOL = "lol"
const LabelOverwatch = "ovw"
const LabelTeamLiquid = "tl"
const LabelUnknown = "unk"

var ValidTypes = [...]string{
	LabelSC2,
	LabelSCRemastered,
	LabelBW,
	LabelCSGO,
	LabelHOTS,
	LabelSmash,
	LabelDota,
	LabelLOL,
	LabelOverwatch,
}

var baseURIs = map[string]string{
	LabelTeamLiquid:   "https://tl.net",
	LabelSCRemastered: "https://tl.net",
	LabelSC2:          "https://tl.net",
	LabelBW:           "https://tl.net",
	LabelCSGO:         "https://tl.net",
	LabelHOTS:         "https://tl.net",
	LabelSmash:        "https://tl.net",
	LabelDota:         "https://tl.net",
	LabelLOL:          "https://tl.net",
	LabelOverwatch:    "https://tl.net",
}

var calendarPath = map[string]string{
	LabelTeamLiquid:   "/calendar/",
	LabelSCRemastered: "/calendar/",
	LabelSC2:          "/calendar/",
	LabelBW:           "/calendar/",
	LabelCSGO:         "/calendar/",
	LabelHOTS:         "/calendar/",
	LabelSmash:        "/calendar/",
	LabelDota:         "/calendar/",
	LabelLOL:          "/calendar/",
	LabelOverwatch:    "/calendar/",
}

var eventPath = map[string]string{
	LabelTeamLiquid:   "/calendar/manage",
	LabelSCRemastered: "/calendar/manage",
	LabelSC2:          "/calendar/manage",
	LabelBW:           "/calendar/manage",
	LabelCSGO:         "/calendar/manage",
	LabelHOTS:         "/calendar/manage",
	LabelSmash:        "/calendar/manage",
	LabelDota:         "/calendar/manage",
	LabelLOL:          "/calendar/manage",
	LabelOverwatch:    "/calendar/manage",
}
var calendarType = map[string]int{
	LabelUnknown:   0,
	LabelSC2:       1,
	LabelBW:        2,
	LabelCSGO:      3,
	LabelHOTS:      4,
	LabelSmash:     5,
	LabelOverwatch: 6,
	LabelDota:      8,
	LabelLOL:       9,
}

func ValidType(typ string) bool {
	if typ == LabelTeamLiquid {
		return true
	}
	for _, t := range ValidTypes {
		if strings.ToLower(typ) == t {
			return true
		}
	}
	return false
}

type lf struct {
	types []string
}

func (l lf) Load(startDate time.Time, period time.Duration) error {
	return nil
}

func getQuery(typ string, date time.Time, by string) url.Values {
	q := make(url.Values)
	q.Add("view", by)
	q.Add("year", date.Format("2006"))
	q.Add("month", date.Format("01"))
	q.Add("day", date.Format("02"))
	if typ != "0" {
		q.Add("game", typ)
	}
	return q
}

func GetEventURL(typ string, date time.Time, byWeek bool) (*url.URL, error) {
	if !ValidType(typ) {
		return nil, fmt.Errorf("invalid type: %s", typ)
	}
	base, ok := baseURIs[typ]
	if !ok {
		return nil, fmt.Errorf("unknown base URI for type: %s", typ)
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("unable to parse base URI: %w", err)
	}
	path, ok := eventPath[typ]
	if !ok {
		return nil, fmt.Errorf("unknown calendar path for type: %s", typ)
	}
	u.Path = path
	game, ok := calendarType[typ]
	if !ok {
		return nil, fmt.Errorf("unknown game id path for type: %s", typ)
	}
	period := "month"
	if byWeek {
		period = "week"
	}
	u.RawQuery = getQuery(fmt.Sprintf("%d", game), date, period).Encode()
	return u, nil
}

func GetCalendarURL(typ string, date time.Time, byWeek bool) (*url.URL, error) {
	if !ValidType(typ) {
		return nil, fmt.Errorf("invalid type: TL:%s", typ)
	}
	base, ok := baseURIs[typ]
	if !ok {
		return nil, fmt.Errorf("unknown base URI for type: %s", typ)
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("unable to parse base URI: %w", err)
	}
	path, ok := calendarPath[typ]
	if !ok {
		return nil, fmt.Errorf("unknown calendar path for type: %s", typ)
	}
	u.Path = path
	game, _ := calendarType[typ]
	period := "month"
	if byWeek {
		period = "week"
	}
	u.RawQuery = getQuery(fmt.Sprintf("%d", game), date, period).Encode()
	return u, nil
}
