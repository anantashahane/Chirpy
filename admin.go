package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

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

func healthHandler(responseWriter http.ResponseWriter, req *http.Request) {
	responseWriter.WriteHeader(200)
	responseWriter.Header().Add("Content-Type", "text/plain; charset=utf-8")
	_, err := responseWriter.Write([]byte("OK"))
	if err != nil {
		os.Exit(2)
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
