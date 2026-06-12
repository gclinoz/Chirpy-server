package main

import (
	"net/http"
	"log"
)

func main() {
	const port = "8080"

	fileServer := http.FileServer(http.Dir("."))
	mux := http.NewServeMux()	

	mux.HandleFunc("/healthz", handleHealth)
	mux.Handle("/app", http.StripPrefix("/app", fileServer))

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
