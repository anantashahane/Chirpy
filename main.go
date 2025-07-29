package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/anantashahane/Chirpy/internal/auth"
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

type chirpResponseBody struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	UserID    string `json:"user_id"`
}

type userDataRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userDataResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Email     string `json:"email"`
	Error     string `json:"error,omitempty"`
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

func validateChirp(body string) bool {
	if len(body) > 140 {
		return false
	}
	return true
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
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	encoder := json.NewEncoder(responseWriter)

	requestedData := userDataRequest{}
	err := decoder.Decode(&requestedData)
	if err != nil {
		responseWriter.WriteHeader(420)
		responseData := userDataResponse{Error: "Incoming json format too zooted."}
		encoder.Encode(responseData)
		return
	}

	hash, err := auth.HashPassword(requestedData.Password)
	if err != nil {
		responseWriter.WriteHeader(420)
		responseData := userDataResponse{Error: "Password failed to hash. Error: " + err.Error()}
		encoder.Encode(responseData)
		return
	}

	creationData, err := apiCfg.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     requestedData.Email,
		Password:  hash,
	})
	if err != nil {
		responseWriter.WriteHeader(406)
		responseData := userDataResponse{Error: fmt.Sprintf("Account %v seems to already be registered.", requestedData.Email)}
		encoder.Encode(responseData)
		return
	}
	responseData := userDataResponse{
		ID:        creationData.ID.String(),
		CreatedAt: creationData.CreatedAt.String(),
		UpdatedAt: creationData.UpdatedAt.String(),
		Email:     creationData.Email,
	}
	responseWriter.WriteHeader(201)
	err = encoder.Encode(responseData)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseData := userDataResponse{Error: "Internal Server error"}
		encoder.Encode(responseData)
		return
	}
}

func (apiCfg *apiConfig) loginUserHandler(responseWriter http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	encoder := json.NewEncoder(responseWriter)

	requestedData := userDataRequest{}
	err := decoder.Decode(&requestedData)
	if err != nil {
		responseWriter.WriteHeader(420)
		responseData := userDataResponse{Error: "Incoming json format too zooted."}
		encoder.Encode(responseData)
		return
	}

	userData, err := apiCfg.db.GetUser(context.Background(), requestedData.Email)
	if err != nil {
		responseWriter.WriteHeader(401)
		responseData := userDataResponse{Error: "No such user, " + requestedData.Email}
		encoder.Encode(responseData)
		return
	}

	if match := auth.PasswordMatchesHash(requestedData.Password, userData.Password); !match {
		responseWriter.WriteHeader(401)
		responseData := userDataResponse{Error: "Incorrect password for user " + requestedData.Email}
		encoder.Encode(responseData)
		return
	}

	responseData := userDataResponse{
		ID:        userData.ID.String(),
		CreatedAt: userData.CreatedAt.String(),
		UpdatedAt: userData.UpdatedAt.String(),
		Email:     userData.Email,
	}
	responseWriter.WriteHeader(200)
	err = encoder.Encode(responseData)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseData := userDataResponse{Error: "Internal Server error"}
		encoder.Encode(responseData)
		return
	}
}

