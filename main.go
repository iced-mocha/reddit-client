package main

import (
	"github.com/gorilla/mux"
	"github.com/icedmocha/reddit/handlers"
	"github.com/icedmocha/reddit/server"
	"log"
	"net/http"
)

type MyServer struct {
	r *mux.Router
}

func (s *MyServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
		rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		rw.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}
	// Stop here if its Preflighted OPTIONS request
	if req.Method == "OPTIONS" {
		return
	}
	// Lets Gorilla work
	s.r.ServeHTTP(rw, req)
}

func main() {

	handler := &handlers.CoreHandler{}
	s, err := server.New(handler)
	if err != nil {
		log.Fatal("error initializing server: ", err)
	}

	/*	log.Fatal(http.ListenAndServe(":3000", ghandlers.CORS(
		ghandlers.AllowedMethods([]string{"GET", "POST"}),
		ghandlers.AllowedOrigins([]string{"*"}),
		ghandlers.AllowedHeaders([]string{"X-Requested-With"}))(s.Router)))*/
	//log.Fatal(http.ListenAndServe(":3000", s.Router))
	http.Handle("/", &MyServer{s.Router})
	http.ListenAndServe(":3000", nil)
}
