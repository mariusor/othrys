package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"git.sr.ht/~mariusor/tagextractor"
	othrys "github.com/mariusor/esports-calendar"
	"github.com/mariusor/esports-calendar/internal/post"
	"gitlab.com/golang-commonmark/markdown"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	"golang.org/x/oauth2"
)

func CheckONICredentialsFile(instance string, cl *http.Client, secret, token string, dryRun bool) (*post.APClient, error) {
	actorIRI := vocab.IRI(instance)
	u, _ := actorIRI.URL()
	key := u.Host

	client.UserAgent = filepath.Join(AppName, AppVersion, key)

	logger := lw.Dev()
	app := new(post.APClient)
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

	return app, saveCredentials(app, filepath.Join(DataPath(), InstanceName(instance)))
}

func CheckFedBOXCredentialsFile(instance, key, secret, token string, dryRun bool) (*post.APClient, error) {
	client.UserAgent = filepath.Join(AppName, AppVersion, key)

	logger := lw.Dev()
	app := new(post.APClient)
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

	return app, saveCredentials(app, filepath.Join(DataPath(), InstanceName(instance)))
}

func UpdateAPAccount(app *post.APClient, a fs.FS, dryRun bool) error {
	var name, desc, avatar, hdr string

	logger := lw.Dev()

	var tags vocab.ItemCollection
	if data, _ := loadStaticFile(a, "static/name.txt"); data != nil {
		name = strings.TrimSpace(string(data))
	}
	if data, _ := loadStaticFile(a, "static/description.txt"); data != nil {
		tagextractor.URLGenerator = func(it vocab.Item) vocab.Item {
			name := othrys.NameOf(it)
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
		actor.Name = othrys.NL(name)
	}
	if len(desc) > 0 {
		actor.Content = othrys.NL(desc)
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
		image.Content = othrys.NL(data)
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
			name := othrys.NameOf(t)
			othrys.SetIDOf(t, actor.ID.AddPath(othrys.TagNormalize(name)))
		}
		operations = append(operations, othrys.WrapObjectInCreate(*actor, actor.Tag))
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
		(post.OperationsBatch{AP: ap, Ops: operations}).Send()
	} else {
		info("Update activity: %+v", updateActor)
	}
	return nil
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