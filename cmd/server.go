package cmd

import (
	"context"
	"fmt"
	"github.com/mariusor/esports-calendar/ical"
	"github.com/urfave/cli"
	"os"
	"syscall"
	"time"

	w "git.sr.ht/~mariusor/wrapper"
)

var Server = cli.Command{
	Name:  "start",
	Usage: "Starts the iCal serving server",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Output debug messages",
		},
		&cli.StringFlag{
			Name:  "host",
			Usage: "Set hostname on which to listen to",
			Value: "localhost",
		},
		&cli.IntFlag{
			Name:  "port",
			Usage: "Set hostname on which to listen to",
			Value: 9999,
		},
	},
	Action: serverStart,
}

var wait = 100 * time.Millisecond

var log = func(s string, args ...interface{}) {
	fmt.Printf(s+"\n", args...)
}

var errFn = func(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args...)
}

func serverStart(c *cli.Context) error {
	listen := fmt.Sprintf("%s:%d", c.String("host"), c.Int("port"))
	log("Listening on %s", listen)

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Get start/stop functions for the http server
	srvRun, srvStop := w.HttpServer(ctx, w.Handler(ical.Routes()), w.ListenOn(listen))
	w.RegisterSignalHandlers(w.SignalHandlers{
		syscall.SIGHUP: func(_ chan int) {
			log("SIGHUP received, reloading configuration")
		},
		syscall.SIGINT: func(exit chan int) {
			log("SIGINT received, stopping")
			exit <- 0
		},
		syscall.SIGTERM: func(exit chan int) {
			log("SIGITERM received, force stopping")
			exit <- 0
		},
		syscall.SIGQUIT: func(exit chan int) {
			log("SIGQUIT received, force stopping with core-dump")
			exit <- 0
		},
	}).Exec(func() error {
		if err := srvRun(); err != nil {
			errFn("Error: %s", err)
			return err
		}
		var err error
		// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
		go func(e error) {
			if err = srvStop(); err != nil {
				errFn("Error: %s", err)
			}
		}(err)
		return err
	})

	return nil
}

type LogFn func(string, ...interface{})
