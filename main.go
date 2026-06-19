package main

import (
	"net/http"
	"log"
	"os"
	"database/sql"

	"github.com/gclinoz/Chirpy-server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	const port = "8080"

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error connecting to database: %s", err)
	}
	api := &apiConfig{
		db:			database.New(db),
		platform:	os.Getenv("PLATFORM"),
		key:		os.Getenv("APIKEY"),
	}

	fileServer := http.FileServer(http.Dir("."))
	fileServerInc := api.middlewareMetricsInc(fileServer)

	mux := http.NewServeMux()	
	mux.Handle("/app/", http.StripPrefix("/app", fileServerInc))
	mux.HandleFunc("GET /api/healthz", handleHealth)
	mux.HandleFunc("POST /admin/reset", api.handleReset)
	mux.HandleFunc("GET /admin/metrics", api.handleCountReq)
	mux.HandleFunc("POST /api/users", api.handleCreateUser)
	mux.HandleFunc("PUT /api/users", api.handleUpdateUser)
	mux.HandleFunc("POST /api/chirps", api.handleCreateChirp)
	mux.HandleFunc("GET /api/chirps", api.handleGetAllChirp)
	mux.HandleFunc("GET /api/chirps/{chirpID}", api.handleGetChirp)
	mux.HandleFunc("POST /api/login", api.handleLogin)
	mux.HandleFunc("POST /api/refresh", api.handleRefresh)
	mux.HandleFunc("POST /api/revoke", api.handleRevoke)

	server := &http.Server{
		Addr:		":" + port,
		Handler:	mux,
	}

	log.Printf("Serving files from project root on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
