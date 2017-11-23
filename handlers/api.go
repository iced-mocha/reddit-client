package handlers

import (
	"net/http"
)

type CoreAPI interface {
	GetPosts(w http.ResponseWriter, r *http.Request)
	GetPostsNoAuth(w http.ResponseWriter, r *http.Request)
	Authorize(w http.ResponseWriter, r *http.Request)
	AuthorizeCallback(w http.ResponseWriter, r *http.Request)
}
