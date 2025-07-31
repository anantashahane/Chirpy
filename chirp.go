package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anantashahane/Chirpy/internal/auth"
	"github.com/anantashahane/Chirpy/internal/database"
	"github.com/google/uuid"
)

type chirpResponseBody struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	UserID    string `json:"user_id"`
}

func validateChirp(body string) bool {
	if len(body) > 140 {
		return false
	}
	return true
}

func (apiCfg *apiConfig) createChirpHandler(responseWriter http.ResponseWriter, req *http.Request) {
	type requestBody struct {
		Body string `json:"body"`
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

	tokenString, err := auth.GetBearerToken(req.Header)
	if err != nil {
		responseWriter.WriteHeader(406)
		responseWriter.Header().Set("Content Type", "plain/text")
		responseWriter.Write([]byte(fmt.Sprintf("Error reading authorisation from header. %s", err.Error())))
		return
	}

	uid, err := auth.ValidateJWT(tokenString, apiCfg.secret)
	if err != nil {
		responseWriter.WriteHeader(401)
		responseWriter.Header().Set("Content Type", "plain/text")
		responseWriter.Write(fmt.Appendf([]byte{}, "Error parsing user from JWT token. %s", err.Error()))
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
