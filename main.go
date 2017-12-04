package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
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

	// Make any one who needs to make requests to use has their cert here
	caCert, err := ioutil.ReadFile("/etc/ssl/certs/core.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cfg := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
	}
	srv := &http.Server{
		Addr:      ":3001",
		Handler:   s.Router,
		TLSConfig: cfg,
	}
	log.Fatal(srv.ListenAndServeTLS("/etc/ssl/certs/reddit.crt", "/etc/ssl/private/reddit.key"))
}
