package handlers

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/suite"
)

type HandlersTestSuite struct {
	suite.Suite
	handler CoreHandler
}

func (suite *HandlersTestSuite) SetupSuite() {
	// Disable logging while testing
	log.SetOutput(ioutil.Discard)

	suite.handler = CoreHandler{Client: &http.Client{}}
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
	s.Equal(newVals["redirect_uri"][0], redirectURI)
	s.Equal(newVals["duration"][0], "permanent")
	s.Equal(newVals["scope"][0], redditAPIScope)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}
