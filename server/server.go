package server

import (
	"github.com/gorilla/mux"
	"github.com/iced-mocha/reddit-client/handlers"
)

type Server struct {
	Router *mux.Router
}

func New(api handlers.CoreAPI) (*Server, error) {
	s := &Server{Router: mux.NewRouter()}

	s.Router.HandleFunc("/v1/{id}/posts", api.GetPosts).Methods("GET")
	s.Router.HandleFunc("/v1/posts", api.GetPostsNoAuth).Methods("GET")
	s.Router.HandleFunc("/v1/authorize_callback", api.AuthorizeCallback).Methods("GET")
	s.Router.HandleFunc("/v1/{userID}/authorize", api.Authorize).Methods("GET")

	return s, nil
}
