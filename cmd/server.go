package cmd

import (
	"context"
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
			Value: 8080,
		},
	},
	Action: serverStart,
}

var wait = 100 * time.Millisecond

var log = func(s string, args ...interface{}) {
	fmt.Printf(s+"\n", args...)
}

var err = func(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args...)
}

func serverStart(c *cli.Context) error {
	listen := fmt.Sprintf("%s:%d", c.String("host"), c.Int("port"))
	log("Listening on %s", listen)

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Get start/stop functions for the http server
	srvRun, srvStop := setupHttpServer(listen, nil, wait, ctx)
	go srvRun(err)

	// Add signal handlers
	sigChan := make(chan os.Signal, 1)
	exitChan := make(chan int)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go waitForSignal(sigChan, exitChan)(log)
	code := <-exitChan

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
	go srvStop(err)
	log("Shutting down")

	if code != 0 {
		return fmt.Errorf("received exit code %d", code)
	}
	return nil
}

type LogFn func(string, ...interface{})

func setupHttpServer(listen string, m http.Handler, wait time.Duration, ctx context.Context) (func(LogFn), func(LogFn)) {
	// TODO(marius): move server run to a separate function,
	//   so we can add other tasks that can run independently.
	//   Like a queue system for lazy loading of IRIs.
	srv := &http.Server{
		Addr:         listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      m,
	}

	run := func(l LogFn) {
		if err := srv.ListenAndServe(); err != nil {
			if l != nil {
				l("%s", err)
			}
			os.Exit(1)
		}
	}

	stop := func(l LogFn) {
		err := srv.Shutdown(ctx)
		if err != nil && l != nil {
			l("%s", err)
		}
		select {
		case <-ctx.Done():
			if l != nil {
				l("%s", ctx.Err())
			}
		}
	}
	// Run our server in a goroutine so that it doesn't block.
	return run, stop
}

func waitForSignal(sigChan chan os.Signal, exitChan chan int) func(LogFn) {
	return func(l LogFn) {
		for {
			s := <-sigChan
			switch s {
			case syscall.SIGHUP:
				l("SIGHUP received, reloading configuration")
			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				l("SIGINT received, stopping")
				exitChan <- 0
			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				l("SIGITERM received, force stopping")
				exitChan <- 0
			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				l("SIGQUIT received, force stopping with core-dump")
				exitChan <- 0
			default:
				l("Unknown signal %d", s)
			}
		}
	}
}
