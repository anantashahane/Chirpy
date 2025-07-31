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
	IsRed        bool   `json:"is_chirpy_red"`
	Error        string `json:"error,omitempty"`
}

// Dryer Code
func userErrorWriter(responseWriter *http.ResponseWriter, encoder *json.Encoder, errorStr string, httpCode int) {
	(*responseWriter).WriteHeader(httpCode)
	responseData := userDataResponse{Error: errorStr}
	encoder.Encode(responseData)
}

func (apiCfg *apiConfig) createUserHandler(responseWriter http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	encoder := json.NewEncoder(responseWriter)

	requestedData := userDataRequest{}
	err := decoder.Decode(&requestedData)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Incoming json format too zooted.", 420)
		return
	}

	hash, err := auth.HashPassword(requestedData.Password)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Password failed to hash. Error: "+err.Error(), 401)
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
		userErrorWriter(&responseWriter, encoder, fmt.Sprintf("Account %v seems to already be registered.", requestedData.Email), 406)
		return
	}
	responseData := userDataResponse{
		ID:        creationData.ID.String(),
		CreatedAt: creationData.CreatedAt.String(),
		UpdatedAt: creationData.UpdatedAt.String(),
		Email:     creationData.Email,
		IsRed:     creationData.IsChirpyRed.Bool,
	}
	responseWriter.WriteHeader(201)
	err = encoder.Encode(responseData)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Json Encode error.", 404)
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
		userErrorWriter(&responseWriter, encoder, "Incoming json format too zooted.", 420)
		return
	}

	userData, err := apiCfg.db.GetUser(context.Background(), requestedData.Email)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "No such user, "+requestedData.Email, 401)
		return
	}

	if match := auth.PasswordMatchesHash(requestedData.Password, userData.Password); !match {
		userErrorWriter(&responseWriter, encoder, "Incorrect password for user "+requestedData.Email, 401)
		return
	}

	token, err := auth.MakeJWT(userData.ID, apiCfg.secret, time.Hour)
	if match := auth.PasswordMatchesHash(requestedData.Password, userData.Password); !match {
		userErrorWriter(&responseWriter, encoder, "Authentication error user from token", 403)
		return
	}

	refreshToken, err := auth.MakeRefreshedToken()
	if match := auth.PasswordMatchesHash(requestedData.Password, userData.Password); !match {
		userErrorWriter(&responseWriter, encoder, "Error generating refresh tokens", 503)
		return
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
		IsRed:        userData.IsChirpyRed.Bool,
	}
	responseWriter.WriteHeader(200)
	err = encoder.Encode(responseData)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Internal Server error, json encoder", 503)
		return
	}
}

func (apiCfg *apiConfig) handleRefresh(responseWriter http.ResponseWriter, req *http.Request) {
	type responseBody struct {
		Token string `json:"token"`
	}

	encoder := json.NewEncoder(responseWriter)

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Error parsing Token "+err.Error(), 404)
		return
	}

	tokenData, err := apiCfg.db.GetRefreshToken(context.Background(), token)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "No such token signed. Please sign in."+err.Error(), 401)
		return
	}
	if tokenData.ExpiresAt.Compare(time.Now()) == -1 {
		userErrorWriter(&responseWriter, encoder, "Sign in timed out. Please sign in.", 401)
		return
	}

	if tokenData.RevokedAt.Valid {
		userErrorWriter(&responseWriter, encoder, "Token revoked at "+tokenData.RevokedAt.Time.String()+". Please sign in.", 401)
		return
	}

	newToken, err := auth.MakeJWT(tokenData.UserID, apiCfg.secret, time.Hour)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Error generating token.", 503)
		return
	}

	tokenJsonData := responseBody{Token: newToken}
	responseWriter.WriteHeader(200)
	err = encoder.Encode(tokenJsonData)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Failed encoding json data.", 503)
		return
	}
}

func (apiCfg *apiConfig) handleRevoke(responseWriter http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	encoder := json.NewEncoder(responseWriter)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Error parsing Token "+err.Error(), 404)
		return
	}

	_, err = apiCfg.db.RevokeToken(context.Background(), database.RevokeTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Tokens:    token,
	})
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Error revoking refresh token.", 404)
		return
	}
	responseWriter.WriteHeader(204)
}

func (apiCfg *apiConfig) handleUserPasswordChange(responseWriter http.ResponseWriter, req *http.Request) {

	// Pull out user from the header data.
	requestedData := userDataRequest{}
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	defer req.Body.Close()

	encoder := json.NewEncoder(responseWriter)

	authString, err := auth.GetBearerToken(req.Header)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Token extraction error: "+err.Error(), 401)
		return
	}

	uid, err := auth.ValidateJWT(authString, apiCfg.secret)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Token decode error: "+err.Error(), 401)
		return
	}

	userData, err := apiCfg.db.GetUserByID(context.Background(), uid)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Unable to find signed user in database: "+err.Error(), 401)
		return
	}

	// Check validity of email ID of the user.
	err = decoder.Decode(&requestedData)
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Error decoding json: "+err.Error(), 401)
		return
	}

	passHash, err := auth.HashPassword(requestedData.Password)

	updatedUserData, err := apiCfg.db.UpdatePassword(context.Background(), database.UpdatePasswordParams{
		Password:  passHash,
		UpdatedAt: time.Now(),
		ID:        userData.ID,
		Email:     requestedData.Email,
	})
	if err != nil {
		userErrorWriter(&responseWriter, encoder, "Error setting password, "+err.Error(), 401)
		return
	}

	responseWriter.WriteHeader(200)
	responseData := userDataResponse{
		ID:           updatedUserData.ID.String(),
		CreatedAt:    updatedUserData.CreatedAt.String(),
		UpdatedAt:    updatedUserData.UpdatedAt.String(),
		Email:        updatedUserData.Email,
		Token:        authString,
		RefreshToken: updatedUserData.Tokens.String,
		IsRed:        updatedUserData.IsChirpyRed.Bool,
	}
	encoder.Encode(responseData)
}

func (apiCfg *apiConfig) upgradeUserHandler(responseWriter http.ResponseWriter, req *http.Request) {
	polkaKey, err := auth.GetPolkaKey(req.Header)

	type UserUpgradeJson struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	if err != nil || polkaKey != apiCfg.polkaKey {
		responseWriter.WriteHeader(401)
		return
	}

	userUpgradeData := UserUpgradeJson{}

	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()

	err = decoder.Decode(&userUpgradeData)
	if err != nil {
		fmt.Println("Unable to decode data.")
		responseWriter.WriteHeader(404)
		return
	}

	if userUpgradeData.Event == "user.upgraded" {
		uid, err := uuid.Parse(userUpgradeData.Data.UserID)
		if err != nil {
			fmt.Println("Unable to parse data.")
			responseWriter.WriteHeader(404)
			return
		}
		userInDB, err := apiCfg.db.GetUserByID(context.Background(), uid)
		if err != nil {
			fmt.Println("User not in DB.")
			responseWriter.WriteHeader(404)
			return
		}
		_, err = apiCfg.db.UpgradeUsertoRed(context.Background(), userInDB.ID)
		if err != nil {
			fmt.Println("Unable to upgrade user in DB.")
			responseWriter.WriteHeader(404)
			return
		}
	} else {
		fmt.Println("Unknown Event " + userUpgradeData.Event)
		responseWriter.WriteHeader(204)
		return
	}
	responseWriter.WriteHeader(204)
}
