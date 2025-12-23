package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/o0n1x/chirpy/internal/api"
	"github.com/o0n1x/chirpy/internal/database"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	secret := os.Getenv("SECRET_JWT")
	polkakey := os.Getenv("POLKA_KEY")
	filepathRoot := "/app/"
	port := "8080"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	cfg := api.ApiConfig{FileserverHits: atomic.Int32{}}
	cfg.DB = database.New(db)
	cfg.Platform = os.Getenv("PLATFORM")
	cfg.SECRET_JWT = secret
	cfg.PolkaKey = polkakey

	mux := http.NewServeMux()
	mux.Handle(filepathRoot, http.StripPrefix("/app/", cfg.MiddlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", api.Healthz)
	mux.HandleFunc("GET /admin/metrics", cfg.Gethits)
	mux.HandleFunc("POST /admin/reset", cfg.Resethits)
	mux.HandleFunc("POST /api/chirps", cfg.CreateChirp)
	mux.HandleFunc("GET /api/chirps", cfg.GetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", cfg.GetChirps)
	mux.HandleFunc("DELETE /api/chirps/{id}", cfg.DeleteChirp)
	mux.HandleFunc("POST /api/users", cfg.CreateUser)
	mux.HandleFunc("PUT /api/users", cfg.UpdateUser)
	mux.HandleFunc("POST /api/login", cfg.Login)
	mux.HandleFunc("POST /api/refresh", cfg.Refresh)
	mux.HandleFunc("POST /api/revoke", cfg.Revoke)
	mux.HandleFunc("POST /api/polka/webhooks", cfg.PolkaWebhook)

	s := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(s.ListenAndServe())
}
