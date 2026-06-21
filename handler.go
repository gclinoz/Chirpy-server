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
	"github.com/gclinoz/Chirpy-server/internal/auth"
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
	key				string
}

type User struct {
	ID			uuid.UUID	`json:"id"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
	Email		string		`json:"email"`
	Token		string		`json:"token"`
	RefToken	string		`json:"refresh_token"`
	IsRed		bool		`json:"is_chirpy_red"`
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
		Password string `json:"password"`
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

	hashed, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}
	paramUser := database.CreateUserParams{
		Email:			params.Email,
		HashedPassword:	hashed,
	}
	data, err := cfg.db.CreateUser(r.Context(), paramUser)
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
		IsRed:		data.IsChirpyRed,
	}
	respondWithJSON(w, 201, resp)
}

func (cfg *apiConfig) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	ts, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, error when getting token")
		return
	}

	uid, err := auth.ValidateJWT(ts, cfg.key)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, invalid token")
		return
	}

	type parameters struct {
		Password			string	`json:"password"`
		Email				string	`json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	hashed, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, "Error when hashing password")
		return
	}

	updateParam := database.UpdateUserParams{
		ID:				uid,
		Email:			params.Email,
		HashedPassword:	hashed,
	}
	err = cfg.db.UpdateUser(r.Context(), updateParam)
	if err != nil {
		respondWithError(w, 500, "Error when updating user info")
		return
	}

	resp := User{
		ID:			uid,
		Email:		params.Email,
	}
	respondWithJSON(w, 200, resp)
}

func (cfg *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password			string	`json:"password"`
		Email				string	`json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	user, err := cfg.db.GetUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}
	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	ts, err := auth.MakeJWT(user.ID, cfg.key, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Error when generating token")
		return
	}

	refParams := database.CreateRefTkParams{
		Token:	auth.MakeRefreshToken(),
		UserID:	user.ID,
	}
	reftk, err := cfg.db.CreateRefTk(r.Context(), refParams)
	if err != nil {
		respondWithError(w, 500, "Error when generating refresh token")
		return
	}

	resp := User{
		ID:			user.ID,
		CreatedAt:	user.CreatedAt,
		UpdatedAt:	user.UpdatedAt,
		Email:		user.Email,
		Token:		ts,
		RefToken:	reftk.Token,
		IsRed:		user.IsChirpyRed,
	}
	respondWithJSON(w, 200, resp)
}

func (cfg *apiConfig) handleRefresh(w http.ResponseWriter, r *http.Request) {
	reftk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, no valid refresh token found")
		return
	}

	refData, err := cfg.db.GetUserFromRefreshToken(r.Context(), reftk)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, error getting refresh token info")
		return
	}
	if time.Now().After(refData.ExpiresAt) || refData.RevokedAt.Valid == true {
		respondWithError(w, 401, "Unauthorized, token expired or revoked")
		return
	}

	ts, err := auth.MakeJWT(refData.UserID, cfg.key, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Error when generating token")
		return
	}

	type parameters struct {
		Token string `json:"token"`
	}
	resp := parameters{Token: ts}
	respondWithJSON(w, 200, resp)
}

func (cfg *apiConfig) handleRevoke(w http.ResponseWriter, r *http.Request) {
	reftk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, no valid refresh token found")
		return
	}

	upParams := database.UpdateRevokeAtParams{
		Token:		reftk,
		UpdatedAt:	time.Now(),
	}
	err = cfg.db.UpdateRevokeAt(r.Context(), upParams)
	if err != nil {
		respondWithError(w, 500, "Error when updating refresh token")
		return
	}
	w.WriteHeader(204)
}

func (cfg *apiConfig) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	ts, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, invalid token")
		return
	}

	uid, err := auth.ValidateJWT(ts, cfg.key)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, invalid token")
		return
	}

	type parameters struct {
		Body	string		`json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Error when decoding request")
		return
	}

	paramChirp := database.CreateChirpParams{
		Body:	params.Body,
		UserID:	uid,
	}
	data, err := cfg.db.CreateChirp(r.Context(), paramChirp)
	if err != nil {
		respondWithError(w, 500, "Error when creating chirps")
		return
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

func (cfg *apiConfig) handleGetAllChirp(w http.ResponseWriter, r *http.Request) {
	data, err := cfg.db.GetAllChirp(r.Context())
	if err != nil {
		respondWithError(w, 500, "Error when getting chirps")
		return
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

func (cfg *apiConfig) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	parsed, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 500, "Invalid chirpID")
		return
	}

	data, err := cfg.db.GetChirp(r.Context(), parsed)
	if err != nil {
		respondWithError(w, 404, "Error when getting the chirp")
		return
	}

	resp := Chirp{
		ID:			data.ID,
		CreatedAt:	data.CreatedAt,
		UpdatedAt:	data.UpdatedAt,
		Body:		data.Body,
		UserID:		data.UserID,
	}
	respondWithJSON(w, 200, resp)
}

func (cfg *apiConfig) handleDelChirp(w http.ResponseWriter, r *http.Request) {
	ts, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, error when getting token")
		return
	}
	uid, err := auth.ValidateJWT(ts, cfg.key)
	if err != nil {
		respondWithError(w, 401, "Unauthorized, invalid token")
		return
	}

	parsed, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 500, "Invalid chirpID")
		return
	}
	data, err := cfg.db.GetChirp(r.Context(), parsed)
	if err != nil {
		respondWithError(w, 404, "Error when getting the chirp")
		return
	}
	
	if uid != data.UserID {
		respondWithError(w, 403, "You are not authorized to do this")
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), data.ID)
	if err != nil {
		respondWithError(w, 500, "Error when deleting chirp")
		return
	}
	w.WriteHeader(204)
}

func (cfg *apiConfig) handleWebHook(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event	string	`json:"event"`
		Data	struct	{
			UserID	string	`json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Error when decoding request")
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	parsed, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, 500, "Invalid user ID")
		return
	}
	if params.Event == "user.upgraded" {
		err = cfg.db.UpdateRed(r.Context(), parsed)
		if err != nil {
			respondWithError(w, 404, "Error when updating red")
			return
		}
		w.WriteHeader(204)
	}
}

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