func (apiCfg *apiConfig) createChirpHandler(responseWriter http.ResponseWriter, req *http.Request) {
	type requestBody struct {
		Body   string `json:"body"`
		UserID string `json:"user_id"`
	}

	encoder := json.NewEncoder(responseWriter)
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()

	requestData := requestBody{}
	err := decoder.Decode(&requestData)
	if err != nil {
		responseWriter.WriteHeader(406)
		responseWriter.Header().Set("Content Type", "plain/text")
		responseWriter.Write([]byte("Json Decode failed."))
		return
	}

	if !validateChirp(requestData.Body) {
		responseWriter.WriteHeader(406)
		responseWriter.Header().Set("Content Type", "plain/text")
		responseWriter.Write([]byte("Chirp too long."))
		return
	}

	uid, err := uuid.Parse(requestData.UserID)
	if err != nil {
		responseWriter.WriteHeader(406)
		responseWriter.Header().Set("Content Type", "plain/text")
		responseWriter.Write([]byte("User ID not valid."))
		return
	}

	savedData, err := apiCfg.db.CreateChirps(context.Background(), database.CreateChirpsParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Body:      requestData.Body,
		UserID:    uid,
	})
	if err != nil {
		responseWriter.WriteHeader(422)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Save failed, check if attached user ID exists."))
		return
	}

	responseData := chirpResponseBody{
		ID:        savedData.ID.String(),
		CreatedAt: savedData.CreatedAt.String(),
		UpdatedAt: savedData.UpdatedAt.String(),
		Body:      savedData.Body,
		UserID:    savedData.UserID.String(),
	}

	responseWriter.WriteHeader(201)
	responseWriter.Header().Set("Content-Type", "application/json")
	err = encoder.Encode(responseData)

	if err != nil {
		responseWriter.WriteHeader(406)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("JSON encode failed."))
		return
	}
}

func (apiCfg *apiConfig) getAllChirpsHandler(responseWriter http.ResponseWriter, req *http.Request) {

	encoder := json.NewEncoder(responseWriter)

	responseBody := []chirpResponseBody{}

	chirps, err := apiCfg.db.GetChirps(context.Background())
	if err != nil {
		responseWriter.WriteHeader(500)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Internal Server failed to access database."))
		return
	}

	for _, chirp := range chirps {
		responseBody = append(responseBody, chirpResponseBody{
			ID:        chirp.ID.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			CreatedAt: chirp.CreatedAt.String(),
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		})
	}

	responseWriter.WriteHeader(200)
	responseWriter.Header().Set("Content-Type", "application/json")
	err = encoder.Encode(responseBody)
	if err != nil {
		responseWriter.WriteHeader(500)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Unable to generate JSON response."))
		return
	}
}

func (apiCfg *apiConfig) handleGetChirpByID(responseWriter http.ResponseWriter, req *http.Request) {
	path := req.PathValue("chirpID")
	id, err := uuid.Parse(path)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Error parsing ID " + path))
		return
	}
	responseBody := chirpResponseBody{}

	dbChirp, err := apiCfg.db.GetChirpByID(context.Background(), id)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("No such chirp with ID " + path))
		return
	}
	responseBody.ID = dbChirp.ID.String()
	responseBody.CreatedAt = dbChirp.CreatedAt.String()
	responseBody.UpdatedAt = dbChirp.UpdatedAt.String()
	responseBody.Body = dbChirp.Body
	responseBody.UserID = dbChirp.UserID.String()

	encoder := json.NewEncoder(responseWriter)
	responseWriter.WriteHeader(200)
	responseWriter.Header().Set("Content-Type", "application/json")

	err = encoder.Encode(responseBody)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Error encoding json data."))
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
	serveMux.HandleFunc("POST /api/login", apiHandler(cfg.loginUserHandler, "/api/"))
	serveMux.HandleFunc("GET /api/healthz/", apiHandler(healthHandler, "/api/"))
	serveMux.HandleFunc("POST /api/chirps", apiHandler(cfg.createChirpHandler, "/api/"))
	serveMux.HandleFunc("GET /api/chirps/", apiHandler(cfg.getAllChirpsHandler, "/api/"))
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiHandler(cfg.handleGetChirpByID, "/api/"))

	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	fmt.Println("Listening on")
	fmt.Println("\tPOST admin/reset")
	fmt.Println()
	fmt.Println("\tGET /app")
	fmt.Println("\tGET api/healthz")
	fmt.Println("\tGET api/metrics")
	fmt.Println("\tPOST api/users")
	fmt.Println("\tPOST api/login")
	fmt.Println("\tPOST api/chirps/")
	fmt.Println("\tGET api/chirps")

	err = server.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
