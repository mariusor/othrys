package plusforward

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const LabelQuakeLive = "qlv"
const LabelQuakeIV = "qiv"
const LabelQuakeIII = "qiii"
const LabelQuakeII = "qii"
const LabelQuakeWorld = "qw"
const LabelDiabotical = "dbt"
const LabelDoom = "doom"
const LabelReflex = "rfl"
const LabelGG = "gg"
const LabelUnreal = "ut"
const LabelWarsow = "wsw"
const LabelDbmb = "dbmb"
const LabelXonotic = "xnt"
const LabelQuakeChampions = "qch"
const LabelQuakeCPMA = "cpma"
const LabelOverwatch = "ovw"
const LabelPlusForward = "pfw"
const LabelUnknown = "unk"

var ValidTypes = [...]string{
	LabelQuakeLive,
	LabelQuakeIV,
	LabelQuakeIII,
	LabelQuakeII,
	LabelQuakeWorld,
	LabelDiabotical,
	LabelDoom,
	LabelReflex,
	LabelGG,
	LabelUnreal,
	LabelWarsow,
	LabelDbmb,
	LabelXonotic,
	LabelQuakeChampions,
	LabelQuakeCPMA,
	LabelOverwatch,
}

var baseURIs = map[string]string{
	LabelPlusForward:    "https://www.plusforward.net",
	LabelQuakeLive:      "https://www.plusforward.net",
	LabelQuakeIV:        "https://www.plusforward.net",
	LabelQuakeIII:       "https://www.plusforward.net",
	LabelQuakeII:        "https://www.plusforward.net",
	LabelQuakeWorld:     "https://www.plusforward.net",
	LabelDiabotical:     "https://www.plusforward.net",
	LabelDoom:           "https://www.plusforward.net",
	LabelReflex:         "https://www.plusforward.net",
	LabelOverwatch:      "https://www.plusforward.net",
	LabelGG:             "https://www.plusforward.net",
	LabelUnreal:         "https://www.plusforward.net",
	LabelWarsow:         "https://www.plusforward.net",
	LabelDbmb:           "https://www.plusforward.net",
	LabelXonotic:        "https://www.plusforward.net",
	LabelQuakeChampions: "https://www.plusforward.net",
	LabelQuakeCPMA:      "https://www.plusforward.net",
}

var calendarPath = map[string]string{
	LabelPlusForward:    "/calendar/",
	LabelQuakeLive:      "/calendar/",
	LabelQuakeIV:        "/calendar/",
	LabelQuakeIII:       "/calendar/",
	LabelQuakeII:        "/calendar/",
	LabelQuakeWorld:     "/calendar/",
	LabelDiabotical:     "/calendar/",
	LabelDoom:           "/calendar/",
	LabelReflex:         "/calendar/",
	LabelOverwatch:      "/calendar/",
	LabelGG:             "/calendar/",
	LabelUnreal:         "/calendar/",
	LabelWarsow:         "/calendar/",
	LabelDbmb:           "/calendar/",
	LabelXonotic:        "/calendar/",
	LabelQuakeChampions: "/calendar/",
	LabelQuakeCPMA:      "/calendar/",
}

var eventPath = map[string]string{
	LabelPlusForward:    "/calendar/manage/",
	LabelQuakeLive:      "/calendar/manage/",
	LabelQuakeIV:        "/calendar/manage/",
	LabelQuakeIII:       "/calendar/manage/",
	LabelQuakeII:        "/calendar/manage/",
	LabelQuakeWorld:     "/calendar/manage/",
	LabelDiabotical:     "/calendar/manage/",
	LabelDoom:           "/calendar/manage/",
	LabelReflex:         "/calendar/manage/",
	LabelOverwatch:      "/calendar/manage/",
	LabelGG:             "/calendar/manage/",
	LabelUnreal:         "/calendar/manage/",
	LabelWarsow:         "/calendar/manage/",
	LabelDbmb:           "/calendar/manage/",
	LabelXonotic:        "/calendar/manage/",
	LabelQuakeChampions: "/calendar/manage/",
	LabelQuakeCPMA:      "/calendar/manage/",
}

var calendarType = map[string]int{
	LabelUnknown:        0,
	LabelQuakeLive:      3,
	LabelQuakeIV:        4,
	LabelQuakeIII:       5,
	LabelQuakeII:        6,
	LabelQuakeWorld:     7,
	LabelDiabotical:     8,
	LabelDoom:           9,
	LabelReflex:         10,
	LabelOverwatch:      13,
	LabelGG:             14,
	LabelUnreal:         15,
	LabelWarsow:         16,
	LabelDbmb:           17,
	LabelXonotic:        18,
	LabelQuakeChampions: 20,
	LabelQuakeCPMA:      21,
}

func ValidType(typ string) bool {
	if typ == LabelPlusForward {
		return true
	}
	for _, t := range ValidTypes {
		if strings.ToLower(typ) == t {
			return true
		}
	}
	return false
}

func getQuery(typ string, date time.Time, by string) url.Values {
	q := make(url.Values)
	q.Add("view", by)
	q.Add("year", date.Format("2006"))
	q.Add("month", date.Format("01"))
	q.Add("day", date.Format("02"))
	q.Add("ongoing", "1")
	if typ != "0" {
		q.Add("cat", typ)
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
		return nil, fmt.Errorf("invalid type: PF:%s", typ)
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
