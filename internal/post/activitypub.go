package post

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	"github.com/McKael/madon"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/mariusor/render"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	othrys "github.com/mariusor/esports-calendar"
	"github.com/mariusor/esports-calendar/calendar"
)

const eventTitleTpl = `{{ if gt (len .Event.Category) 0}}{{.Event.Category}}: {{ end }}{{ .Event.Stage }}`
const eventContentTpl = `{{ if gt (len .Event.Content) 0}}<div>{{ .Event.Content }}</div>{{ end }}`

var (
	defaultRenderOptions = render.Options{
		Layout:     "main",
		Extensions: []string{".html"},
		Funcs: []template.FuncMap{{
			"sanitize":     sanitize,
			"lower":        strings.ToLower,
			"tagNormalize": othrys.TagNormalize,
		}},
		Delims:                    render.Delims{Left: "{{", Right: "}}"},
		Charset:                   "UTF-8",
		DisableCharset:            false,
		HTMLContentType:           "text/html",
		DisableHTTPErrorRendering: true,
		RequirePartials:           true,
	}

	errRenderer = render.New(defaultRenderOptions)
	ren         = render.New(defaultRenderOptions)

	contHTMLTemplate = template.Must(template.New("daily-ToActivityPub").
				Funcs(template.FuncMap{
			"sanitize":   sanitize,
			"lower":      strings.ToLower,
			"renderTag":  renderTagHTML,
			"commonTags": commonTags,
		}).Parse(eventContentTpl))

	titleHTMLTemplate = template.Must(template.New("daily-ToActivityPub-title").
				Funcs(template.FuncMap{
			"sanitize": sanitize,
		}).Parse(eventTitleTpl))
)

var infFn client.LogFn = func(s string, i ...interface{}) {}
var errFn client.LogFn = func(s string, i ...interface{}) {}

func maxItems(max int) client.FilterFn {
	v := url.Values{}
	v.Add("maxItems", strconv.Itoa(max))
	return func() url.Values {
		return v
	}
}

func typeFilter(types ...string) client.FilterFn {
	v := url.Values{}
	for _, name := range types {
		v.Add("type", name)
	}
	return func() url.Values {
		return v
	}
}

func withTagObjects() url.Values {
	v := url.Values{}
	v.Add("object.type", "")
	return v
}

func newActivityPubTag(tag string, baseURL vocab.IRI) *vocab.Object {
	tag = "#" + othrys.TagNormalize(tag)
	t := new(vocab.Object)
	t.Name = nl(tag)
	t.To.Append(vocab.PublicNS)
	t.ID = baseURL.AddPath(strings.TrimPrefix(tag, "#"))
	return t
}

func apTags(rel calendar.Event, baseURL vocab.IRI) vocab.ItemCollection {
	names := make([]string, 0)
	for _, tag := range rel.TagNames {
		names = append(names, tag)
	}

	tags := make(vocab.ItemCollection, 0)
	for _, tag := range names {
		if t := newActivityPubTag(tag, baseURL); !tags.Contains(t) {
			tags = append(tags, t)
		}
	}
	return tags
}

func acceptFollows(actor *vocab.Actor, cl client.PubClient) error {
	inbox, err := cl.Inbox(context.Background(), actor, typeFilter("Follow"), maxItems(100))
	if err != nil {
		return err
	}
	followers, err := cl.Followers(context.Background(), actor)
	if err != nil {
		return err
	}
	followerIRIs := make(vocab.IRIs, 0)
	vocab.OnCollectionIntf(followers, func(col vocab.CollectionInterface) error {
		for _, fol := range col.Collection() {
			followerIRIs = append(followerIRIs, fol.GetLink())
		}
		return nil
	})

	toSend := make([]*vocab.Activity, 0)
	vocab.OnCollectionIntf(inbox, func(col vocab.CollectionInterface) error {
		for _, act := range col.Collection() {
			if act.GetType() != vocab.FollowType {
				continue
			}
			skip := false
			vocab.OnActivity(act, func(follow *vocab.Activity) error {
				skip = followerIRIs.Contains(follow.Actor.GetLink())
				if !skip {
					infFn("Accepting Follow request from: %s", follow.Actor.GetLink())
				}
				return nil
			})
			if skip {
				continue
			}

			accept := new(vocab.Activity)
			accept.Type = vocab.AcceptType
			accept.CC = append(accept.CC, vocab.PublicNS)
			accept.Actor = actor
			accept.InReplyTo = act.GetID()
			accept.Object = act.GetID()
			toSend = append(toSend, accept)
		}
		return nil
	})

	g, ctx := errgroup.WithContext(context.Background())
	for _, accept := range toSend {
		g.Go(func() error {
			if _, _, err := cl.ToOutbox(ctx, accept); err != nil {
				errFn("Failed accepting follow: %+s", err)
			}
			return nil
		})
	}
	return g.Wait()
}

