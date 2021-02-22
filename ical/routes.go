package ical

import (
	"net/http"
)

func Routes(path string) http.Handler {
	r := http.NewServeMux()
	r.Handle("/", NewHandler(path))
	return r
}
