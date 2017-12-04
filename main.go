package main

import (
	"log"
	"net/http"
	"os"

	"github.com/iced-mocha/reddit-client/config"
	"github.com/iced-mocha/reddit-client/handlers"
	"github.com/iced-mocha/reddit-client/server"
)

type Configuration struct {
	RedditSecret string `json:"reddit-secret"`
}

func checkExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

const (
	certFile = "server.crt"
	keyFile  = "server.key"
)

func main() {
	conf, err := config.New("config.yml")
	if err != nil {
		log.Fatalf("Unable to create config object: %v", err)
	}

	handler, err := handlers.New(conf)
	if err != nil {
		log.Fatalf("Unable to create handler: %v", err)
	}

	s, err := server.New(handler)
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}

	if !checkExists(certFile) || !checkExists(keyFile) {
		// Make the server available via http
		log.Fatalf("Could not detect both crt and key file quiting...")
	} else {
		// Cert and Key files found so use https
		log.Println("server.crt and server.key file detected starting server using https")
		http.ListenAndServeTLS(":3001", certFile, keyFile, s.Router)
	}
}
