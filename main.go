package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})	
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	responseStr := fmt.Sprintf("Hits: %d", hits)
	w.Write([]byte(responseStr))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func main() {
		mux := http.NewServeMux()	
		server := &http.Server{
			Addr: ":8080",
			Handler: mux,
		}
		apiConf := apiConfig{}
		mux.Handle("/app/", apiConf.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer((http.Dir("."))))))
		mux.HandleFunc("/healthz", readiness)
		mux.HandleFunc("/metrics", apiConf.handlerMetrics)
		mux.HandleFunc("/reset", apiConf.handlerReset)
		server.ListenAndServe()
}