func defaultActivityPubTags(date time.Time, baseURL vocab.IRI) vocab.ItemCollection {
	return vocab.ItemCollection{
		newActivityPubTag(strings.ToLower(date.Month().String()), baseURL),
		newActivityPubTag("esports", baseURL),
		newActivityPubTag("calendar", baseURL),
	}
}

type apContent struct {
	Event calendar.Event
	Tags  vocab.ItemCollection
}

func renderEventTitle(rel calendar.Event) (string, error) {
	model := apContent{Event: rel}
	title := bytes.NewBuffer(nil)
	if err := titleHTMLTemplate.Execute(title, model); err != nil {
		return "", fmt.Errorf("unable to build post content: %w", err)
	}
	return title.String(), nil
}

func renderEventContent(rel calendar.Event, tags vocab.ItemCollection) (string, error) {
	model := apContent{Event: rel, Tags: tags}
	contBuff := bytes.NewBuffer(nil)
	if err := contHTMLTemplate.Execute(contBuff, model); err != nil {
		errFn("unable to render post %s", err)
		return "", err
	}
	return contBuff.String(), nil
}

func loadTagsToEvent(rel calendar.Event, tags vocab.ItemCollection) (calendar.Event, vocab.ItemCollection) {
	remainingTags := make(vocab.ItemCollection, 0)

	rel.Tags = make(vocab.ItemCollection, 0)
	for _, t := range tags {
		for _, tag := range rel.TagNames {
			tag = "#" + othrys.TagNormalize(tag)
			tagName := othrys.NameOf(t)
			if strings.EqualFold(tag, tagName) && !rel.Tags.Contains(t) {
				rel.Tags = append(rel.Tags, t)
			}
		}
	}
found:
	for _, t := range tags {
		if rel.Tags.Contains(t) {
			continue found
		}
		if !remainingTags.Contains(t) {
			remainingTags = append(remainingTags, t)
		}
	}
	return rel, remainingTags
}

func equalOrInCollection(toCheck, with vocab.Item) bool {
	if vocab.IsItemCollection(toCheck) {
		return false
	}
	if vocab.IsItemCollection(with) {
		inCollection := false
		vocab.OnItemCollection(with, func(col *vocab.ItemCollection) error {
			for _, it := range *col {
				if equalOrInCollection(toCheck, it) {
					inCollection = true
					break
				}
			}
			return nil
		})
		return inCollection
	}
	urlSame := with.GetLink().Equals(toCheck.GetLink(), true)
	nameSame := strings.EqualFold(othrys.NameOf(with), othrys.NameOf(toCheck))
	return urlSame && nameSame
}

func removeExistingTags(ctx context.Context, client client.PubGetter, actor *vocab.Actor, tags vocab.ItemCollection) (vocab.ItemCollection, error) {
	col, err := client.Outbox(ctx, actor, typeFilter(string(vocab.CreateType)), withTagObjects)
	if err != nil {
		return nil, err
	}

	tagsToCreate := make(vocab.ItemCollection, 0)
	for _, tag := range tags {
		needsCreating := true
		for _, it := range col.Collection() {
			var ob vocab.Item
			vocab.OnActivity(it, func(act *vocab.Activity) error {
				ob = act.Object
				return nil
			})
			if equalOrInCollection(tag, ob) {
				needsCreating = false
				break
			}
		}
		if needsCreating && !tagsToCreate.Contains(tag) {
			tagsToCreate = append(tagsToCreate, tag)
		}
	}
	return tagsToCreate, nil
}

