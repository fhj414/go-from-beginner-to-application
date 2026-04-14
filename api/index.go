package handler

import (
	"log"
	"net/http"
	"sync"

	"github.com/fhj/go-from-beginner-to-application/pkg/gopherquestapp"
)

var (
	initOnce sync.Once
	app      http.Handler
	initErr  error
)

func getHandler() (http.Handler, error) {
	initOnce.Do(func() {
		srv, err := gopherquestapp.NewServerFromEnv()
		if err != nil {
			initErr = err
			return
		}
		app = srv.Handler()
	})
	return app, initErr
}

// Handler is the Vercel Go Function entrypoint.
func Handler(w http.ResponseWriter, r *http.Request) {
	h, err := getHandler()
	if err != nil {
		log.Printf("vercel handler init: %v", err)
		http.Error(w, "server init failed", http.StatusInternalServerError)
		return
	}
	h.ServeHTTP(w, r)
}
