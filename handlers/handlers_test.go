package handlers

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/reddit-client/config"
	"github.com/stretchr/testify/suite"
)

type HandlersTestSuite struct {
	suite.Suite
	router  *mux.Router
	handler CoreHandler
}

func MockGetRedditIdentity(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{ "name": "test" }`))
}

func (suite *HandlersTestSuite) SetupSuite() {
	// Disable logging while testing
	log.SetOutput(ioutil.Discard)

	suite.handler = CoreHandler{client: &http.Client{}}

	// In order to test using path params we need to run a server and send requests to it
	suite.router = mux.NewRouter()
	suite.router.HandleFunc("/api/v1/me", MockGetRedditIdentity).Methods(http.MethodGet)
	//suite.router.HandleFunc("/v1/users", suite.handler.InsertUser).Methods(http.MethodPost)

	// Spin up our testing server
	s := httptest.NewServer(suite.router)

	suite.handler.conf = &config.Config{RedditOAuthURL: s.URL, RedditSecret: "secret", RedditClientID: "clientid", RedirectURI: "ruri"}

}

func (s *HandlersTestSuite) TestGetIdentity() {
	username, err := s.handler.GetIdentity("bearer")
	s.Nil(err)
	s.Equal("test", username)
}

func (s *HandlersTestSuite) TestNew() {
	// Trying to create handler with nil config should fail
	h, err := New(nil)
	s.NotNil(err)
	s.Nil(h)

	// Should be able to successfully create a handler
	h, err = New(s.handler.conf)
	s.Nil(err)
	s.NotNil(h)
}

func (s *HandlersTestSuite) TestAddRedditKeys() {
	vals := make(url.Values)

	// We should have all the needed reddit values added after calling the funciton
	newVals := s.handler.addRedditKeys(vals, "test")
	s.Contains(newVals, "client_id")
	s.Contains(newVals, "response_type")
	s.Contains(newVals, "state")
	s.Contains(newVals, "redirect_uri")
	s.Contains(newVals, "duration")
	s.Contains(newVals, "scope")
	// TODO: cant testthe value of this yet since its pulled from env var -once its moved to config file we can
	//	s.Equal(newVals["client_id"])
	s.Equal(newVals["response_type"][0], "code")
	s.Equal(newVals["state"][0], "test")
	s.Equal(newVals["redirect_uri"][0], s.handler.conf.RedirectURI)
	s.Equal(newVals["duration"][0], "permanent")
	s.Equal(newVals["scope"][0], redditAPIScope)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}
