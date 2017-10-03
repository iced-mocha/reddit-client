package main

import (
	"github.com/icedmocha/reddit/handlers"
	"github.com/icedmocha/reddit/server"
	"log"
	"net/http"
)

func main() {

	handler := &handlers.CoreHandler{}
	s, err := server.New(handler)
	if err != nil {
		log.Fatal("error initializing server: ", err)
	}

	log.Fatal(http.ListenAndServe(":3000", s.Router))
}
