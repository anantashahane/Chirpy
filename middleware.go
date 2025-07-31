package main

import (
	"net/http"
	"sync/atomic"

	"github.com/anantashahane/Chirpy/internal/database"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func apiHandler(next http.HandlerFunc, prefix string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := http.StripPrefix(prefix, next)
		api.ServeHTTP(w, r)
	})
}
