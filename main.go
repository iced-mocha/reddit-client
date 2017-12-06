package main

import (
	"crypto/tls"
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

	srv := &http.Server{
		Addr:      ":3001",
		Handler:   s.Router,
		TLSConfig: &tls.Config{},
	}
	log.Fatal(srv.ListenAndServeTLS("/usr/local/etc/ssl/certs/reddit.crt", "/usr/local/etc/ssl/private/reddit.key"))
}
