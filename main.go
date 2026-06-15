package main

import (
	"fmt"
	"net/http"
	"log"
	"sync/atomic"
)

func main() {
	const port = "8080"

	api := &apiConfig{}
	fileServer := http.FileServer(http.Dir("."))
	fileServerInc := api.middlewareMetricsInc(fileServer)

	mux := http.NewServeMux()	
	mux.Handle("/app/", http.StripPrefix("/app", fileServerInc))
	mux.HandleFunc("GET /api/healthz", handleHealth)
	mux.HandleFunc("POST /admin/reset", api.handleReset)
	mux.HandleFunc("GET /admin/metrics", api.handleCountReq)

	server := &http.Server{
		Addr:		":" + port,
		Handler:	mux,
	}

	log.Printf("Serving files from project root on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}

func handleHealth (w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

type apiConfig struct {
	fileserverHits	atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handleCountReq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
}

func (cfg *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	fmt.Fprintf(w, "Hits reset to 0")
}
