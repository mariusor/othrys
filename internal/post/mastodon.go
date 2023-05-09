package post

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/McKael/madon"
	othrys "github.com/mariusor/esports-calendar"
	"github.com/mariusor/esports-calendar/calendar"
)

const maxPostSize = 500
const mastodonTitleTpl = `Events for {{ .Format "Monday, 02 Jan 2006" -}}`
const mastodonContentTpl = `{{- range $event := .Events }}
{{ $event | sanitize }} {{ renderTags $event.TagNames "#" }}
{{ end }}
#{{ .Date.Month.String | lower }} {{range $typ := .Types }} #{{ $typ}}{{ end }} #esports #calendar`

const linksTpl = `{{ . | sanitize }}
{{- range $link := .URI }}
 {{ $link }}
{{- end }}
{{range $typ := .Types }} #{{ $typ}}{{ end }} #esports #calendar`

var badStrings = []string{"​"}

var (
	toRemoveStrings = []string{"(early)", "(later)", "(mid)", "-", " ", "#", "'"}
)

func removeStrings(s string, replace ...string) string {
	for _, r := range replace {
		s = strings.ReplaceAll(s, r, "")
	}
	return s
}

func sanitize(r calendar.Event) string {
	return removeStrings(r.String(), badStrings...)
}

var linksTemplate = template.Must(template.New("daily-links").
	Funcs(template.FuncMap{
		"sanitize": sanitize,
		"lower":    strings.ToLower,
	}).Parse(linksTpl))

var contTemplate = template.Must(template.New("daily-PostToMastodon").
	Funcs(template.FuncMap{
		"sanitize":   sanitize,
		"lower":      strings.ToLower,
		"renderTags": renderTagsText,
	}).Parse(mastodonContentTpl))

var titleTemplate = template.Must(template.New("daily-PostToMastodon-title").
	Funcs(template.FuncMap{
		"sanitize": sanitize,
	}).Parse(mastodonTitleTpl))

type postContent struct {
	tags   []string
	Date   time.Time
	Types  []string
	Events calendar.Events
}

func stringsContain(sl []string, v string) bool {
	for _, vs := range sl {
		if vs == v {
			return true
		}
	}
	return false
}

func uniqueValues[T comparable](sl []T, containsFn func(sl []T, u T) bool) []T {
	newSl := make([]T, 0, len(sl))
	for _, v := range sl {
		if !containsFn(newSl, v) {
			newSl = append(newSl, v)
		}
	}
	return newSl
}

func (c postContent) Tags() []string {
	tags := make([]string, 0)
	for _, r := range c.Events {
		tags = append(tags, r.TagNames...)
	}
	for i, t := range tags {
		tags[i] = othrys.TagNormalize(t)
	}
	return uniqueValues(tags, stringsContain)
}

type postModel struct {
	title, content string
}

func renderTitle(gd time.Time, rel calendar.Events) (string, error) {
	title := bytes.NewBuffer(nil)
	if err := titleTemplate.Execute(title, gd); err != nil {
		return "", fmt.Errorf("unable to build post content: %w", err)
	}
	return title.String(), nil
}

func renderPosts(d time.Time, rel calendar.Events) (string, error) {
	model := postContent{Date: d, Events: rel}
	contBuff := bytes.NewBuffer(nil)
	if err := contTemplate.Execute(contBuff, model); err != nil {
		infFn("unable to render post %s", err)
		return "", err
	}
	return contBuff.String(), nil
}

const unlisted = "unlisted"

type PosterFn func(events map[time.Time]calendar.Events) error

func PostToMastodon(client *madon.Client) PosterFn {
	if client == nil {
		return PostToStdout
	}
	return func(group map[time.Time]calendar.Events) error {
		var inReplyTo int64 = 0
		posts := make([]postModel, 0)

		for d, events := range group {
			title, err := renderTitle(d, events)
			if err != nil {
				errFn("Unable to render title: %s", err)
			}

			cleaveFn := func(d time.Time, content *string) func(rel []calendar.Event) bool {
				return func(rel []calendar.Event) bool {
					var err error
					*content, err = renderPosts(d, rel)
					if err != nil {
						return false
					}
					return len(*content) < maxPostSize
				}
			}

			for {
				var content string
				_, events = cleaveSlice(events, cleaveFn(d, &content))

				posts = append(posts, postModel{title: title, content: content})
				if events == nil {
					break
				}
			}
		}

		for i, model := range posts {
			if len(posts) > 1 {
				model.title = fmt.Sprintf("%s: %d/%d", model.title, i+1, len(posts))
			}
			if inReplyTo > 0 {
				time.Sleep(500 * time.Millisecond)
			}
			s, err := client.PostStatus(model.content, inReplyTo, nil, len(model.title) > 0, model.title, unlisted)
			if err != nil {
				return fmt.Errorf("%s: %w", client.InstanceURL, err)
			} else {
				infFn("Post at: %s", s.URI)
			}
		}

		return nil
	}
}

func InstanceName(inst string) string {
	u, err := url.ParseRequestURI(inst)
	if err != nil {
		inst = u.Host
	}
	return url.PathEscape(filepath.Clean(filepath.Base(inst)))
}

func splitSlice[T any](sl []T, size int) [][]T {
	result := make([][]T, 0)
	if len(sl) <= size {
		result = append(result, sl)
		return result
	}
	if size == 0 {
		size = 1
	}
	cur := 0
	end := size
	for {
		if cur+size < len(sl) {
			end = cur + size
		} else {
			end = len(sl)
		}
		chunk := sl[cur:end]
		cur += size
		result = append(result, chunk)
		if cur >= len(sl) {
			break
		}
	}
	return result
}

func cleaveSlice[T any](incoming []T, checkFn func([]T) bool) ([]T, []T) {
	if checkFn(incoming) {
		return incoming, nil
	}

	var remainder []T
	for {
		cleaveLen := len(incoming) / 2
		halves := splitSlice[T](incoming, cleaveLen)
		if len(halves) >= 2 {
			for _, h := range halves[1:] {
				remainder = append(remainder, h...)
			}
		}
		if checkFn(halves[0]) {
			return halves[0], remainder
		}
		if len(halves[0]) == len(incoming) {
			break
		}
		incoming = halves[0]
	}
	return incoming, nil
}
