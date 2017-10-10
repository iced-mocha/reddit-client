package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

const (
	frontEndURL         = "http://localhost:8080"
	redirectURI         = "http://localhost:3001/v1/authorize_callback"
	redditBaseURL       = "https://www.reddit.com"
	accessTokenEndpoint = "/api/v1/access_token"
	authorizeEndpoint   = "/api/v1/authorize"
	userAgent           = "web:icedmocha:v0.0.1 (by /u/icedmoch)"
	redditSecretKey     = "REDDIT_SECRET"
	redditClientIDKey   = "REDDIT_CLIENT_ID"
)

type CoreHandler struct{}

type AuthRequest struct {
	BearerToken string
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

var client *http.Client

func init() {
	client = &http.Client{}
}

// Consumes an existing values and adds keys that are required for reddit oauth
func (api *CoreHandler) addRedditKeys(vals url.Values) url.Values {
	// These values are mandated by reddit oauth docs
	vals.Add("client_id", os.Getenv(redditClientIDKey))
	vals.Add("response_type", "code")
	// TODO: This state value should be randomly generated each time
	vals.Add("state", "test")
	// This must match the uri registered on reddit
	vals.Add("redirect_uri", redirectURI)
	vals.Add("duration", "permanent")
	vals.Add("scope", "history identity mysubreddits read")

	return vals
}

// Helper function for fetching auth token fro reddit
func (api *CoreHandler) requestOauth() ([]byte, error) {
	log.Println("Preparing to request for oauth token from reddit")

	req, err := http.NewRequest(http.MethodGet, redditBaseURL+authorizeEndpoint, nil)
	if err != nil {
		log.Println("Errored when creating request")
	}

	req.Header.Set("User-Agent", userAgent)
	q := req.URL.Query()

	// Add our reddit specific keys to our URL
	req.URL.RawQuery = api.addRedditKeys(q).Encode()

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Errored when sending request to the server")
		return nil, err
	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error occured when reading response body")
		return nil, err
	}

	return contents, nil
}

// Fetches post from Reddit
// /v1/{id}/posts
func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	// Parse body and get bearer token from the request
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Body of GetPosts request%v\n", string(body))

	// Unmarshall response containing our bearer token
	authRequest := &AuthRequest{}
	err = json.Unmarshal(body, authRequest)
	if err != nil {
		log.Printf("Unable to parse response body: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make a request to get posts from Reddit
	req, err := http.NewRequest(http.MethodGet, "http://oauth.reddit.com/r/hockey/hot", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Attach our bearer token
	req.Header.Add("Authorization", "bearer "+authRequest.BearerToken)
	req.Header.Add("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Errored when sending request to the server: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make sure we close the body
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to complete request: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func (api *CoreHandler) postBearerToken(bearerToken string) {
	// Post the bearer token to be saved in core
	// TODO: Where is the UserID passed?
	jsonStr := []byte(fmt.Sprintf("{ \"bearertoken\": \"%v\"", bearerToken))

	req, err := http.NewRequest(http.MethodPost, redditBaseURL+accessTokenEndpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Printf("Unable to post bearer token: %v\n", err)
	}

	// TODO: add retry logic
	_, err = client.Do(req)
	if err != nil {
		log.Printf("Unable to complete request: %v\n", err)
	}
}

// We get redirected back here after attemp to retrieve an oauth code from Reddit
func (api *CoreHandler) AuthorizeCallback(w http.ResponseWriter, r *http.Request) {
	log.Println("Reaceived callback from Reddit oauth")

	// Get the query string
	vals := r.URL.Query()

	// If "error" is not an empty string we have not received our access code
	if val, ok := vals["error"]; ok {
		if len(val) != 0 {
			log.Printf("Did not receive authorization. Error: %v\n", vals["errpr"][0])
			return
		}
	}

	// TODO: need to verify that this state matches what we sent
	//fmt.Printf("State: %v", vals["state"])

	var bearerToken string
	var err error

	// Make sure the code exists
	if len(vals["code"]) > 0 {
		// Now request bearer token using the code we received
		bearerToken, err = api.requestToken(vals["code"][0])
		if err != nil {
			log.Printf("Unable to receive bearer token: %v\n", err)
			return
		}
	}

	// Post code back to core async as the rest is not dependant on this
	go api.postBearerToken(bearerToken)

	// Redirect to frontend
	http.Redirect(w, r, frontEndURL, http.StatusMovedPermanently)
}

// Helper function to request a bearer token from reddit using the given code
// Returns the bearer token and an error should one occur
func (api *CoreHandler) requestToken(code string) (string, error) {
	log.Printf("About to request bearer token for code: %v\n", code)
	jsonStr := []byte("grant_type=authorization_code&code=" + code + "&redirect_uri=" + redirectURI)

	// Prepare the request for the token
	req, err := http.NewRequest(http.MethodPost, redditBaseURL+accessTokenEndpoint, bytes.NewBuffer(jsonStr))
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(os.Getenv(redditClientIDKey), os.Getenv(redditSecretKey))

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to complate request for bearer token: %v\n", err)
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read response body: %v\n", err)
		return "", err
	}

	// Unmarshall response containing our bearer token
	authResponse := &AuthResponse{}
	err = json.Unmarshal(body, authResponse)
	if err != nil {
		log.Printf("Unable to parse response from reddit: %v\n", err)
		return "", err
	}

	return authResponse.AccessToken, nil
}

// This function initiates a request from Reddit to authorize via oauth
// GET /v1/authorize
func (api *CoreHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	URL, err := url.Parse(redditBaseURL + authorizeEndpoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add the keys required for requesting oauth from Reddit
	URL.RawQuery = api.addRedditKeys(URL.Query()).Encode()

	// Redirect to reddit to request oauth
	http.Redirect(w, r, URL.String(), http.StatusFound)
}
