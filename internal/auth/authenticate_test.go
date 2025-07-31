package auth

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashingAccuracy(t *testing.T) {
	type password struct {
		assigned string
		tried    string
	}
	passwords := []password{
		password{assigned: "iAmB4Tm@n", tried: "iAmB4Tm@n"},
		password{assigned: "iAmB4Tm@n", tried: "meaw"},
		password{assigned: "brotha, what", tried: ""},
	}
	for _, passwordPair := range passwords {
		hash, err := HashPassword(passwordPair.assigned)
		if err != nil {
			t.Fatal(err)
		}
		if PasswordMatchesHash(passwordPair.tried, hash) != (passwordPair.assigned == passwordPair.tried) {
			t.Errorf("hash didn't match.")
		}
	}

}

func TestJWT(t *testing.T) {
	type jwtCompilationData struct {
		userID uuid.UUID
		secret string
	}
	tokenData := []jwtCompilationData{}

	for i := 0; i < 10; i++ {
		user := uuid.New()
		secret := uuid.New().String()
		tokenData = append(tokenData, jwtCompilationData{userID: user, secret: secret})
	}
	jwtIdentifiers := []string{}
	for _, token := range tokenData {
		tokenValuation, err := MakeJWT(token.userID, token.secret, time.Minute*10)
		if err != nil {
			t.Fatal(err)
		}
		jwtIdentifiers = append(jwtIdentifiers, tokenValuation)
	}
	for i, identifier := range jwtIdentifiers {
		uuid, err := ValidateJWT(identifier, tokenData[i].secret)
		if err != nil {
			t.Fatal(err)
		}
		if uuid != tokenData[i].userID {
			t.Fatal("User ID didn't match.")
		}
	}
}

func TestGetBearer(t *testing.T) {
	type headerValues struct {
		key   string
		value string
		error error
	}

	tests := []headerValues{
		headerValues{key: "Content-Type", value: "Bearer deez_nuts", error: fmt.Errorf("(\"Bearer\", token) pair not found in http header.")},
		headerValues{key: "Bearer", value: "Authorisation mate", error: fmt.Errorf("(\"Bearer\", token) pair not found in http header.")},
		headerValues{key: "Authorization", value: "I am hungry", error: fmt.Errorf("(\"Bearer\", token) pair not found in http header.")},
		headerValues{key: "Authorization", value: "Barer 98765424ghvuedfgui", error: fmt.Errorf("Keyword \"Bearer\" not found.")},
		headerValues{key: "Authorization", value: "Bearer 2iurhi23ugr2bjk23roi3ap", error: nil},
	}

	for _, test := range tests {
		header := http.Header{}
		header.Set(test.key, test.value)
		value, err := GetBearerToken(header)
		if test.error != nil {
			if test.error.Error() != err.Error() {
				t.Errorf("Testing %s : %s.\n Error Message didn't match expectation. \n\tExp: %s\n\tGot %s", test.key, test.value, test.error, err)
			}
		} else {
			if strings.Fields(test.value)[1] != value {
				t.Errorf("Output didn't match. \n\tExp: %s.\n\tGot %s", strings.Fields(test.value)[1], value)
			}
		}
	}
}
