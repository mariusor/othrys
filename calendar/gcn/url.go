package gcn

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const LabelGravel = "gr"
const LabelCX = "cx"
const LabelRoad = "road"
const LabelMTB = "mtb"
const LabelGCN = "gcn"

const LabelUnknown = "unk"

var ValidTypes = [...]string{
	LabelGravel,
	LabelCX,
	LabelRoad,
	LabelMTB,
}

func GetCalendarURL(typ string, date time.Time) (*url.URL, error) {
	if !ValidType(typ) {
		return nil, fmt.Errorf("invalid discipline: TL:%s", typ)
	}
	u, err := url.ParseRequestURI("https://www.globalcyclingnetwork.com/racing/races")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("year", strconv.Itoa(date.Year()))
	q.Add("month", strconv.Itoa(int(date.Month())))
	if typ != LabelGCN {
		q.Add("type", calendarType[typ])
	}
	u.RawQuery = q.Encode()
	return u, nil
}

var calendarType = map[string]string{
	LabelRoad:   "Road",
	LabelMTB:    "Mountain Biking",
	LabelCX:     "Cyclocross",
	LabelGravel: "Gravel",
}

func Label(discipline string) string {
	for d, v := range calendarType {
		if strings.EqualFold(d, discipline) {
			return v
		}
	}
	return LabelUnknown
}

func ValidType(typ string) bool {
	if typ == LabelGCN {
		return true
	}
	for _, t := range ValidTypes {
		if strings.ToLower(typ) == t {
			return true
		}
	}
	return false
}
