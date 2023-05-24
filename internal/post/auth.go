package post

import "github.com/urfave/cli"

type LoginCredentials interface {
	Valid(c *cli.Context) bool
	Post() PosterFn
}
