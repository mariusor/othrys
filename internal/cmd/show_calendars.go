package cmd

import (
	"fmt"
	"github.com/mariusor/esports-calendar/calendar/liquid"
	"github.com/mariusor/esports-calendar/calendar/plusforward"
	"io"
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

var validTypes = calendar.DefaultCalendars

func writeHelpLabels(w io.StringWriter, labels ...string) error {
	for _, lbl := range labels {
		w.WriteString("\t\t")
		w.WriteString(lbl)
		w.WriteString(": ")
		w.WriteString(calendar.Labels[lbl])
		w.WriteString("\n")
	}
	return nil
}
func showHelp() string {
	h := strings.Builder{}
	h.WriteString("Valid calendar Types:\n")
	h.WriteString("Global:\n")
	writeHelpLabels(&h, validTypes...)
	h.WriteString("\n")
	h.WriteString("Specific:\n")
	h.WriteString("\t")
	h.WriteString(calendar.Labels["tl"])
	h.WriteString(":\n")
	writeHelpLabels(&h, liquid.ValidTypes[:]...)
	h.WriteString("\n")
	h.WriteString("\t")
	h.WriteString(calendar.Labels["pfw"])
	h.WriteString(":\n")
	writeHelpLabels(&h, plusforward.ValidTypes[:]...)
	return h.String()
}

func showCalendars(c *cli.Context) error {
	fmt.Printf("%s\n", strings.Join(calendar.GetTypes(nil), ", "))
	return nil
}
