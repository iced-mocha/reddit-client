package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

const (
	accessTokenURL = "https://www.reddit.com/api/v1/access_token"
	authorizeURL   = "https://www.reddit.com/api/v1/authorize"
	userAgent      = "web:icedmocha:v0.0.1 (by /u/icedmoch)"
)

type CoreHandler struct{}

func addRedditKeys(vals url.Values) url.Values {
	vals.Add("client_id", "2fRgcQCHkIAqkw")
	vals.Add("response_type", "code")
	// This string should be random
	vals.Add("state", "test")
	// This must match the uri registered on reddit
	vals.Add("redirect_uri", "http://localhost:3000/v1/authorize_callback")
	vals.Add("duration", "permanent")
	vals.Add("scope", "history identity mysubreddits read")

	return vals
}

// Helper function for fetching auth token fro reddit
func (api *CoreHandler) requestOauth() ([]byte, error) {

	client := &http.Client{}

	req, err := http.NewRequest("GET", authorizeURL, nil)
	if err != nil {
		log.Fatalf("unable to reach reddit", err)
	}

	req.Header.Set("User-Agent", userAgent)

	q := req.URL.Query()
	// Add our reddit specific keys to our URL
	req.URL.RawQuery = addRedditKeys(q).Encode()

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Errored when sending request to the server")
		return nil, err
	}
	defer resp.Body.Close()
	contents, _ := ioutil.ReadAll(resp.Body)

	log.Printf("Successfully made request\n")

	return contents, nil

}

func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("User-Agent", userAgent)

	client := &http.Client{}

	resp, err := client.Get("http://www.reddit.com/user/icedmoch/upvoted")
	if err != nil {
		log.Fatalf("error reaching reddit %v", err)
	}

	// Make sure we close the body
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reaching reddit %v", err)
	}

	w.Write(body)
}

// We get redirected back here after attemp to retrieve an oauth token from Reddit
func (api *CoreHandler) AuthorizeCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("Reached callback")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("User-Agent", userAgent)
	w.Header().Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", userAgent)
	r.Header.Set("Content-Type", "application/json")

	// Get the query string
	vals := r.URL.Query()

	// If "error" is not an empty string we have not received
	// our bearer token

	if len(vals["error"]) != 0 {
		log.Printf("Did not receive authorization")
		// For now return
		return
	}

	// We need to verify that this state matches what we sent
	fmt.Printf("State: %v", vals["state"])

	// We need to verify that this state matches what we sent
	fmt.Printf("Code: %v", vals["code"])
	// Otherwise we have no errors so lets take our bear token and
	// get our auth token

	//api.requestCode(vals["code"][0])

	w.Write([]byte("{hello: \"test\"}"))
}

func (api *CoreHandler) requestCode(code string) {
	var jsonStr = []byte("grant_type=authorization_code&code=" + code + "&redirect_uri=http://localhost:3000/v1/authorize_callback")

	req, err := http.NewRequest("POST", accessTokenURL, bytes.NewBuffer(jsonStr))
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth("2fRgcQCHkIAqkw", "SECRET")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

// Endpoint for Requesting an oauth token from reddit
func (api *CoreHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	URL, _ := url.Parse("https://www.reddit.com/api/v1/authorize")

	q := URL.Query()

	q.Add("client_id", "2fRgcQCHkIAqkw")
	q.Add("response_type", "code")
	q.Add("state", "test")
	q.Add("redirect_uri", "http://localhost:3000/v1/authorize_callback")
	q.Add("duration", "permanent")
	q.Add("scope", "history identity mysubreddits read")
	URL.RawQuery = q.Encode()

	http.Redirect(w, r, URL.String(), 301)
}
