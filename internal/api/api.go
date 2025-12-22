package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/o0n1x/chirpy/internal/database"
)

type ApiConfig struct {
	FileserverHits atomic.Int32
	DB             *database.Queries
	Platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

//
// HANDLERS
//

func Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *ApiConfig) GetChirps(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("id")
	if chirpID != "" {
		chirpUUID, err := uuid.Parse(chirpID)
		if err != nil {
			log.Printf("Error invalid chirp ID: %v", err)
			respondWithError(w, 400, "invalid ID")
			return
		}
		chirp, err := cfg.DB.GetChirp(r.Context(), chirpUUID)
		if err != nil {
			log.Printf("Error retrieving chirp: %v", err)
			respondWithError(w, 404, "chirp not found")
			return
		}

		respondWithJSON(w, 200, Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.UUID,
		})
		return
	}

	chirps, err := cfg.DB.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error retrieving chirps: %v", err)
		respondWithError(w, 500, "Failed to retrieve chirps")
		return
	}
	var returningChirps []Chirp
	for _, chirp := range chirps {
		returningChirps = append(returningChirps, Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.UUID,
		})
	}
	respondWithJSON(w, 200, returningChirps)
}

func (cfg *ApiConfig) Gethits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.FileserverHits.Load())))
}

func (cfg *ApiConfig) Resethits(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		w.WriteHeader(403)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	cfg.FileserverHits.Store(0)
	cfg.DB.DeleteUsers(r.Context())

}

// func validateChirp(w http.ResponseWriter, r *http.Request) {
// 	type parameters struct {
// 		Body string `json:"body"`
// 	}

// 	type returnVals struct {
// 		CleanedBody string `json:"cleaned_body"`
// 	}

// 	decoder := json.NewDecoder(r.Body)
// 	params := parameters{}
// 	err := decoder.Decode(&params)
// 	if err != nil {
// 		log.Printf("Error decoding parameters: %s", err)
// 		respondWithError(w, 400, "Invalid JSON in the request body")
// 		return
// 	}

// 	if len(params.Body) == 0 {
// 		respondWithError(w, 400, "Body is required")
// 		return
// 	}

// 	isvalid := len(params.Body) <= 140
// 	if !isvalid {
// 		respondWithError(w, 400, "Chirp is too long")
// 		return
// 	} else {

// 		cleanedbody := returnVals{
// 			CleanedBody: cleanifyString(params.Body),
// 		}

// 		respondWithJSON(w, 200, cleanedbody)
// 		return
// 	}

// }

func (cfg *ApiConfig) CreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 400, "Invalid JSON in the request body")
		return
	}
	if len(params.Body) == 0 {
		respondWithError(w, 400, "Body is required")
		return
	}

	isvalid := len(params.Body) <= 140
	if !isvalid {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	params.Body = cleanifyString(params.Body)
	chirp, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   params.Body,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
	})
	if err != nil {
		log.Printf("Error creating chirp: %v", err)
		respondWithError(w, 500, "Failed to create chirp")
		return
	}
	respondWithJSON(w, 201, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID.UUID,
	})

}

func (cfg *ApiConfig) CreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 400, "Invalid JSON in the request body")
		return
	}
	user, err := cfg.DB.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(w, 500, "Failed to create user")
		return
	}

	respondWithJSON(w, 201, User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})

}

//
// HELPER FUNCTIONS
//

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnErr struct {
		Error string `json:"error"`
	}

	respBody := returnErr{
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

func cleanifyString(s string) string {
	str := strings.Split(s, " ")
	badwords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}
	for i, word := range str {
		if badwords[strings.ToLower(word)] {
			str[i] = "****"
		}
	}
	return strings.Join(str, " ")
}
