package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

const (
	redirectURI         = "http://localhost:3000/v1/authorize_callback"
	redditBaseURL       = "https://www.reddit.com"
	accessTokenEndpoint = "/api/v1/access_token"
	authorizeEndpoint   = "/api/v1/authorize"
	userAgent           = "web:icedmocha:v0.0.1 (by /u/icedmoch)"
	redditSecretKey     = "REDDIT_SECRET"
	redditClientIDKey   = "REDDIT_CLIENT_ID"
)

type CoreHandler struct{}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

// Consumes an existing values and adds keys that are required for reddit oauth
func (api *CoreHandler) addRedditKeys(vals url.Values) url.Values {
	vals.Add("client_id", os.Getenv(redditClientIDKey))
	log.Printf("Adding client id %v\n", os.Getenv(redditClientIDKey))
	vals.Add("response_type", "code")
	// This string should be random
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
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, redditBaseURL+authorizeEndpoint, nil)
	if err != nil {
		log.Fatal(err)
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

func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("User-Agent", userAgent)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://oauth.reddit.com/r/hockey/hot", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "bearer "+os.Getenv("BEARER_TOKEN"))
	req.Header.Add("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Errored when sending request to the server: %v\n", err)
		return
	}

	// Make sure we close the body
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reaching reddit %v", err)
	}

	w.Write(body)
}

// We get redirected back here after attemp to retrieve an oauth code from Reddit
func (api *CoreHandler) AuthorizeCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("Reached callback")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("User-Agent", userAgent)

	// Get the query string
	vals := r.URL.Query()
	log.Printf("%v\n", vals)

	// If "error" is not an empty string we have not received our access code
	if val, ok := vals["error"]; ok {
		if len(val) != 0 {
			log.Printf("Did not receive authorization")
			// For now return
			return
		}
	}

	// TODO: need to verify that this state matches what we sent
	//fmt.Printf("State: %v", vals["state"])

	// Now request code
	api.requestToken(vals["code"][0])

	w.Write([]byte("{hello: \"test\"}"))
}

// Requests the bearer token from reddit using the given code
func (api *CoreHandler) requestToken(code string) {
	log.Println("Preparing to request bearer token")
	jsonStr := []byte("grant_type=authorization_code&code=" + code + "&redirect_uri=" + redirectURI)

	log.Printf("About to send request to %v%v\n", redditBaseURL, accessTokenEndpoint)
	req, err := http.NewRequest(http.MethodPost, redditBaseURL+accessTokenEndpoint, bytes.NewBuffer(jsonStr))
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(os.Getenv(redditClientIDKey), os.Getenv(redditSecretKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Response body %v\n", string(body))

	// Unmarshall response containing our bearer token
	authResponse := &AuthResponse{}
	err = json.Unmarshal(body, authResponse)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(authResponse.AccessToken)
}

// Endpoint for Requesting an oauth token from reddit
func (api *CoreHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	URL, err := url.Parse(redditBaseURL + authorizeEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	// Add the keys required for requesting oauth
	q := api.addRedditKeys(URL.Query())
	URL.RawQuery = q.Encode()

	http.Redirect(w, r, URL.String(), http.StatusMovedPermanently)
}
