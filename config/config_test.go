package config

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	permissions = 0755
	validConfig = `
frontend-url: "frontend"
core-url: "core"
redirect-uri: "redirect"
reddit-secret: "secret"
reddit-client-id: "clientid"
`
	invalidConfig = `
front:    end-url: "http://localhost:8080"
core-u:			rl: "http://localhost:3000"
redirect-uri: "http://localhost:3001/v1/authorize_callback"
reddit-secret: "se"
reddit-client-id: "2fRgcQCHkIAqkw"
`
)

type ConfigTestSuite struct {
	suite.Suite
	config *Config
}

func (s *ConfigTestSuite) SetupSuite() {
	log.SetOutput(ioutil.Discard)

	s.config = &Config{FrontendURL: "frontend", CoreURL: "core", RedirectURI: "redirect", RedditSecret: "secret", RedditClientID: "clientid"}
}

func (s *ConfigTestSuite) TestNew() {
	// Write our test yml to a temp file
	tmpfile, err := ioutil.TempFile("/tmp", "config")
	s.Nil(err)
	defer os.Remove(tmpfile.Name())

	// Invalid config should cause New to fail
	s.Nil(ioutil.WriteFile(tmpfile.Name(), []byte(invalidConfig), permissions))
	conf, err := New(tmpfile.Name())
	s.NotNil(err)
	s.Nil(conf)

	// Valid config should succeed
	s.Nil(ioutil.WriteFile(tmpfile.Name(), []byte(validConfig), permissions))
	conf, err = New(tmpfile.Name())
	s.Nil(err)
	s.Equal(s.config, conf)

}

func TestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
