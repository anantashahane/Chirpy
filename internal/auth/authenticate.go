package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func PasswordMatchesHash(password, hash string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return false
	}
	return true
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	})
	tokenIdentifier, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return tokenIdentifier, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claimer := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claimer, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})

	// Check if parsing failed BEFORE accessing token.Claims
	if err != nil {
		return uuid.UUID{}, err
	}

	// Now it's safe to access token.Claims
	idStr, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	key := headers.Get("Authorization")
	btPair := strings.Fields(key)
	if len(btPair) != 2 {
		return "", fmt.Errorf("(\"Bearer\", token) pair not found in http header.")
	}
	if btPair[0] != "Bearer" {
		return "", fmt.Errorf("Keyword \"Bearer\" not found.")
	}
	return btPair[1], nil
}

func GetPolkaKey(headers http.Header) (string, error) {
	key := headers.Get("Authorization")
	btPair := strings.Fields(key)
	if len(btPair) != 2 {
		return "", fmt.Errorf("(\"ApiKey\", token) pair not found in http header.")
	}
	if btPair[0] != "ApiKey" {
		return "", fmt.Errorf("Keyword \"ApiKey\" not found.")
	}
	return btPair[1], nil
}

func MakeRefreshedToken() (string, error) {
	token := make([]byte, 32)
	rand.Read(token)
	return hex.EncodeToString(token), nil
}
