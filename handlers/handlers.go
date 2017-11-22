package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/reddit-client/config"
	"github.com/iced-mocha/shared/models"
)

const (
	redditBaseURL       = "https://www.reddit.com"
	accessTokenEndpoint = "/api/v1/access_token"
	authorizeEndpoint   = "/api/v1/authorize"
	identityEndpoint    = "/api/v1/me"
	settingsEndpoint    = "/settings"
	userAgent           = "web:icedmocha:v0.0.1 (by /u/icedmoch)"

	// This words give us access to specific things in Reddit API - see docs for more info
	redditAPIScope = "history identity mysubreddits read"
)

type CoreHandler struct {
	client *http.Client
	conf   *config.Config
}

type AuthRequest struct {
	BearerToken string
}

// Struct for the response from reddit when request a bearer token
type RedditAuthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

// Struct for the response from reddit when GETting the /api/v1/me endpoint
type IdentityResponse struct {
	RedditUsername string `json:"name"`
}

type RedditImage struct {
	Source struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"source"`
}

type RedditPost struct {
	ID           string        `json:"id"`
	Author       string        `json:"author"`
	URL          string        `json:"url"`
	Title        string        `json:"title"`
	RelativePath string        `json:"permalink"`
	PostLink     string        `json:"url"`
	Subreddit    string        `json:"subreddit_name_prefixed"`
	Images       []RedditImage `json:"images"`
	Gif          RedditImage   `json:"gif"`
	Score        int           `json:"score"`
	UnixTime     float64       `json:"created_utc"`
}

type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		Children []struct {
			Data RedditPost
		}
	}
}

func New(conf *config.Config) (*CoreHandler, error) {
	if conf == nil {
		return nil, errors.New("must initialize handler with non-nil config")
	}

	h := &CoreHandler{client: &http.Client{}}
	h.conf = conf
	return h, nil
}

// Consumes an existing values object and adds keys that are required for reddit oauth
func (api *CoreHandler) addRedditKeys(vals url.Values, userID string) url.Values {
	// These values are mandated by reddit oauth docs
	vals.Add("client_id", api.conf.RedditClientID)
	vals.Add("response_type", "code")
	// TODO: This state value should be randomly generated each time - for now we are setting it as the users userID - not sure which is better
	vals.Add("state", userID)
	// This must match the uri registered on reddit
	vals.Add("redirect_uri", api.conf.RedirectURI)
	vals.Add("duration", "permanent")
	vals.Add("scope", redditAPIScope)

	return vals
}

func (api *CoreHandler) GetIdentity(bearerToken string) (string, error) {
	// Make a request to get identity from Reddit
	req, err := http.NewRequest(http.MethodGet, api.conf.RedditOAuthURL+identityEndpoint, nil)
	if err != nil {
		return "", err
	}

	// Attach our bearer token
	req.Header.Add("Authorization", "bearer "+bearerToken)
	// This is required by the Reddit API terms and conditions
	req.Header.Add("User-Agent", userAgent)

	resp, err := api.client.Do(req)
	if err != nil {
		log.Printf("Errored when retrieving identity from Reddit: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Did not receive 200 OK when trying to get identity from reddit. Received: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read response body from Reddit: %v", err)
		return "", err
	}

	id := IdentityResponse{}
	err = json.Unmarshal(body, &id)
	if err != nil {
		log.Printf("Unable to unmarshall response: %v\n", err)
		return "", err
	}

	log.Printf("Received identity for user: %v.", id.RedditUsername)
	return id.RedditUsername, nil
}

// TODO: Store bearer token in url
// Consumes a request object and parse the bearer token contained in the body
func (api *CoreHandler) getBearerToken(r *http.Request) (string, error) {
	// Parse body and get bearer token from the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	// Unmarshall response containing our bearer token
	authRequest := &AuthRequest{}
	err = json.Unmarshal(body, authRequest)
	if err != nil {
		return "", nil
	}

	if authRequest.BearerToken == "" {
		return "", errors.New("received empty bearer token in request")
	}

	return authRequest.BearerToken, nil
}

// Fetches post from Reddit
// GET /v1/{id}/posts
func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := api.getBearerToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make a request to get posts from Reddit
	req, err := http.NewRequest(http.MethodGet, "http://oauth.reddit.com/r/all", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Attach our bearer token
	req.Header.Add("Authorization", "bearer "+bearerToken)
	// This is required by the Reddit API terms and conditions
	req.Header.Add("User-Agent", userAgent)

	resp, err := api.client.Do(req)
	if err != nil {
		log.Printf("Errored when sending request to the server: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Printf("Received response from Reddit with status code: %v while getting posts", resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to complete request: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// We need to get rid of some the meta data that comes with the response
	vals := RedditResponse{}
	err = json.Unmarshal(body, &vals)
	if err != nil {
		log.Printf("Unable to unmarshall response: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts := []models.Post{}
	for _, c := range vals.Data.Children {
		post := c.Data

		var heroImg string
		if post.Gif.Source.URL != "" {
			heroImg = post.Gif.Source.URL
		} else if len(post.Images) >= 1 {
			// TODO: Pick best image instead of first one
			heroImg = post.Images[0].Source.URL
		}

		generic := models.Post{
			ID:       post.ID,
			Date:     time.Unix(int64(post.UnixTime), 10),
			Author:   post.Author,
			Title:    post.Title,
			HeroImg:  heroImg,
			PostLink: "https://reddit.com" + post.RelativePath,
			Platform: "reddit",
			URL:      post.URL,
		}

		posts = append(posts, generic)
	}

	res, err := json.Marshal(posts)
	if err != nil {
		log.Printf("Unable to marshall response: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(res)
}

// Posts Reddit Username and bearer token to be stored in core
func (api *CoreHandler) postBearerToken(bearerToken, userID string) {
	// Post the bearer token to be saved in core
	log.Printf("Preparing to store reddit account in core for user: %v", userID)
	redditUsername, err := api.GetIdentity(bearerToken)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	jsonStr := []byte(fmt.Sprintf(`{ "type": "reddit", "username": "%v", "token": "%v"}`, redditUsername, bearerToken))
	req, err := http.NewRequest(http.MethodPost, api.conf.CoreURL+"/v1/users/"+userID+"/authorize/reddit", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Printf("Unable to post bearer token for user: %v - %v", userID, err)
		return
	}

	// TODO: add retry logic
	resp, err := api.client.Do(req)
	if err != nil {
		log.Printf("Unable to complete request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Could not post reddit data to core: %v", err)
	}
}

// We get redirected back here after attempt to retrieve an oauth code from Reddit
func (api *CoreHandler) AuthorizeCallback(w http.ResponseWriter, r *http.Request) {
	log.Println("Reaceived callback from Reddit oauth")

	// Get the query string
	vals := r.URL.Query()

	// If "error" is not an empty string we have not received our access code
	// This is error param is specified by the Reddit API
	if val, ok := vals["error"]; ok {
		if len(val) != 0 {
			log.Printf("Did not receive authorization. Error: %v\n", vals["error"][0])
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

	// Post code back to core async as the rest is not dependant on this -- vals["state"] should be userID
	go api.postBearerToken(bearerToken, vals["state"][0])

	// Redirect to frontend
	http.Redirect(w, r, api.conf.FrontendURL+settingsEndpoint, http.StatusMovedPermanently)
}

// Helper function to request a bearer token from reddit using the given code
// Returns: the bearer token and an error should one occur
func (api *CoreHandler) requestToken(code string) (string, error) {
	log.Printf("About to request bearer token for code: %v\n", code)
	jsonStr := []byte("grant_type=authorization_code&code=" + code + "&redirect_uri=" + api.conf.RedirectURI)

	// Prepare the request for the bearer token
	req, err := http.NewRequest(http.MethodPost, redditBaseURL+accessTokenEndpoint, bytes.NewBuffer(jsonStr))
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(api.conf.RedditClientID, api.conf.RedditSecret)

	resp, err := api.client.Do(req)
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
	authResponse := &RedditAuthResponse{}
	err = json.Unmarshal(body, authResponse)
	if err != nil {
		log.Printf("Unable to parse response from reddit: %v\n", err)
		return "", err
	}

	return authResponse.AccessToken, nil
}

// This function initiates a request from Reddit to authorize via oauth
// GET /v1/{userID}/authorize
func (api *CoreHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	URL, err := url.Parse(redditBaseURL + authorizeEndpoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the userID from the path
	vars := mux.Vars(r)

	// Add the keys required for requesting oauth from Reddit
	URL.RawQuery = api.addRedditKeys(URL.Query(), vars["userID"]).Encode()

	// Redirect to reddit to request oauth
	http.Redirect(w, r, URL.String(), http.StatusFound)
}
