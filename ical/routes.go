package ical

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"net/http"
)

func Routes() http.Handler {
	r := chi.NewRouter()

	c := cal{}

	r.Use(middleware.GetHead)
	r.Use(middleware.Logger)

	// http://calendar/starcraft/2015/
	r.Get("/{year}/{type}", c.ServeHTTP)
	// http://calendar/starcraft/
	r.Get("/{type}", c.ServeHTTP)

	r.Handle("/favicon.ico", nil)
	r.NotFound(nil)
	r.MethodNotAllowed(nil)

	return r
}
