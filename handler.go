package main

import (
	"log"
	"fmt"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gclinoz/Chirpy-server/internal/database"
)

func handleHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

type apiConfig struct {
	fileserverHits	atomic.Int32
	db				*database.Queries
	platform		string
}

type User struct {
	ID			uuid.UUID	`json:"id"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
	Email		string		`json:"email"`
}

type Chirp struct {
	ID        uuid.UUID	`json:"id"`
	CreatedAt time.Time	`json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string	`json:"body"`
	UserID    uuid.UUID	`json:"user_id"`
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
	if cfg.platform != "dev" {
		respondWithError(w, 403, "You are not authorized to do this")
		return
	}

	cfg.fileserverHits.Store(0)
	err := cfg.db.DeleteAllUser(r.Context())
	if err != nil {
		log.Printf("Error deleting users: %s", err)
		w.WriteHeader(500)
		return
	}
	fmt.Fprintf(w, "Hits reset to 0 and Delete all users")
}

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	data, err := cfg.db.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("Error when creating new user: %s", err)
		w.WriteHeader(500)
		return
	}

	resp := User{
		ID:			data.ID,
		CreatedAt:	data.CreatedAt,
		UpdatedAt:	data.UpdatedAt,
		Email:		data.Email,
	}
	respondWithJSON(w, 201, resp)
}

func (cfg *apiConfig) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body	string		`json:"body"`
		User_id	uuid.UUID	`json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Invalid chirp")
		return
	}

	paramChirp := database.CreateChirpParams{
		Body:	params.Body,
		UserID:	params.User_id,
	}
	data, err := cfg.db.CreateChirp(r.Context(), paramChirp)
	if err != nil {
		respondWithError(w, 500, "Error when creating chirps")
	}

	resp := Chirp{
		ID:			data.ID,
		CreatedAt:	data.CreatedAt,
		UpdatedAt:	data.UpdatedAt,
		Body:		data.Body,
		UserID:		data.UserID,
	}
	respondWithJSON(w, 201, resp)
}

func (cfg *apiConfig) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	data, err := cfg.db.GetAllChirp(r.Context())
	if err != nil {
		respondWithError(w, 500, "Error when getting chirps")
	}
	
	resp := []Chirp{}
	for _, val := range data {
		resp = append(resp, Chirp{
				ID:			val.ID,
				CreatedAt:	val.CreatedAt,
				UpdatedAt:	val.UpdatedAt,
				Body:		val.Body,
				UserID:		val.UserID,
			},
		)
	}
	respondWithJSON(w, 200, resp)
}
// func handleValid(w http.ResponseWriter, r *http.Request) {
// 	type parameters struct {
// 		Body string `json:"body"`
// 	}
// 	decoder := json.NewDecoder(r.Body)
// 	params := parameters{}
// 	err := decoder.Decode(&params)
// 	if err != nil {
// 		log.Printf("Error decoding parameters: %s", err)
// 		w.WriteHeader(500)
// 		return
// 	}
// 	if len(params.Body) > 140 {
// 		respondWithError(w, 400, "Chirp is too long")
// 		return
// 	}
// 	type validResponse struct {
// 		Cleaned_body string `json:"cleaned_body"`
// 	}
// 	success := validResponse{
// 		Cleaned_body: replaceBad(params.Body),
// 	}
// 	respondWithJSON(w, 200, success)
// }

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnVals struct {
		Error string `json:"error"`
	}

	respBody := returnVals{
		Error: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

// func replaceBad(words string) string {
// 	splited := strings.Split(words, " ")
// 	replaced := []string{}
// 	for _, val := range splited {
// 		if strings.ToLower(val) == "kerfuffle" ||
// 		strings.ToLower(val) == "sharbert" ||
// 		strings.ToLower(val) == "fornax" {
// 			replaced = append(replaced, "****")
// 			continue
// 		}
// 		replaced = append(replaced, val)
// 	}
// 	return strings.Join(replaced, " ")
// }