func ToActivityPub(cl *APClient) PosterFn {
	logger := lw.Dev()

	tok := cl.Tok.AccessToken
	oauth := cl.Conf.Client(context.Background(), cl.Tok)
	ap := client.New(
		client.WithHTTPClient(oauth),
		client.WithLogger(logger),
	)

	errFn = logger.Errorf
	infFn = logger.Infof

	c, cancelFn := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn()

	actor, err := ap.Actor(c, cl.ID)
	if err != nil {
		errFn("%s, falling back to just printing", err)
		return ToStdout
	}

	if err := acceptFollows(actor, ap); err != nil {
		errFn("failed to accept follows for actor: %s", err)
	}

	ctx := context.Background()

	return func(group map[time.Time]calendar.Events) error {
		activities := make([]vocab.Activity, 0)
		for gd, events := range group {
			object := make(vocab.ItemCollection, 0)
			for _, event := range events {
				var globalTags vocab.ItemCollection

				ob := new(vocab.Event)
				ob.Type = vocab.EventType

				ob.StartTime = event.StartTime
				ob.EndTime = event.StartTime.Add(event.Duration)
				ob.Duration = event.Duration
				ob.Updated = event.LastModified

				tags := append(defaultActivityPubTags(event.StartTime, actor.ID), apTags(event, actor.ID)...)
				toCreateTags, err := removeExistingTags(ctx, ap, actor, tags)
				if err != nil {
					infFn("Error when loading tags from server: %s", err)
				}
				if len(toCreateTags) > 0 {
					activities = append(activities, othrys.WrapObjectInCreate(*actor, toCreateTags))
				}

				event, globalTags = loadTagsToEvent(event, tags)

				content, err := renderEventContent(event, globalTags)
				if err != nil {
					errFn("Unable to render HTML object: %s", err)
					continue
				}

				if len(content) > 0 {
					ob.Content = othrys.NL(content)
				}
				ob.Tag = tags

				title, err := renderEventTitle(event)
				if err == nil {
					ob.Name = othrys.NL(title)
				}
				if source, err := renderPosts(gd, calendar.Events{event}); err == nil {
					ob.Source = vocab.Source{
						MediaType: "text/markdown",
						Content:   othrys.NL(source),
					}
				}

				ob.To = vocab.ItemCollection{vocab.PublicNS}
				ob.CC = vocab.ItemCollection{vocab.Followers.Of(actor)}

				object = append(object, ob)
			}
			activities = append(activities, othrys.WrapObjectInCreate(*actor, object))
		}
		(OperationsBatch{AP: ap, Ops: activities}).Send()

		if tr, ok := oauth.Transport.(*oauth2.Transport); ok {
			cl.Tok, err = tr.Source.Token()
			if cl.Tok.AccessToken == tok {
				return nil
			}
			if err != nil {
				errFn("Unable to refresh OAuth2 token: %s", err)
			} else {
				if err := saveCredentials(cl, filepath.Join(cl.Type, InstanceName(cl.ID.String()))); err != nil {
					errFn("Unable to save new credentials for %s: %s", cl.ID, err)
				}
				infFn("Refreshed OAuth2 credentials %s", cl.ID)
			}
		}
		return nil
	}
}

type APClient struct {
	ID    vocab.IRI
	Types []string
	Type  string
	Conf  oauth2.Config
	Tok   *oauth2.Token
}

func GetHTTPClient() *http.Client {
	cl := http.DefaultClient

	if cl.Transport == nil {
		cl.Transport = &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 20,
			DialContext: (&net.Dialer{
				// This is the TCP connect timeout in this instance.
				Timeout: 2500 * time.Millisecond,
			}).DialContext,
			TLSHandshakeTimeout: 2500 * time.Millisecond,
		}
	}
	if tr, ok := cl.Transport.(*http.Transport); ok {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = new(tls.Config)
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	if tr, ok := cl.Transport.(*oauth2.Transport); ok {
		if tr, ok := tr.Base.(*http.Transport); ok {
			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = new(tls.Config)
			}
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
	}
	return cl
}

type OperationsBatch struct {
	AP  client.PubSubmitter
	Ops []vocab.Activity
}

func (b OperationsBatch) Send() {
	for _, act := range b.Ops {
		_, created, err := b.AP.ToOutbox(context.Background(), act)
		if err != nil {
			errFn("%+s", err)
		} else {
			infFn("Created object: %s", created.GetLink())
		}
	}
}

func saveCredentials(cl any, path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open file %w", err)
	}
	defer f.Close()

	d := gob.NewEncoder(f)
	return d.Encode(cl)
}

func LoadCredentials(path string) (map[string]any, error) {
	creds := make(map[string]any)

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, cl := range []any{new(APClient), new(madon.Client)} {
			if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(cl); err != nil {
				continue
			}
			creds[filepath.Base(path)] = cl
		}
		return nil
	})

	return creds, err
}
