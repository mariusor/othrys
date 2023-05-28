package cmd

import (
	"fmt"
	"io/fs"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli"

	"git.sr.ht/~mariusor/othrys"
	"git.sr.ht/~mariusor/othrys/calendar"
	"git.sr.ht/~mariusor/othrys/internal/post"
)

var AuthorizeCmd = cli.Command{
	Name:    "auth",
	Aliases: []string{"authorize"},
	Usage:   "Authorizes the application against a Fediverse instance",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "key",
			Usage: "Client application key",
		},
		&cli.StringFlag{
			Name:  "secret",
			Usage: "Client application secret",
		},
		&cli.StringFlag{
			Name:  "token",
			Usage: "Personal access token",
		},
		&cli.StringFlag{
			Name:  "instance",
			Usage: "The instance to authenticate against",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: "The type of the instance: Mastodon, FedBOX, oni",
			Value: "mastodon",
		},
		&cli.StringSliceFlag{
			Name:  "calendar",
			Usage: "The calendars that the application will serve",
			Value: (*cli.StringSlice)(&calendar.DefaultCalendars),
		},
		&cli.BoolFlag{
			Name:  "update-account",
			Usage: "Update the details of the mastodon account",
		},
	},
	Action: Authorize,
}

func Authorize(c *cli.Context) error {
	key := c.String("key")
	secret := c.String("secret")
	accessToken := c.String("token")
	instance := c.String("instance")
	typ := c.String("type")
	dryRun := c.GlobalBool("dry-run")
	update := c.Bool("update-account")

	calendars := stringSliceValues(c, "calendar")

	s, err := fs.Sub(othrys.AccountDetails, "static")
	if err != nil {
		errFn("Unable to find folder with calendars description and names.")
	} else {
		if len(calendars) == 1 {
			s, err = fs.Sub(s, calendars[0])
			if err != nil {
				errFn("Unable to find folder with calendars description and names.")
			}
		}
	}
	calendars = calendar.GetTypes(calendars)

	switch typ {
	case TypeMastodon:
		getTok := getAccessToken("Paste authorization code: ")
		client, err := CheckMastodonCredentialsFile(DataPath(), key, secret, accessToken, instance, calendars, dryRun, getTok)
		if err != nil {
			return err
		}
		if update && s != nil {
			return UpdateMastodonAccount(client, s, dryRun)
		}
		info("Success, authorized client for: %s", client.InstanceURL)
	case TypeONI:
		cl := post.GetHTTPClient()
		client, err := CheckONICredentialsFile(instance, cl, secret, accessToken, calendars, dryRun)
		if err != nil {
			return err
		}
		if update && s != nil {
			return UpdateAPAccount(client, s, calendars, dryRun)
		}
		info("Success, authorized client for: %s", client.Conf.ClientID)
	case TypeFedBOX:
		client, err := CheckFedBOXCredentialsFile(instance, key, secret, accessToken, calendars, dryRun)
		if err != nil {
			return err
		}
		if update && s != nil {
			// Update the ActivityPub Actor with the Avatar/Image/Text
			return UpdateAPAccount(client, s, calendars, dryRun)
		}
		info("Success, authorized client for: %s", client.Conf.ClientID)
	default:
		return fmt.Errorf("invalid instance type %s", typ)
	}
	return nil
}

type model struct {
	prompt    string
	textInput *textinput.Model
	err       error
}

func initialModel(prompt string) model {
	ti := textinput.New()
	ti.Placeholder = "..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 45
	ti.EchoMode = textinput.EchoPassword

	return model{
		prompt:    prompt,
		textInput: &ti,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type errMsg error

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	*m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.prompt,
		m.textInput.View(),
	) + "\n"
}

func getAccessToken(prompt string) func() (string, error) {
	return func() (string, error) {
		m := initialModel(prompt)
		err := tea.NewProgram(m).Start()
		return m.textInput.Value(), err
	}
}
