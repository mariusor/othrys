package cmd

import (
	"fmt"
	"github.com/mariusor/esports-calendar/calendar/liquid"
	"github.com/mariusor/esports-calendar/calendar/plusforward"
	"github.com/urfave/cli"
	"strings"
)

var ShowTypes = cli.Command{
	Name:               "calendars",
	Usage:              "Lists supported calendar type, use --help to see a human readable list",
	Action:             showCalendars,
	CustomHelpTemplate: showHelp(),
}

func showHelp() string {
	h := strings.Builder{}
	h.WriteString("Valid calendar types:\n")
	h.WriteString("\n")
	h.WriteString(fmt.Sprintf("Global: %s", strings.Join(ValidTypes(), ", ")))
	h.WriteString(fmt.Sprintf("Global: %s, %s\n\tLoad all types on specific sites", liquid.LabelTeamLiquid, plusforward.LabelPlusForward))
	h.WriteString("\n")
	h.WriteString("Specific:\n")
	h.WriteString(fmt.Sprintf("\tTeamLiquid: %s\n", strings.Join(liquid.ValidTypes[:], ", ")))
	h.WriteString(fmt.Sprintf("\tPlusForward: %s\n", strings.Join(plusforward.ValidTypes[:], ", ")))
	return h.String()
}

func showCalendars(c *cli.Context) error {
	fmt.Printf("%s\n", strings.Join(ValidTypes(), ", "))
	return nil
}
