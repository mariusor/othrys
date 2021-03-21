package cmd

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"

	"github.com/mariusor/esports-calendar/calendar"
)

var ShowTypes = cli.Command{
	Name:               "calendars",
	Usage:              "Lists supported calendar type, use --help to see a human readable list",
	Action:             showCalendars,
	CustomHelpTemplate: showHelp(),
}

var validTypes = calendar.GetTypes(nil)

func showHelp() string {
	h := strings.Builder{}
	h.WriteString("Valid calendar types:\n")
	h.WriteString("\n")
	h.WriteString(fmt.Sprintf("Global: %s", strings.Join(validTypes, ", ")))
	h.WriteString(fmt.Sprintf("Global: %s, %s\n\tLoad all types on specific sites", calendar.DefaultCalendars[0],calendar.DefaultCalendars[0]))
	h.WriteString("\n")
	h.WriteString("Specific:\n")
	//h.WriteString(fmt.Sprintf("\tTeamLiquid: %s\n", strings.Join(liquid.ValidTypes[:], ", ")))
	//h.WriteString(fmt.Sprintf("\tPlusForward: %s\n", strings.Join(plusforward.ValidTypes[:], ", ")))
	return h.String()
}

func showCalendars(c *cli.Context) error {
	fmt.Printf("%s\n", strings.Join(validTypes, ", "))
	return nil
}
