package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/anantashahane/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	//MARK:- API config, stores website hits.
	cfg := apiConfig{}

	//MARK:- Configuring Database.
	godotenv.Load()
	fmt.Println(os.Getenv("PLATFORM"))
	dbURL := os.Getenv("DB_URL")
	secret := os.Getenv("SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Error loading database " + dbURL)
		os.Exit(4)
	}
	cfg.db = database.New(db)
	cfg.platform = os.Getenv("PLATFORM")
	cfg.secret = secret
	cfg.polkaKey = os.Getenv("POKLA_KEY")

	serveMux := http.NewServeMux()
	server := http.Server{}

	server.Addr = ":8080"
	server.Handler = serveMux

	serveMux.HandleFunc("GET /admin/metrics/", apiHandler(cfg.metricsHandler, "/admin/"))
	serveMux.HandleFunc("POST /admin/reset", apiHandler(cfg.resetHandler, "/admin/"))
	serveMux.HandleFunc("GET /api/healthz/", apiHandler(healthHandler, "/api/"))

	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	serveMux.HandleFunc("POST /api/users", apiHandler(cfg.createUserHandler, "/api/"))
	serveMux.HandleFunc("PUT /api/users", apiHandler(cfg.handleUserPasswordChange, "/api/"))
	serveMux.HandleFunc("POST /api/login", apiHandler(cfg.loginUserHandler, "/api/"))
	serveMux.HandleFunc("POST /api/refresh", apiHandler(cfg.handleRefresh, "/api/"))
	serveMux.HandleFunc("POST /api/revoke", apiHandler(cfg.handleRevoke, "/api/"))

	serveMux.HandleFunc("POST /api/chirps", apiHandler(cfg.createChirpHandler, "/api/"))
	serveMux.HandleFunc("GET /api/chirps/", apiHandler(cfg.getAllChirpsHandler, "/api/"))
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiHandler(cfg.handleGetChirpByID, "/api/"))
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiHandler(cfg.deleteChirpHandler, "/api/"))

	serveMux.HandleFunc("POST /api/polka/webhooks", apiHandler(cfg.upgradeUserHandler, "/api/polka/webhooks"))

	fmt.Println("Listening on")
	fmt.Println("\tPOST admin/reset")
	fmt.Println()
	fmt.Println("\tGET /app")
	fmt.Println("\tGET api/healthz")
	fmt.Println("\tGET api/metrics")
	fmt.Println("\tPOST api/users")
	fmt.Println("\tPUT api/users")
	fmt.Println("\tPOST api/login")
	fmt.Println("\tGET api/chirps/[{chripID}]")
	fmt.Println("\tGET api/chirps")
	fmt.Println("\tDELETE api/chirps/{chirpID}")
	fmt.Println("\tPost api/refresh")
	fmt.Println("\tPost api/revoke")
	fmt.Println("\tPost api/polka/webhooks")

	err = server.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
