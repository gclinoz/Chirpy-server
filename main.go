package main

import (
	"net/http"
	"log"
)

func main() {
	const port = "8080"

	fileServer := http.FileServer(http.Dir("."))
	mux := http.NewServeMux()	
	mux.Handle("/", fileServer)

	server := &http.Server{
		Addr:		":" + port,
		Handler:	mux,
	}

	log.Printf("Serving files from project root on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
