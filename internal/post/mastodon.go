package post

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/mariusor/esports-calendar/calendar"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/McKael/madon"
	vocab "github.com/go-ap/activitypub"
)

const maxPostSize = 500
const titleTpl = `Events for {{ .Format "Monday, 02 Jan 2006" -}}`
const releaseTpl = `{{- range $event := .Events }}
{{ $event | sanitize }} {{ renderTags $event.TagNames "#" }}
{{ end }}
#{{ .Date.Month.String | lower }} {{range $typ := .Types }} #{{ $typ}}{{ end }} #esports #calendar`

const linksTpl = `{{ . | sanitize }}
{{- range $link := .URI }}
 {{ $link }}
{{- end }}
{{range $typ := .Types }} #{{ $typ}}{{ end }} #esports #calendar`

var badStrings = []string{"​"}

type release struct {
	tags     vocab.ItemCollection
	URI      string
	TagNames []string
}

func (r release) String() string {
	return ""
}

func sanitize(r release) string {
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
	}).Parse(releaseTpl))

var titleTemplate = template.Must(template.New("daily-PostToMastodon-title").
	Funcs(template.FuncMap{
		"sanitize": sanitize,
	}).Parse(titleTpl))

type postContent struct {
	tags     []string
	Date     time.Time
	Releases calendar.Events
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
	for _, r := range c.Releases {
		tags = append(tags, r.TagNames...)
	}
	for i, t := range tags {
		tags[i] = tagNormalize(t)
	}
	return uniqueValues(tags, stringsContain)
}

type postModel struct {
	title, content string
}

func renderLinks(rel calendar.Event) (string, error) {
	if len(rel.Links) == 0 {
		return "", fmt.Errorf("no URIs for current release")
	}
	contBuff := bytes.NewBuffer(nil)
	if err := linksTemplate.Execute(contBuff, rel); err != nil {
		return "", fmt.Errorf("unable to generate PostToMastodon %w", err)
	}
	return contBuff.String(), nil
}

func renderTitle(gd time.Time, rel calendar.Events) (string, error) {
	title := bytes.NewBuffer(nil)
	if err := titleTemplate.Execute(title, gd); err != nil {
		return "", fmt.Errorf("unable to build post content: %w", err)
	}
	return title.String(), nil
}

func renderPosts(d time.Time, rel calendar.Events) (string, error) {
	model := postContent{Date: d, Releases: rel}
	contBuff := bytes.NewBuffer(nil)
	if err := contTemplate.Execute(contBuff, model); err != nil {
		infFn("unable to render post %s", err)
		return "", err
	}
	return contBuff.String(), nil
}

const unlisted = "unlisted"

type PosterFn func(releases map[time.Time]calendar.Events) error

func PostToMastodon(client *madon.Client, withLinks bool) PosterFn {
	if client == nil {
		return PostToStdout
	}
	return func(groupped map[time.Time]calendar.Events) error {
		var inReplyTo int64 = 0
		posts := make([]postModel, 0)
		linkPosts := make(map[int][]postModel)
		for d, releases := range groupped {
			title := bytes.NewBuffer(nil)
			if err := titleTemplate.Execute(title, d); err != nil {
				return fmt.Errorf("%s: unable to build post content: %w", client.InstanceURL, err)
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
				var current calendar.Events
				var content string
				current, releases = cleaveSlice(releases, cleaveFn(d, &content))

				posts = append(posts, postModel{title: title.String(), content: content})
				if withLinks {
					curIndex := len(posts) - 1
					for _, rel := range current {
						if linksContent, err := renderLinks(rel); err == nil {
							linkPosts[curIndex] = append(linkPosts[curIndex], postModel{title: "", content: linksContent})
						}
					}
				}

				if releases == nil {
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

func loadStaticFile(AccountDetails fs.FS, f string) ([]byte, error) {
	desc, err := AccountDetails.Open(f)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(desc)
}

func UpdateMastodonAccount(app *madon.Client, a fs.FS, dryRun bool) error {
	var namePtr, descPtr, avatarPtr, hdrPtr *string
	if data, _ := loadStaticFile(a, "static/name.txt"); data != nil {
		name := strings.TrimSpace(string(data))
		namePtr = &name
	}
	if data, _ := loadStaticFile(a, "static/description.txt"); data != nil {
		description := strings.TrimSpace(string(data))
		descPtr = &description
	}
	if data, _ := loadStaticFile(a, "static/avatar.png"); data != nil {
		avatar := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(data))
		avatarPtr = &avatar
	}
	if data, _ := loadStaticFile(a, "static/header.png"); data != nil {
		hdr := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(data))
		hdrPtr = &hdr
	}
	if !dryRun {
		if _, err := app.UpdateAccount(namePtr, descPtr, avatarPtr, hdrPtr, nil); err != nil {
			return err
		}
	}
	return nil
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
