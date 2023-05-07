package post

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/tagextractor"
	"github.com/McKael/madon"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	"github.com/mariusor/render"
	"gitlab.com/golang-commonmark/markdown"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/mariusor/esports-calendar/calendar"
)

const releaseHTMLTpl = ``

var (
	defaultRenderOptions = render.Options{
		Layout:     "main",
		Extensions: []string{".html"},
		Funcs: []template.FuncMap{{
			"sanitize":     sanitize,
			"lower":        strings.ToLower,
			"tagNormalize": tagNormalize,
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

	contHTMLTemplate = template.Must(template.New("daily-PostToActivityPub").
				Funcs(template.FuncMap{
			"sanitize":   sanitize,
			"lower":      strings.ToLower,
			"renderTag":  renderTagHTML,
			"commonTags": commonTags,
		}).Parse(releaseHTMLTpl))
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
	tag = "#" + tagNormalize(tag)
	t := new(vocab.Object)
	t.Name = nl(tag)
	t.To.Append(vocab.PublicNS)
	t.ID = baseURL.AddPath(strings.TrimPrefix(tag, "#"))
	return t
}

func apTags(releases calendar.Events, baseURL vocab.IRI) vocab.ItemCollection {
	if len(releases) == 0 {
		return nil
	}
	names := make([]string, 0)
	for _, rel := range releases {
		for _, tag := range rel.TagNames {
			names = append(names, tag)
		}
	}

	tags := make(vocab.ItemCollection, 0)
	for _, tag := range names {
		if t := newActivityPubTag(tag, baseURL); !tags.Contains(t) {
			tags = append(tags, t)
		}
	}
	return tags
}

func setIDOf(it vocab.Item, id vocab.ID) {
	if vocab.LinkTypes.Contains(it.GetType()) {
		vocab.OnLink(it, func(lnk *vocab.Link) error {
			lnk.ID = id
			return nil
		})
	} else {
		vocab.OnObject(it, func(ob *vocab.Object) error {
			ob.ID = id
			return nil
		})
	}
}

func nameOf(it vocab.Item) string {
	var name string
	if vocab.LinkTypes.Contains(it.GetType()) {
		vocab.OnLink(it, func(lnk *vocab.Link) error {
			name = lnk.Name.First().String()
			return nil
		})
	} else {
		vocab.OnObject(it, func(ob *vocab.Object) error {
			name = ob.Name.First().String()
			return nil
		})
	}
	return name
}

func tagsContainsName(tags vocab.ItemCollection, ob vocab.Item) bool {
	name := nameOf(ob)

	if name == "" {
		return false
	}
	for _, tag := range tags {
		tagName := "#" + tagNormalize(nameOf(tag))
		if strings.EqualFold(tagName, name) {
			return true
		}
	}
	return false
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
		newActivityPubTag("metal", baseURL),
		newActivityPubTag("releases", baseURL),
	}
}

type apContent struct {
	tags     []string
	Date     time.Time
	Releases calendar.Events
	Tags     vocab.ItemCollection
}

func renderHTMLObject(d time.Time, rel calendar.Events, tags vocab.ItemCollection) (string, error) {
	model := apContent{Date: d, Releases: rel, Tags: tags}
	contBuff := bytes.NewBuffer(nil)
	if err := contHTMLTemplate.Execute(contBuff, model); err != nil {
		errFn("unable to render post %s", err)
		return "", err
	}
	return contBuff.String(), nil
}

func loadTagsToGroup(group calendar.Events, tags vocab.ItemCollection) (calendar.Events, vocab.ItemCollection) {
	remainingTags := make(vocab.ItemCollection, 0)
	for i, rel := range group {
		rel.Tags = make(vocab.ItemCollection, 0)
		for _, t := range tags {
			for _, tag := range rel.TagNames {
				tag = "#" + tagNormalize(tag)
				tagName := nameOf(t)
				if strings.EqualFold(tag, tagName) && !rel.Tags.Contains(t) {
					rel.Tags = append(rel.Tags, t)
				}
			}
		}
		group[i] = rel
	}
found:
	for _, t := range tags {
		for _, rel := range group {
			if rel.Tags.Contains(t) {
				continue found
			}
		}
		if !remainingTags.Contains(t) {
			remainingTags = append(remainingTags, t)
		}
	}
	return group, remainingTags
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
	nameSame := strings.EqualFold(nameOf(with), nameOf(toCheck))
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

func PostToActivityPub(cl *APClient) PosterFn {
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
		return PostToStdout
	}

	if err := acceptFollows(actor, ap); err != nil {
		errFn("failed to accept follows for actor: %s", err)
	}

	ctx := context.Background()

	return func(groupped map[time.Time]calendar.Events) error {
		activities := make([]vocab.Activity, 0)
		for gd, group := range groupped {
			ob := new(vocab.Object)
			ob.Type = vocab.NoteType

			var globalTags vocab.ItemCollection
			tags := append(defaultActivityPubTags(gd, actor.ID), apTags(group, actor.ID)...)
			toCreateTags, err := removeExistingTags(ctx, ap, actor, tags)
			if err != nil {
				infFn("Error when loading tags from server: %s", err)
			}
			if len(toCreateTags) > 0 {
				activities = append(activities, wrapObjectInCreate(*actor, toCreateTags))
			}

			group, globalTags = loadTagsToGroup(group, tags)

			content, err := renderHTMLObject(gd, group, globalTags)
			if err != nil {
				errFn("Unable to render HTML object: %s", err)
				continue
			}
			ob.Content = nl(content)
			ob.Tag = tags

			title, err := renderTitle(gd, group)
			if err == nil {
				ob.Name = nl(title)
			}
			if source, err := renderPosts(gd, group); err == nil {
				ob.Source = vocab.Source{
					MediaType: "text/markdown",
					Content:   nl(source),
				}
			}

			ob.To = vocab.ItemCollection{vocab.PublicNS}
			ob.CC = vocab.ItemCollection{vocab.Followers.Of(actor)}

			activities = append(activities, wrapObjectInCreate(*actor, ob))
		}
		(batch{ap: ap, activities: activities}).Send()

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

func wrapObjectInCreate(actor vocab.Actor, p vocab.Item) vocab.Activity {
	now := time.Now().UTC()
	return vocab.Activity{
		Type:         vocab.CreateType,
		Published:    now,
		Updated:      now,
		AttributedTo: actor.GetLink(),
		Actor:        actor.GetLink(),
		Object:       p,
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

func getActorOAuthEndpoint(actor vocab.Actor) oauth2.Endpoint {
	e := oauth2.Endpoint{
		AuthURL:  fmt.Sprintf("%s/oauth/authorize", actor.ID),
		TokenURL: fmt.Sprintf("%s/oauth/token", actor.ID),
	}
	if actor.Endpoints != nil {
		if !vocab.IsNil(actor.Endpoints.OauthAuthorizationEndpoint) {
			e.AuthURL = actor.Endpoints.OauthAuthorizationEndpoint.GetLink().String()
		}
		if !vocab.IsNil(actor.Endpoints.OauthTokenEndpoint) {
			e.TokenURL = actor.Endpoints.OauthTokenEndpoint.GetLink().String()
		}
	}
	return e
}

/*
func CheckONICredentialsFile(instance string, cl *http.Client, secret, token string, dryRun bool) (*APClient, error) {
	actorIRI := vocab.IRI(instance)
	u, _ := actorIRI.URL()
	key := u.Host

	client.UserAgent = filepath.Join(cmd.AppName, cmd.AppVersion, key)

	logger := lw.Dev()
	app := new(APClient)
	get := client.New(
		client.WithLogger(logger),
		client.WithHTTPClient(cl),
	)
	if token == "" && secret == "" {
		return nil, errors.Newf("neither a bearer token nor a password have been provided")
	}

	ctx := context.Background()
	actor, err := get.Actor(ctx, actorIRI)
	if err != nil {
		return nil, err
	}
	if vocab.IsNil(actor) || actor.ID == "" {
		return nil, errors.Newf("unable to load OAuth2 client application actor")
	}

	app.ID = actor.ID
	app.Conf = oauth2.Config{
		ClientID:     key,
		ClientSecret: secret,
		Endpoint:     getActorOAuthEndpoint(*actor),
		RedirectURL:  "http://localhost:3000",
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, cl)
	if token == "" {
		app.Tok, err = app.Conf.PasswordCredentialsToken(ctx, actor.ID.String(), app.Conf.ClientSecret)
		if err != nil {
			return nil, err
		}
	} else {
		tok := new(oauth2.Token)
		tok.AccessToken = token
		c := app.Conf.Client(ctx, tok)
		_, err = c.Get(app.ID.String())
		if err == nil {
			if tr, ok := c.Transport.(*oauth2.Transport); ok {
				app.Tok, err = tr.Source.Token()
				if err != nil {
					return nil, errors.Annotatef(err, "Unable to check received token")
				}
			}
		}
	}

	if app.Tok == nil {
		return nil, errors.Newf("Failed to load a valid OAuth2 token for client")
	}

	return app, saveCredentials(app, filepath.Join(cmd.DataPath(), InstanceName(instance)))
}

func CheckFedBOXCredentialsFile(instance, key, secret, token string, dryRun bool) (*APClient, error) {
	client.UserAgent = filepath.Join(cmd.AppName, cmd.AppVersion, key)

	logger := lw.Dev()
	app := new(APClient)
	get := client.New(
		client.WithLogger(logger),
		client.SkipTLSValidation(true),
		client.SetDefaultHTTPClient(),
	)
	ctx := context.Background()

	actorIRI := vocab.CollectionPath("actors").Of(vocab.IRI(instance)).GetLink().AddPath(key)
	actor, err := get.Actor(ctx, actorIRI)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, errors.Newf("unable to load OAuth2 client application actor")
	}

	app.ID = actor.ID
	app.Conf = oauth2.Config{
		ClientID:     key,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth/authorize", actor.ID),
			TokenURL: fmt.Sprintf("%s/oauth/token", actor.ID),
		},
		RedirectURL: "http://localhost:3000", // this should match what we used to register the client, the web interface URL
	}

	if token == "" {
		app.Tok, err = app.Conf.PasswordCredentialsToken(ctx, actor.ID.String(), app.Conf.ClientSecret)
		if err != nil {
			return nil, err
		}
	} else {
		tok := new(oauth2.Token)
		tok.AccessToken = token
		c := app.Conf.Client(ctx, tok)
		_, err = c.Get(app.ID.String())
		if err == nil {
			if tr, ok := c.Transport.(*oauth2.Transport); ok {
				app.Tok, err = tr.Source.Token()
				if err != nil {
					return nil, errors.Annotatef(err, "Unable to check received token")
				}
			}
		}
	}

	if app.Tok == nil {
		return nil, errors.Newf("Failed to load a valid OAuth2 token for client")
	}

	return app, saveCredentials(app, filepath.Join(cmd.DataPath(), InstanceName(instance)))
}
*/

func UpdateAPAccount(app *APClient, a fs.FS, dryRun bool) error {
	var name, desc, avatar, hdr string

	logger := lw.Dev()
	errFn = logger.Errorf
	infFn = logger.Infof

	var tags vocab.ItemCollection
	if data, _ := loadStaticFile(a, "static/name.txt"); data != nil {
		name = strings.TrimSpace(string(data))
	}
	if data, _ := loadStaticFile(a, "static/description.txt"); data != nil {
		tagextractor.URLGenerator = func(it vocab.Item) vocab.Item {
			name := nameOf(it)
			return app.ID.AddPath(strings.TrimPrefix(name, "#"))
		}
		data, tags = tagextractor.FindAndReplace(bytes.TrimSpace(data))

		md := markdown.New(
			markdown.HTML(true),
			markdown.Tables(true),
			markdown.Linkify(false),
			markdown.Typographer(true),
			markdown.Breaks(true),
		)

		cont := bytes.Buffer{}
		if err := md.Render(&cont, data); err == nil {
			desc = cont.String()
		}
	}
	if data, _ := loadStaticFile(a, "static/avatar.png"); data != nil {
		avatar = fmt.Sprintf("data:image/png;base64,%s", base64.RawStdEncoding.EncodeToString(data))
	}
	if data, _ := loadStaticFile(a, "static/header.png"); data != nil {
		hdr = fmt.Sprintf("data:image/png;base64,%s", base64.RawStdEncoding.EncodeToString(data))
	}
	if app.ID == "" {
		return errors.Newf("empty application id")
	}

	ap := client.New(
		client.WithHTTPClient(app.Conf.Client(context.Background(), app.Tok)),
		client.SetDefaultHTTPClient(),
		client.SkipTLSValidation(true),
		client.WithLogger(logger),
	)

	actor, err := ap.Actor(context.Background(), app.ID)
	if err != nil {
		return err
	}
	if len(name) > 0 {
		actor.Name = nl(name)
	}
	if len(desc) > 0 {
		actor.Content = nl(desc)
	}

	saveImage := func(iri vocab.IRI, data string) vocab.Activity {
		saveImage := vocab.Activity{
			Type:         vocab.UpdateType,
			Updated:      time.Now().UTC(),
			AttributedTo: actor.GetLink(),
			Actor:        actor.GetLink(),
		}

		image, _ := ap.Object(context.Background(), iri)
		if image == nil {
			image = &vocab.Object{}
			saveImage.Type = vocab.CreateType
		}
		image.ID = iri
		image.AttributedTo = actor.GetLink()
		image.Type = vocab.ImageType
		image.MediaType = "image/png"
		image.Content = nl(data)
		saveImage.Object = image
		return saveImage
	}

	operations := make([]vocab.Activity, 0)
	if len(avatar) > 0 {
		operations = append(operations, saveImage(actor.ID.AddPath("icon"), avatar))
		actor.Icon = actor.ID.AddPath("icon")
	}
	if len(hdr) > 0 {
		operations = append(operations, saveImage(actor.ID.AddPath("image"), hdr))
		actor.Icon = actor.ID.AddPath("image")
	}
	if tags.Count() > 0 {
		for _, t := range actor.Tag {
			if !actor.Tag.Contains(t) {
				actor.Tag = append(actor.Tag, t)
			}
			name := nameOf(t)
			setIDOf(t, actor.ID.AddPath(tagNormalize(name)))
		}
		operations = append(operations, wrapObjectInCreate(*actor, actor.Tag))
	}
	updateActor := vocab.Activity{
		Type:         vocab.UpdateType,
		Updated:      time.Now().UTC(),
		AttributedTo: actor.GetLink(),
		Actor:        actor.GetLink(),
		Object:       actor,
	}
	operations = append(operations, updateActor)
	if !dryRun {
		(batch{ap: ap, activities: operations}).Send()
	} else {
		infFn("Update activity: %+v", updateActor)
	}
	return nil
}

type batch struct {
	ap         client.PubSubmitter
	activities []vocab.Activity
}

func (b batch) Send() {
	for _, act := range b.activities {
		_, created, err := b.ap.ToOutbox(context.Background(), act)
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

func removeStrings(s string, replace ...string) string {
	for _, r := range replace {
		s = strings.ReplaceAll(s, r, "")
	}
	return s
}
