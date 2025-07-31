package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anantashahane/Chirpy/internal/auth"
	"github.com/anantashahane/Chirpy/internal/database"
	"github.com/google/uuid"
)

type userDataRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userDataResponse struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error,omitempty"`
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

	token, err := auth.MakeJWT(userData.ID, apiCfg.secret, time.Hour)
	if err != nil {
		fmt.Printf("Error generating token: %s\n", err.Error())
	}

	refreshToken, err := auth.MakeRefreshedToken()
	if err != nil {
		fmt.Printf("Error generating refresh token: %s\n", err.Error())
	}

	apiCfg.db.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{
		Tokens:    refreshToken,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    userData.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
		RevokedAt: sql.NullTime{},
	})

	responseData := userDataResponse{
		ID:           userData.ID.String(),
		CreatedAt:    userData.CreatedAt.String(),
		UpdatedAt:    userData.UpdatedAt.String(),
		Email:        userData.Email,
		Token:        token,
		RefreshToken: refreshToken,
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

func (apiCfg *apiConfig) handleRefresh(responseWriter http.ResponseWriter, req *http.Request) {
	type responseBody struct {
		Token string `json:"token"`
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Error parsing Token " + err.Error()))
		return
	}

	tokenData, err := apiCfg.db.GetRefreshToken(context.Background(), token)
	if err != nil {
		responseWriter.WriteHeader(401)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("No such token signed. Please sign in." + err.Error()))
		return
	}
	if tokenData.ExpiresAt.Compare(time.Now()) == -1 {
		responseWriter.WriteHeader(401)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Sign in timed out. Please sign in."))
		return
	}

	if tokenData.RevokedAt.Valid {
		responseWriter.WriteHeader(401)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Token revoked at " + tokenData.RevokedAt.Time.String() + ". Please sign in."))
		return
	}

	newToken, err := auth.MakeJWT(tokenData.UserID, apiCfg.secret, time.Hour)
	if err != nil {
		responseWriter.WriteHeader(503)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Error generating token."))
		return
	}

	tokenJsonData := responseBody{Token: newToken}
	encoder := json.NewEncoder(responseWriter)

	responseWriter.WriteHeader(200)
	err = encoder.Encode(tokenJsonData)
	if err != nil {
		responseWriter.WriteHeader(503)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Failed encoding json data."))
		return
	}
}

func (apiCfg *apiConfig) handleRevoke(responseWriter http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		responseWriter.WriteHeader(404)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Error parsing Token " + err.Error()))
		return
	}

	_, err = apiCfg.db.RevokeToken(context.Background(), database.RevokeTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Tokens:    token,
	})
	if err != nil {
		responseWriter.WriteHeader(404)
		responseWriter.Header().Set("Content-Type", "plain/text")
		responseWriter.Write([]byte("Error revoking refresh token."))
		return
	}
	responseWriter.WriteHeader(204)
}
