package main

import (
	"github.com/iced-mocha/reddit/handlers"
	"github.com/iced-mocha/reddit/server"
	"log"
	"net/http"
)

type Configuration struct {
	RedditSecret string `json:"reddit-secret"`
}

func main() {
	handler := &handlers.CoreHandler{}
	s, err := server.New(handler)
	if err != nil {
		log.Fatal("error initializing server: ", err)
	}

	log.Fatal(http.ListenAndServe(":3001", s.Router))
}
