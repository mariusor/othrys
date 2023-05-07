package post

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/McKael/madon"
	vocab "github.com/go-ap/activitypub"
	"github.com/mariusor/esports-calendar/internal/cmd"
)

const ExecOpenCmd = "xdg-open"

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
	Releases []release
}

func FormatDuration(d time.Duration) string {
	label := "hour"
	val := float64(d) / float64(time.Hour)
	if d > ResolutionDay {
		label = "day"
		val = float64(d) / float64(ResolutionDay)
	}
	if d > ResolutionWeek {
		label = "week"
		val = float64(d) / float64(ResolutionWeek)
	}
	if d > ResolutionMonthish {
		label = "month"
		val = float64(d) / float64(ResolutionMonthish)
	}
	if d > ResolutionYearish {
		label = "year"
		val = float64(d) / float64(ResolutionYearish)
	}
	if val != 1.0 && val != -1.0 {
		label = label + "s"
	}
	return fmt.Sprintf("%+.2g%s", val, label)
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

func removeStrings(s string, replace ...string) string {
	for _, r := range replace {
		s = strings.ReplaceAll(s, r, "")
	}
	return s
}

var (
	repl            = regexp.MustCompile("metal$")
	toRemoveStrings = []string{"(early)", "(later)", "(mid)", "-", " ", "#", "'"}
)

func tagNormalize(t string) string {
	hasHash := len(t) > 2 && t[0] == '#'
	if hasHash {
		t = t[1:]
	}
	if strings.EqualFold(t, "Post-Metal") {
		return "postmetal"
	}
	if strings.EqualFold(t, "Metal") {
		return "metal"
	}
	t = strings.ToLower(t)
	t = removeStrings(t, toRemoveStrings...)
	t = repl.ReplaceAllLiteralString(t, "")
	return t
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

func renderLinks(rel release) (string, error) {
	if len(rel.URI) == 0 {
		return "", fmt.Errorf("no URIs for current release")
	}
	contBuff := bytes.NewBuffer(nil)
	if err := linksTemplate.Execute(contBuff, rel); err != nil {
		return "", fmt.Errorf("unable to generate PostToMastodon %w", err)
	}
	return contBuff.String(), nil
}

func renderTitle(gd time.Time, rel []release) (string, error) {
	title := bytes.NewBuffer(nil)
	if err := titleTemplate.Execute(title, gd); err != nil {
		return "", fmt.Errorf("unable to build post content: %w", err)
	}
	return title.String(), nil
}

func renderPosts(d time.Time, rel []release) (string, error) {
	model := postContent{Date: d, Releases: rel}
	contBuff := bytes.NewBuffer(nil)
	if err := contTemplate.Execute(contBuff, model); err != nil {
		Error("unable to render post %s", err)
		return "", err
	}
	return contBuff.String(), nil
}

const unlisted = "unlisted"

type PosterFn func(releases map[time.Time][]release) error

func PostToMastodon(client *madon.Client, withLinks bool) PosterFn {
	if client == nil {
		return PostToStdout
	}
	return func(groupped map[time.Time][]release) error {
		var inReplyTo int64 = 0
		posts := make([]postModel, 0)
		linkPosts := make(map[int][]postModel)
		for d, releases := range groupped {
			title := bytes.NewBuffer(nil)
			if err := titleTemplate.Execute(title, d); err != nil {
				return fmt.Errorf("%s: unable to build post content: %w", client.InstanceURL, err)
			}

			cleaveFn := func(d time.Time, content *string) func(rel []release) bool {
				return func(rel []release) bool {
					var err error
					*content, err = renderPosts(d, rel)
					if err != nil {
						return false
					}
					return len(*content) < maxPostSize
				}
			}

			for {
				var current []release
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
				Info("Post at: %s", s.URI)
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

func ValidMastodonAuth(c *madon.Client) bool {
	return ValidMastodonApp(c) && c.UserToken != nil && c.UserToken.AccessToken != ""
}

func ValidMastodonApp(c *madon.Client) bool {
	return c != nil && c.Name != "" && c.ID != "" && c.Secret != "" && c.APIBase != "" && c.InstanceURL != ""
}

func saveMastodonCredentials(c *madon.Client, path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open file %w", err)
	}
	defer f.Close()

	d := gob.NewEncoder(f)
	return d.Encode(c)
}

func loadMastodonCredentials(c *madon.Client, path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to load credentials file %s: %w", path, err)
	}
	defer f.Close()
	d := gob.NewDecoder(f)
	return d.Decode(c)
}

func InstanceName(inst string) string {
	u, err := url.ParseRequestURI(inst)
	if err != nil {
		inst = u.Host
	}
	return url.PathEscape(filepath.Clean(filepath.Base(inst)))
}

func CheckMastodonCredentialsFile(key, secret, token, instance string, dryRun bool, getAccessTokenFn func() (string, error)) (*madon.Client, error) {
	app := new(madon.Client)
	path := filepath.Join(cmd.DataPath(), instance)
	if err := loadMastodonCredentials(app, path); err != nil {
		if len(key) > 0 && len(secret) > 0 {
			if len(key) > 0 {
				app.ID = key
			}
			if len(secret) > 0 {
				app.Secret = secret
			}
			app.Name = cmd.AppName
			app.InstanceURL = "https://" + instance
			app.APIBase = app.InstanceURL + "/api/v1"
			app.UserToken = new(madon.UserToken)
			if len(token) > 0 {
				app.UserToken.AccessToken = token
			}
		} else {
			if app, err = madon.NewApp(cmd.AppName, cmd.AppWebsite, cmd.AppScopes, "", instance); err != nil {
				return nil, fmt.Errorf("unable to initialize mastodon application: %w", err)
			}
		}
	}
	if ValidMastodonAuth(app) {
		return app, saveMastodonCredentials(app, filepath.Join(cmd.DataPath(), InstanceName(app.InstanceURL)))
	}
	if !dryRun {
		userAuthUri, err := app.LoginOAuth2("", nil)
		if err != nil {
			return nil, fmt.Errorf("unable to login to %s: %w", app.InstanceURL, err)
		}
		if err = exec.Command(ExecOpenCmd, userAuthUri).Run(); err != nil {
			Info("unable to use %s to open %s: %s", ExecOpenCmd, app.InstanceURL, err)
			fmt.Printf("Go to this URL in your browser: %s\n", userAuthUri)
		}
		if app.UserToken == nil {
			app.UserToken = new(madon.UserToken)
		}
		tok, err := getAccessTokenFn()
		if err != nil {
			return nil, fmt.Errorf("unable to login to %s: %w", app.InstanceURL, err)
		}
		if tok == "" {
			return nil, fmt.Errorf("empty authentication token")
		}
		app.UserToken.AccessToken = tok
		app.UserToken.CreatedAt = time.Now().UnixMilli()
		if !ValidMastodonAuth(app) {
			return nil, fmt.Errorf("unable to get user authorization")
		}

		if err := saveMastodonCredentials(app, app.InstanceURL); err != nil {
			Info("unable to save credentials: %s", err)
		}
	}
	return app, nil
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
