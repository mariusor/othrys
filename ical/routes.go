package ical

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"net/http"
)

func Routes() http.Handler {
	r := chi.NewRouter()

	i := ical{}

	r.Use(middleware.GetHead)
	r.Use(middleware.Logger)

	// http://calendar/starcraft/2015/
	r.Get("/{type}/{year}", i.ServeHTTP)
	// http://calendar/starcraft/
	r.Get("/{type}", i.ServeHTTP)

	r.Handle("/favicon.ico", nil)
	r.NotFound(nil)
	r.MethodNotAllowed(nil)

	return r
}
