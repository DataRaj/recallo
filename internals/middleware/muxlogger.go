package middleware

import (
	"log"
	"net/http"
)

func Loggingmiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s from %s", r.Method, r.URL.Path, r.Proto, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
