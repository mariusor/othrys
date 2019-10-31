package plusforward

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const LabelQuakeLive = "qlv"
const LabelQIV = "qiv"
const LabelQIII = "qiii"
const LabelQII = "qii"
const LabelQuakeWorld = "qw"
const LabelDiabotical = "dbt"
const LabelDoom = "doom"
const LabelReflex = "rfl"
const LabelGG = "gg"
const LabelUnreal = "ut"
const LabelWarsow = "wsw"
const LabelDbmb = "dbmb"
const LabelXonotic = "xnt"
const LabelQChampions = "qch"
const LabelQuakeCPMA = "cpma"
const LabelPlusForward = "pfw"
const LabelOverwatch = "ovw"
const LabelUnknown = "unk"

var ValidTypes = [...]string{
	LabelQuakeLive,
	LabelQIV,
	LabelQIII,
	LabelQII,
	LabelQuakeWorld,
	LabelDiabotical,
	LabelDoom,
	LabelReflex,
	LabelGG,
	LabelUnreal,
	LabelWarsow,
	LabelDbmb,
	LabelXonotic,
	LabelQChampions,
	LabelQuakeCPMA,
	LabelOverwatch,
}

var baseURIs = map[string]string{
	LabelPlusForward: "https://www.plusforward.net",
	LabelQuakeLive:   "https://www.plusforward.net",
	LabelQIV:         "https://www.plusforward.net",
	LabelQIII:        "https://www.plusforward.net",
	LabelQII:         "https://www.plusforward.net",
	LabelQuakeWorld:  "https://www.plusforward.net",
	LabelDiabotical:  "https://www.plusforward.net",
	LabelDoom:        "https://www.plusforward.net",
	LabelReflex:      "https://www.plusforward.net",
	LabelOverwatch:   "https://www.plusforward.net",
	LabelGG:          "https://www.plusforward.net",
	LabelUnreal:      "https://www.plusforward.net",
	LabelWarsow:      "https://www.plusforward.net",
	LabelDbmb:        "https://www.plusforward.net",
	LabelXonotic:     "https://www.plusforward.net",
	LabelQChampions:  "https://www.plusforward.net",
	LabelQuakeCPMA:   "https://www.plusforward.net",
}

var calendarPath = map[string]string{
	LabelPlusForward: "/calendar/",
	LabelQuakeLive:   "/calendar/",
	LabelQIV:         "/calendar/",
	LabelQIII:        "/calendar/",
	LabelQII:         "/calendar/",
	LabelQuakeWorld:  "/calendar/",
	LabelDiabotical:  "/calendar/",
	LabelDoom:        "/calendar/",
	LabelReflex:      "/calendar/",
	LabelOverwatch:   "/calendar/",
	LabelGG:          "/calendar/",
	LabelUnreal:      "/calendar/",
	LabelWarsow:      "/calendar/",
	LabelDbmb:        "/calendar/",
	LabelXonotic:     "/calendar/",
	LabelQChampions:  "/calendar/",
	LabelQuakeCPMA:   "/calendar/",
}

var eventPath = map[string]string{
	LabelPlusForward: "/calendar/manage/",
	LabelQuakeLive:   "/calendar/manage/",
	LabelQIV:         "/calendar/manage/",
	LabelQIII:        "/calendar/manage/",
	LabelQII:         "/calendar/manage/",
	LabelQuakeWorld:  "/calendar/manage/",
	LabelDiabotical:  "/calendar/manage/",
	LabelDoom:        "/calendar/manage/",
	LabelReflex:      "/calendar/manage/",
	LabelOverwatch:   "/calendar/manage/",
	LabelGG:          "/calendar/manage/",
	LabelUnreal:      "/calendar/manage/",
	LabelWarsow:      "/calendar/manage/",
	LabelDbmb:        "/calendar/manage/",
	LabelXonotic:     "/calendar/manage/",
	LabelQChampions:  "/calendar/manage/",
	LabelQuakeCPMA:   "/calendar/manage/",
}

var calendarType = map[string]int{
	LabelPlusForward: 3,
	LabelQuakeLive:   4,
	LabelQIV:         5,
	LabelQIII:        6,
	LabelQII:         7,
	LabelQuakeWorld:  8,
	LabelDiabotical:  9,
	LabelDoom:        10,
	LabelReflex:      13,
	LabelOverwatch:   14,
	LabelGG:          15,
	LabelUnreal:      16,
	LabelWarsow:      17,
	LabelDbmb:        18,
	LabelXonotic:     20,
	LabelQChampions:  21,
}

func ValidType(typ string) bool {
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
	q.Add("game", typ)
	return q
}

func GetEventURL(date time.Time, typ string, byWeek bool) (*url.URL, error) {
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

func GetCalendarURL(date time.Time, typ string, byWeek bool) (*url.URL, error) {
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
	path, ok := calendarPath[typ]
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
