package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type CoreHandler struct{}

func (api *CoreHandler) requestOauth() ([]byte, error) {

	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://www.reddit.com/api/v1/authorize", nil)
	if err != nil {
		log.Fatalf("unable to reach reddit", err)
	}

	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("User-Agent", "web:icedmocha:v0.0.1 (by /u/icedmoch)")

	q := req.URL.Query()
	q.Add("client_id", "2fRgcQCHkIAqkw")
	q.Add("response_type", "code")
	// This string should be random
	q.Add("state", "test")
	// This must match the uri registered on reddit
	q.Add("redirect_uri", "http://localhost:3000/v1/authorize_callback")
	q.Add("duration", "permanent")
	q.Add("scope", "history identity mysubreddits read")
	req.URL.RawQuery = q.Encode()

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
	w.Header().Set("User-Agent", "web:icedmocha:v0.0.1 (by /u/icedmoch)")

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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("User-Agent", "web:icedmocha:v0.0.1 (by /u/icedmoch)")
	w.Header().Set("Content-Type", "application/json")
	r.Header.Set("Access-Control-Allow-Origin", "*")
	r.Header.Set("User-Agent", "web:icedmocha:v0.0.1 (by /u/icedmoch)")
	r.Header.Set("Content-Type", "application/json")

	// Get the query string
	vals := r.URL.Query()

	// If "error" is not an empty string we have not received
	// our bearer token

	if vals["error"][0] != "" {
		log.Printf("Did not receive authorization")
		// For now return
		return
	}

	// We need to verify that this state matches what we sent
	fmt.Printf("State: %v", vals["state"])

	// Otherwise we have no errors so lets take our bear token and
	// get our auth token

	api.requestCode(vals["code"][0])

	w.Write([]byte("{hello: \"test\"}"))
}

func (api *CoreHandler) requestCode(code string) {
	url := "https://www.reddit.com/api/v1/access_token"

	var jsonStr = []byte("grant_type=authorization_code&code=" + code + "&redirect_uri=http://localhost:3000/v1/authorize_callback")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.SetBasicAuth("2fRgcQCHkIAqkw", "wF2eLhE3M4jjiXM4JIe2Nfj0y5o")

	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("User-Agent", "web:icedmocha:v0.0.1 (by /u/icedmoch)")

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

// Requests an oauth token from reddit
func (api *CoreHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("User-Agent", "web:icedmocha:v0.0.1 (by /u/icedmoch)")
	w.Header().Set("Content-Type", "text/html")
	r.Header.Set("Access-Control-Allow-Origin", "*")
	r.Header.Set("User-Agent", "web:icedmocha:v0.0.1 (by /u/icedmoch)")
	r.Header.Set("Content-Type", "application/json")

	// The contents of this call will be a webpage asking for users authentication
	contents, _ := api.requestOauth()

	w.Write(contents)
}
