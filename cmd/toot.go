package cmd

import (
	"fmt"
	"github.com/McKael/madon"
	"github.com/urfave/cli"
	"os"
)

var Toot = cli.Command{
	Name:  "toot",
	Usage: "Post events to mastodon",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "calendar",
			Usage: "Which calendars to list",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Output debug messages",
		},
		&cli.StringFlag{
			Name:  "start",
			Usage: "Date at which to start",
			Value: defaultStartTime.Format("2006-01-02"),
		},
		&cli.DurationFlag{
			Name:  "end",
			Usage: "Date interval to check",
			Value: defaultDuration,
		},
	},
	Action: toot,
}

func errPrint(s string, p ...interface{}) {
	fmt.Fprintf(os.Stderr, s, p...)
}

func toot(c *cli.Context) error {
	var gClient *madon.Client
	var err error

	// Overwrite variables using Viper
	AppName := ""
	AppWebsite := ""
	scopes := make([]string, 0)
	instanceURL := ""
	appID := ""
	appSecret := ""
	verbose := true

	if instanceURL == "" {
		return fmt.Errorf("no instance provided")
	}

	if verbose {
		errPrint("Instance: '%s'", instanceURL)
	}

	if appID != "" && appSecret != "" {
		// We already have an app key/secret pair
		gClient, err = madon.RestoreApp(AppName, instanceURL, appID, appSecret, nil)
		if err != nil {
			return err
		}
		// Check instance
		if _, err := gClient.GetCurrentInstance(); err != nil {
			return fmt.Errorf("could not connect to server with provided app ID/secret: %w", err)
		}
		if verbose {
			errPrint("Using provided app ID/secret")
		}
		return nil
	}

	if appID != "" || appSecret != "" {
		errPrint("Warning: provided app id/secrets incomplete -- registering again")
	}

	gClient, err = madon.NewApp(AppName, AppWebsite, scopes, madon.NoRedirect, instanceURL)
	if err != nil {
		return fmt.Errorf("app registration failed: %w", err)
	}

	errPrint("Registered new application.")

	var st *madon.Status
	st, err = gClient.PostStatus("tootText", 0, nil, false, "", "public")
	if st.URL != "" {

	}

	return err
}
