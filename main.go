package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/anantashahane/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func apiHandler(next http.HandlerFunc, prefix string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := http.StripPrefix(prefix, next)
		api.ServeHTTP(w, r)
	})
}

func healthHandler(responseWriter http.ResponseWriter, req *http.Request) {
	responseWriter.WriteHeader(200)
	responseWriter.Header().Add("Content-Type", "text/plain; charset=utf-8")
	_, err := responseWriter.Write([]byte("OK"))
	if err != nil {
		os.Exit(2)
	}
}

func sensorCompetition(chirp string) (sensored_chirp string) {
	sensored_chirp = chirp
	competitiors := []string{"kerfuffle", "sharbert", "fornax"}
	found := []string{}
	for _, competitor := range competitiors {
		for word := range strings.FieldsSeq(chirp) {
			if competitor == strings.ToLower(word) {
				found = append(found, word)
			}
		}
	}
	fmt.Println(found)
	for _, word := range found {
		sensored_chirp = strings.ReplaceAll(sensored_chirp, word, "****")
	}
	return
}

func validateChirpHandler(responseWriter http.ResponseWriter, req *http.Request) {
	type incomingPayload struct {
		Body string `json:"body"`
	}

	type outGoingPayload struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(req.Body)
	encoder := json.NewEncoder(responseWriter)

	incomingPayloadData := incomingPayload{}
	err := decoder.Decode(&incomingPayloadData)

	responseWriter.Header().Set("Content-Type", "application/json")
	if err != nil {
		responseWriter.WriteHeader(400)
		return
	}

	if len(incomingPayloadData.Body) > 140 {
		responseWriter.WriteHeader(400)
		return
	}
	censored_chirp := sensorCompetition(incomingPayloadData.Body)

	outGoingPayloadData := outGoingPayload{CleanedBody: censored_chirp}
	responseWriter.WriteHeader(200)
	encoder.Encode(&outGoingPayloadData)
}

func (apiCfg *apiConfig) metricsHandler(responseWriter http.ResponseWriter, req *http.Request) {
	responseWriter.WriteHeader(200)
	responseWriter.Header().Add("Content-Type", "text/html")
	text := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
		    	<p>Chirpy has been visited %d times!</p>
		    </body>
		</html>
		`, apiCfg.fileServerHits.Load())
	_, err := responseWriter.Write([]byte(text))
	if err != nil {
		os.Exit(3)
	}
}

func (apiCfg *apiConfig) resetHandler(responseWriter http.ResponseWriter, req *http.Request) {
	if apiCfg.platform != "dev" {
		responseWriter.WriteHeader(403)
	} else {
		responseWriter.WriteHeader(200)
		apiCfg.fileServerHits.Store(0)
		_, err := apiCfg.db.DeleteAllUsers(context.Background())
		if err != nil {
			fmt.Println("Failed to reset users table.")
		}
	}
}

func (apiCfg *apiConfig) createUserHandler(responseWriter http.ResponseWriter, req *http.Request) {
	type requestBody struct {
		Email string `json:"email"`
	}

	type responseBody struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Email     string `json:"email"`
		Error     string `json:"error,omitempty"`
	}

	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	encoder := json.NewEncoder(responseWriter)

	requestedData := requestBody{}
	err := decoder.Decode(&requestedData)
	if err != nil {
		responseWriter.WriteHeader(420)
		responseData := responseBody{Error: "Incoming json format too zooted."}
		encoder.Encode(responseData)
		return
	}

	creationData, err := apiCfg.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     requestedData.Email,
	})
	if err != nil {
		responseWriter.WriteHeader(406)
		responseData := responseBody{Error: fmt.Sprintf("Account %v seems to already be registered.", requestedData.Email)}
		encoder.Encode(responseData)
		return
	}
	responseData := responseBody{
		ID:        creationData.ID.String(),
		CreatedAt: creationData.CreatedAt.String(),
		UpdatedAt: creationData.UpdatedAt.String(),
		Email:     creationData.Email,
	}
	responseWriter.WriteHeader(201)
	err = encoder.Encode(responseData)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseData := responseBody{Error: "Internal Server error"}
		encoder.Encode(responseData)
		return
	}
}

func main() {
	//MARK:- API config, stores website hits.
	cfg := apiConfig{}

	//MARK:- Configuring Database.
	godotenv.Load()
	fmt.Println(os.Getenv("PLATFORM"))
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Error loading database " + dbURL)
		os.Exit(4)
	}
	cfg.db = database.New(db)
	cfg.platform = os.Getenv("PLATFORM")

	serveMux := http.NewServeMux()
	server := http.Server{}

	server.Addr = ":8080"
	server.Handler = serveMux

	serveMux.HandleFunc("GET /admin/metrics/", apiHandler(cfg.metricsHandler, "/admin/"))
	serveMux.HandleFunc("POST /admin/reset", apiHandler(cfg.resetHandler, "/admin/"))

	serveMux.HandleFunc("POST /api/users", apiHandler(cfg.createUserHandler, "/api/"))
	serveMux.HandleFunc("GET /api/healthz/", apiHandler(healthHandler, "/api/"))
	serveMux.HandleFunc("POST /api/validate_chirp", apiHandler(validateChirpHandler, "/api/"))

	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	fmt.Println("Listening on")
	fmt.Println("\tPOST admin/reset\n")
	fmt.Println("\tGET /app")
	fmt.Println("\tGET api/healthz")
	fmt.Println("\tGET api/metrics")
	fmt.Println("\tPOST api/validate_chirp/")
	fmt.Println("\tPOST api/users")

	err = server.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
