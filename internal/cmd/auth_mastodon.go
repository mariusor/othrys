package cmd

import (
	"encoding/gob"
	"fmt"
	"github.com/McKael/madon"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const ExecOpenCmd = "xdg-open"

func CheckMastodonCredentialsFile(path, key, secret, token, instance string, dryRun bool, getAccessTokenFn func() (string, error)) (*madon.Client, error) {
	app := new(madon.Client)
	path = filepath.Join(path, instance)
	if err := loadMastodonCredentials(app, path); err != nil {
		if len(key) > 0 && len(secret) > 0 {
			if len(key) > 0 {
				app.ID = key
			}
			if len(secret) > 0 {
				app.Secret = secret
			}
			app.Name = AppName
			app.InstanceURL = "https://" + instance
			app.APIBase = app.InstanceURL + "/api/v1"
			app.UserToken = new(madon.UserToken)
			if len(token) > 0 {
				app.UserToken.AccessToken = token
			}
		} else {
			if app, err = madon.NewApp(AppName, AppWebsite, AppScopes, "", instance); err != nil {
				return nil, fmt.Errorf("unable to initialize mastodon application: %w", err)
			}
		}
	}
	if ValidMastodonAuth(app) {
		return app, saveMastodonCredentials(app, filepath.Join(DataPath(), InstanceName(app.InstanceURL)))
	}
	if !dryRun {
		userAuthUri, err := app.LoginOAuth2("", nil)
		if err != nil {
			return nil, fmt.Errorf("unable to login to %s: %w", app.InstanceURL, err)
		}
		if err = exec.Command(ExecOpenCmd, userAuthUri).Run(); err != nil {
			//infFn("unable to use %s to open %s: %s", ExecOpenCmd, app.InstanceURL, err)
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
			//infFn("unable to save credentials: %s", err)
		}
	}
	return app, nil
}

func InstanceName(inst string) string {
	u, err := url.ParseRequestURI(inst)
	if err != nil {
		inst = u.Host
	}
	return url.PathEscape(filepath.Clean(filepath.Base(inst)))
}
func ValidMastodonAuth(c *madon.Client) bool {
	return ValidMastodonApp(c) && c.UserToken != nil && c.UserToken.AccessToken != ""
}

func ValidMastodonApp(c *madon.Client) bool {
	return c != nil && c.Name != "" && c.ID != "" && c.Secret != "" && c.APIBase != "" && c.InstanceURL != ""
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

func saveMastodonCredentials(c *madon.Client, path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open file %w", err)
	}
	defer f.Close()

	d := gob.NewEncoder(f)
	return d.Encode(c)
}
