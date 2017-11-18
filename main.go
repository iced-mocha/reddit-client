package main

import (
	"log"
	"net/http"

	"github.com/iced-mocha/reddit-client/config"
	"github.com/iced-mocha/reddit-client/handlers"
	"github.com/iced-mocha/reddit-client/server"
)

type Configuration struct {
	RedditSecret string `json:"reddit-secret"`
}

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

	log.Fatal(http.ListenAndServe(":3001", s.Router))
}
