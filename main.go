package main

import (
	"net/http"
	"log"
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
	mux.HandleFunc("POST /api/validate_chirp", handleValid)

	server := &http.Server{
		Addr:		":" + port,
		Handler:	mux,
	}

	log.Printf("Serving files from project root on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
