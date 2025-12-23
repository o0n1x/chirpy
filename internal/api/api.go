package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/o0n1x/chirpy/internal/auth"
	"github.com/o0n1x/chirpy/internal/database"
)

type ApiConfig struct {
	FileserverHits atomic.Int32
	DB             *database.Queries
	Platform       string
	SECRET_JWT     string
	PolkaKey       string
}

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
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

	var chirps []database.Chirp

	author_id := r.URL.Query().Get("author_id")
	if author_id != "" {
		userUUID, err := uuid.Parse(author_id)
		if err != nil {
			log.Printf("Error invalid author ID: %v", err)
			respondWithError(w, 400, "invalid author ID")
			return
		}
		chirps, err = cfg.DB.GetChirpsByAuthor(r.Context(), uuid.NullUUID{UUID: userUUID, Valid: true})
		if err != nil {
			log.Printf("Error retrieving chirp: %v", err)
			respondWithError(w, 404, "chirp not found")
			return
		}

	} else {
		var err error
		chirps, err = cfg.DB.GetChirps(r.Context())
		if err != nil {
			log.Printf("Error retrieving chirps: %v", err)
			respondWithError(w, 500, "Failed to retrieve chirps")
			return
		}
	}

	sortby := r.URL.Query().Get("sort")

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
	if sortby == "desc" {
		sort.Slice(returningChirps, func(i, j int) bool {
			return returningChirps[i].CreatedAt.After(returningChirps[j].CreatedAt)
		})
	}
	respondWithJSON(w, 200, returningChirps)
}

func (cfg *ApiConfig) DeleteChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("id")
	if chirpID == "" {
		respondWithError(w, 400, "no ChirpID provided")
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error parsing header: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}
	userid, err := auth.ValidateJWT(token, cfg.SECRET_JWT)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}

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
	if chirp.UserID.UUID != userid {
		respondWithError(w, 403, "this user is not the author of the chirp")
		return
	}
	err = cfg.DB.DeleteChirp(context.Background(), chirp.ID)
	if err != nil {
		log.Printf("Error deleting chirp: %v", err)
		respondWithError(w, 404, "chirp not found")
		return
	}

	respondWithJSON(w, 204, nil)

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

func (cfg *ApiConfig) CreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error parsing header: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}
	userid, err := auth.ValidateJWT(token, cfg.SECRET_JWT)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
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
		UserID: uuid.NullUUID{UUID: userid, Valid: true},
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
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 400, "Invalid JSON in the request body")
		return
	}
	hashedpass, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(w, 500, "Failed to hash password")
		return
	}
	user, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: sql.NullString{String: hashedpass, Valid: true},
	})
	if err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(w, 500, "Failed to create user")
		return
	}

	respondWithJSON(w, 201, User{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	})

}

func (cfg *ApiConfig) UpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error parsing header: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}
	userid, err := auth.ValidateJWT(token, cfg.SECRET_JWT)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 400, "Invalid JSON in the request body")
		return
	}
	hashedpass, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(w, 500, "Failed to hash password")
		return
	}
	user, err := cfg.DB.UpdateUser(context.Background(), database.UpdateUserParams{
		ID:             userid,
		Email:          params.Email,
		HashedPassword: sql.NullString{String: hashedpass, Valid: true},
	})
	if err != nil {
		log.Printf("Error updating user: %v", err)
		respondWithError(w, 500, "Failed to update user")
		return
	}

	respondWithJSON(w, 200, User{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	})

}

func (cfg *ApiConfig) Login(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 400, "Invalid JSON in the request body")
		return
	}

	user, err := cfg.DB.GetUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("user not found: %v", err)
		respondWithError(w, 400, "Incorrect email or password")
		return
	}
	ok, err := auth.CheckPasswordHash(params.Password, user.HashedPassword.String)
	if !ok {
		log.Printf("password does not match: %v", err)
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	jwt_token, err := auth.MakeJWT(user.ID, cfg.SECRET_JWT, time.Hour)
	if err != nil {
		log.Printf("Error creating token: %v", err)
		respondWithError(w, 500, "Failed to create token")
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Error creating refresh token: %v", err)
		respondWithError(w, 500, "Failed to create refresh token")
		return
	}

	_, err = cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    uuid.NullUUID{UUID: user.ID, Valid: true},
		ExpiresAt: time.Now().Add(time.Hour * 1440), // 60 days
	})
	if err != nil {
		log.Printf("refresh token creation failed: %v", err)
		respondWithError(w, 500, "refresh token creation failed")
		return
	}

	respondWithJSON(w, 200, struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		IsChirpyRed  bool      `json:"is_chirpy_red"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		IsChirpyRed:  user.IsChirpyRed,
		Token:        jwt_token,
		RefreshToken: refreshToken,
	})

}

func (cfg *ApiConfig) Refresh(w http.ResponseWriter, r *http.Request) {
	tkn, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error parsing header: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}

	token, err := cfg.DB.GetRefreshToken(context.Background(), tkn)
	if err != nil {
		log.Printf("refresh token retreival failed: %v", err)
		respondWithError(w, 401, "refresh token invalid")
		return
	}
	if token.ExpiresAt.Before(time.Now()) {
		respondWithError(w, 401, "refresh token expired")
		return
	}
	if token.RevokedAt.Valid {
		respondWithError(w, 401, "refresh token revoked")
		return
	}
	if !token.UserID.Valid {
		log.Print("refresh token retreival failed: user id is null")
		respondWithError(w, 500, "token generation failed")
		return
	}
	jwt_token, err := auth.MakeJWT(token.UserID.UUID, cfg.SECRET_JWT, time.Hour)
	if err != nil {
		log.Printf("Error creating token: %v", err)
		respondWithError(w, 500, "Failed to create token")
		return
	}

	respondWithJSON(w, 200, struct {
		Token string `json:"token"`
	}{
		Token: jwt_token,
	})

}

func (cfg *ApiConfig) Revoke(w http.ResponseWriter, r *http.Request) {
	tkn, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error parsing header: %v", err)
		respondWithError(w, 401, "Token missing or invalid")
		return
	}
	err = cfg.DB.RevokeRefreshToken(context.Background(), tkn)
	if err != nil {
		log.Printf("revoking token failed: %v", err)
		respondWithError(w, 401, "refresh token invalid")
		return
	}
	respondWithJSON(w, 204, nil)
}

func (cfg *ApiConfig) PolkaWebhook(w http.ResponseWriter, r *http.Request) {
	apikey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		log.Printf("Error parsing header: %v", err)
		respondWithError(w, 401, "API key invalid")
		return
	}
	if apikey != cfg.PolkaKey {
		respondWithError(w, 401, "API key invalid")
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 400, "Invalid JSON in the request body")
		return
	}
	if params.Event != "user.upgraded" {
		log.Print("Error: invalid webhood event")
		respondWithJSON(w, 204, nil)
		return
	} else {
		err = cfg.DB.UpgradeUser(context.Background(), params.Data.UserID)
		if err != nil {
			log.Printf("upgrading user failed: %v", err)
			respondWithError(w, 404, "upgrading user failed")
			return
		}
		respondWithJSON(w, 204, nil)
	}

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
