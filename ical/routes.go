package ical

import (
	"net/http"
)

func Routes() http.Handler {
	r := http.NewServeMux()
	r.Handle("/", NewHandler())
	return r
}
