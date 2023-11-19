package gcn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL = "https://www.globalcyclingnetwork.com/"
)

func LoadEvents(u *url.URL, date time.Time) (events, error) {
	if u == nil {
		return nil, fmt.Errorf("nil URL received")
	}

	// NOTE(marius): this looks useless as the table is loaded using an xhr request to
	// https://hk2y3slq.apicdn.sanity.io/v2021-06-09/data/query/production?query=*%5B_type%20%3D%3D%20%27raceEdition%27%20%26%26%20(raceDescription.dateStart%20%3E%3D%20%24startDate%20%7C%7C%20raceDescription.dateFinish%20%3E%3D%20%24startDate)%20%26%26%20(raceDescription.dateFinish%20%3C%3D%20%24endDate%20%7C%7C%20raceDescription.dateStart%20%3C%3D%20%24endDate)%5D%20%7C%20order(raceDescription.dateStart%20asc)%20%7B%0A%20%20%0A%20%20_type%2C%0A%20%20%22name%22%3A%20raceName%2C%0A%20%20%22id%22%3A%20editionId%2C%0A%20%20%22startDate%22%3A%20raceDescription.dateStart%2C%0A%20%20%22endDate%22%3A%20raceDescription.dateFinish%2C%0A%20%20%22classification%22%3A%20raceDescription.classificationLabel%2C%0A%20%20%22discipline%22%3A%20raceDescription.discipline%2C%0A%20%20%22country%22%3A%20raceDescription.nation%2C%0A%20%20%22nationIso%22%3A%20raceDescription.nationIso%2C%0A%20%20%22gender%22%3A%20raceDescription.gender%2C%0A%20%20slug%2C%0A%0A%7D&%24dateRange=%7B%22start%22%3A%222023-12-01T00%3A00%3A00.000Z%22%2C%22end%22%3A%222023-12-31T23%3A59%3A59.000Z%22%7D&%24startDate=%222023-12-01T00%3A00%3A00.000Z%22&%24endDate=%222023-12-31T23%3A59%3A59.000Z%22
	query := `*[_type == 'raceEdition' && (raceDescription.dateStart >= $startDate || raceDescription.dateFinish >= $startDate) && (raceDescription.dateFinish <= $endDate || raceDescription.dateStart <= $endDate)] | order(raceDescription.dateStart asc) {
  _type,
  "name": raceName,
  "id": editionId,
  "startDate": raceDescription.dateStart,
  "endDate": raceDescription.dateFinish,
  "classification": raceDescription.classificationLabel,
  "discipline": raceDescription.discipline,
  "country": raceDescription.nation,
  "nationIso": raceDescription.nationIso,
  "gender": raceDescription.gender,
  slug,
}`

	q := url.Values{}
	q.Add("query", query)
	// $dateRange={"start":"2023-12-01T00:00:00.000Z","end":"2023-12-31T23:59:59.000Z"}&$startDate="2023-12-01T00:00:00.000Z"&$endDate="2023-12-31T23:59:59.000Z"
	startDate := date.Format("2006-01-02T15:04:05.999Z")
	endDate := date.Add(31 * 24 * time.Hour).Format("2006-01-02T15:04:05.999")
	dateRange := fmt.Sprintf(`{"start":"%s","end":"%s"}`, startDate, endDate)
	q.Add("$dateRange", dateRange)
	q.Add("$startDate", fmt.Sprintf("%q", startDate))
	q.Add("$endDate", fmt.Sprintf("%q", endDate))

	u, _ = url.ParseRequestURI("https://hk2y3slq.apicdn.sanity.io/v2021-06-09/data/query/production")
	u.RawQuery = q.Encode()

	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	raw := bytes.Buffer{}
	_, err = raw.ReadFrom(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body: %w", err)
	}

	r := Response{}
	err = json.Unmarshal(raw.Bytes(), &r)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json body: %w", err)
	}

	evs := make(events, 0)
	for _, race := range r.Result {
		evs = append(evs, event{
			CalID:          race.ID,
			StartTime:      race.StartDate,
			EndTime:        race.EndDate,
			LastModified:   time.Now().UTC(),
			Name:           race.Name,
			Discipline:     race.Discipline,
			Classification: race.Classification,
			Links:          []string{fmt.Sprintf("%s%s", BaseURL, race.Slug.Current)},
			Canceled:       false,
			TagNames:       []string{race.Gender, race.NationIso, race.Country},
		})
	}
	return evs, nil
}

type Slug struct {
	Current string
}

type Object struct {
	Classification string
	Country        string
	Discipline     string
	NationIso      string
	Gender         string
	Type           string `json:"_type"`
	Name           string
	ID             int64
	Slug           Slug
	StartDate      time.Time
	EndDate        time.Time
}

type Response struct {
	Query          string
	Result         []Object
	ResponseTimeMs int `json:"ms"`
}
