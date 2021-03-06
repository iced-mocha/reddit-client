package handlers

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/html"
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

	// These words give us access to specific things in Reddit API - see docs for more info
	redditAPIScope   = "history identity mysubreddits read"
	targetImageWidth = 600
)

type CoreHandler struct {
	client *http.Client
	conf   *config.Config
}

type AuthRequest struct {
	BearerToken  string `json:"bearer-token"`
	RefreshToken string `json:"refresh-token"`
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

type ImageSource struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type RedditImage struct {
	Source   *ImageSource `json:"source"`
	Variants struct {
		GIF struct {
			Source      *ImageSource   `json:"source"`
			Resolutions []*ImageSource `json:"resolutions"`
		} `json:"gif"`
		MP4 struct {
			Source      *ImageSource   `json:"source"`
			Resolutions []*ImageSource `json:"resolutions"`
		} `json:"mp4"`
	} `json:"variants"`
	Resolutions []*ImageSource `json:"resolutions"`
}

type RedditPost struct {
	ID           string `json:"id"`
	Author       string `json:"author"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	RelativePath string `json:"permalink"`
	Subreddit    string `json:"subreddit"`
	Preview      struct {
		Images []RedditImage `json:"images"`
	} `json:"preview"`
	Score    int     `json:"score"`
	UnixTime float64 `json:"created_utc"`
	IsVideo  bool    `json:"is_video"`
	Content  string  `json:"selftext_html"`
}

type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		Children []struct {
			Data RedditPost `json:"data"`
		} `json:"children"`
		After string `json:"after"`
	} `json:"data"`
}

func New(conf *config.Config) (*CoreHandler, error) {
	if conf == nil {
		return nil, errors.New("must initialize handler with non-nil config")
	}

	caCert, err := ioutil.ReadFile("/usr/local/etc/ssl/certs/core.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	h := &CoreHandler{client: client}
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

	resp, err := http.DefaultClient.Do(req)
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
func (api *CoreHandler) getRedditAuth(r *http.Request) (*AuthRequest, error) {
	// Parse body and get bearer token from the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshall response containing our bearer token
	authRequest := &AuthRequest{}
	err = json.Unmarshal(body, authRequest)
	if err != nil {
		log.Printf("Received invalid reddit auth information")
		return nil, err
	}

	return authRequest, nil
}

func getBestImage(image RedditImage) string {
	gif := image.Variants.GIF
	bestImage := getBestResolution(append(gif.Resolutions, gif.Source))
	if bestImage == "" {
		bestImage = getBestResolution(append(image.Resolutions, image.Source))
	}
	return html.UnescapeString(bestImage)
}

func getBestVideo(image RedditImage) string {
	mp4 := image.Variants.MP4
	bestVideo := getBestResolution(append(mp4.Resolutions, mp4.Source))
	return html.UnescapeString(bestVideo)
}

func getBestResolution(images []*ImageSource) string {
	var bestImage string
	var bestImageWidth int
	for _, i := range images {
		if i == nil {
			continue
		} else if i.Width > targetImageWidth && i.Width < bestImageWidth ||
			bestImageWidth < targetImageWidth && i.Width > bestImageWidth {

			bestImage = i.URL
			bestImageWidth = i.Width
		}
	}
	return bestImage
}

func getContentHTML(content string) string {
	prefixLen := len("&lt;!-- SC_OFF --&gt;")
	suffixLen := len("&lt;!-- SC_ON --&gt;")
	if len(content) < prefixLen {
		return ""
	}
	innerContent := content[prefixLen : len(content)-suffixLen]
	return html.UnescapeString(innerContent)
}

func (api *CoreHandler) getPostsAuth(query, token string) (*http.Request, error) {
	url := "http://oauth.reddit.com/"

	req, err := http.NewRequest(http.MethodGet, url+query, nil)
	if err != nil {
		return nil, err
	}

	// Attach our bearer token
	req.Header.Add("Authorization", "bearer "+token)

	// This is required by the Reddit API terms and conditions
	req.Header.Add("User-Agent", userAgent)

	return req, nil
}

func (api *CoreHandler) getPosts(query string) (*http.Request, error) {
	url := "http://www.reddit.com/.json"

	req, err := http.NewRequest(http.MethodGet, url+query, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)

	return req, nil
}

// Caller must close resp.Body
func (api *CoreHandler) completeRequest(auth *AuthRequest, username string, req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Errored when sending request to the Reddit: %v", err)
		return nil, err
	}

	// If we get a 401 back from reddit lets try again but first refresh our token
	if resp.StatusCode == 401 && auth.BearerToken != "" && auth.RefreshToken != "" {
		resp.Body.Close()
		query := req.URL.RawQuery
		if len(query) > 0 {
			query = "?" + query
		}
		// First refresh our token
		auth, err := api.Refresh(auth.RefreshToken)
		if err != nil {
			return nil, err
		}

		go api.postRedditAuth(auth, username)
		req, err := api.getPostsAuth(query, auth.BearerToken)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}

	return resp, nil
}

// Fetches post from Reddit
// GET /v1/{id}/posts
func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	var pageToken string
	if arr, ok := queryParams["continue"]; ok && len(arr) > 0 {
		pageToken = arr[0]
	}
	log.Printf("Received page token: %v", pageToken)

	id := mux.Vars(r)["id"]
	redditAuth, err := api.getRedditAuth(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var query string
	if pageToken != "" {
		query += "?after=" + pageToken
	}

	var req *http.Request
	if redditAuth.BearerToken == "" {
		req, err = api.getPosts(query)
	} else {
		req, err = api.getPostsAuth(query, redditAuth.BearerToken)
	}

	resp, err := api.completeRequest(redditAuth, id, req)
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
	log.Printf("Reddit response: %+v", vals.Data.After)

	posts := []models.Post{}
	for _, c := range vals.Data.Children {
		post := c.Data

		var heroImg string
		var video string
		if len(post.Preview.Images) > 0 {
			mainImage := post.Preview.Images[0]
			heroImg = getBestImage(mainImage)
			video = getBestVideo(mainImage)
		}

		generic := models.Post{
			ID:        post.ID,
			Date:      time.Unix(int64(post.UnixTime), 10),
			Author:    post.Author,
			Title:     html.UnescapeString(post.Title),
			HeroImg:   heroImg,
			Video:     video,
			IsVideo:   post.IsVideo,
			PostLink:  "https://reddit.com" + post.RelativePath,
			Platform:  "reddit",
			URL:       post.URL,
			Score:     post.Score,
			Subreddit: post.Subreddit,
			Content:   getContentHTML(post.Content),
		}

		posts = append(posts, generic)
	}

	var nextURL string
	if vals.Data.After != "" {
		// TODO: This is needed because we use this same code for both authenitcated
		// and non authenticated requests. It is gross and should be fixed soon
		if id == "" {
			nextURL = fmt.Sprintf("%v/v1/posts?continue=%v", api.conf.RedditClientURL, vals.Data.After)
		} else {
			nextURL = fmt.Sprintf("%v/v1/%v/posts?continue=%v", api.conf.RedditClientURL, id, vals.Data.After)
		}
	}
	clientResp := models.ClientResp{
		Posts:   posts,
		NextURL: nextURL,
	}

	res, err := json.Marshal(clientResp)
	if err != nil {
		log.Printf("Unable to marshall response: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(res)
}

func (api *CoreHandler) GetPostsNoAuth(w http.ResponseWriter, r *http.Request) {
	// TODO: This is gross, fix it. Make generic function to handle reddit posts, call it from GetPosts and GetPostsNoAuth
	api.GetPosts(w, r)
}

// Posts Reddit Username and bearer token to be stored in core
func (api *CoreHandler) postRedditAuth(auth *AuthRequest, userID string) {
	// Post the bearer token to be saved in core
	log.Printf("Preparing to store reddit account in core for user: %v", userID)
	redditUsername, err := api.GetIdentity(auth.BearerToken)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	jsonStr := []byte(fmt.Sprintf(`{ "type": "reddit", "username": "%v", "token": "%v", "refresh-token": "%v"}`,
		redditUsername, auth.BearerToken, auth.RefreshToken))
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

	var rAuth *RedditAuthResponse
	var err error

	// Make sure the code exists
	if len(vals["code"]) > 0 {
		// Now request bearer token using the code we received
		rAuth, err = api.requestToken(vals["code"][0])
		if err != nil {
			log.Printf("Unable to receive bearer token: %v\n", err)
			return
		}
	}

	auth := &AuthRequest{BearerToken: rAuth.AccessToken, RefreshToken: rAuth.RefreshToken}
	// Post code back to core async as the rest is not dependant on this -- vals["state"] should be userID
	go api.postRedditAuth(auth, vals["state"][0])

	// Redirect to frontend
	http.Redirect(w, r, api.conf.FrontendURL+settingsEndpoint, http.StatusMovedPermanently)
}

// Helper function to request a bearer token from reddit using the given code
// Returns: the bearer token and an error should one occur
func (api *CoreHandler) requestToken(code string) (*RedditAuthResponse, error) {
	log.Printf("About to request bearer token for code: %v\n", code)
	jsonStr := []byte("grant_type=authorization_code&code=" + code + "&redirect_uri=" + api.conf.RedirectURI)

	// Prepare the request for the bearer token
	req, err := http.NewRequest(http.MethodPost, redditBaseURL+accessTokenEndpoint, bytes.NewBuffer(jsonStr))
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(api.conf.RedditClientID, api.conf.RedditSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Unable to complate request for bearer token: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read response body: %v\n", err)
		return nil, err
	}

	// Unmarshall response containing our bearer token
	authResponse := &RedditAuthResponse{}
	err = json.Unmarshal(body, authResponse)
	if err != nil {
		log.Printf("Unable to parse response from reddit: %v\n", err)
		return nil, err
	}

	return authResponse, nil
}

func (api *CoreHandler) Refresh(refreshToken string) (*AuthRequest, error) {
	url := "https://www.reddit.com/api/v1/access_token"
	contents := []byte(fmt.Sprintf("grant_type=refresh_token&refresh_token=%v", refreshToken))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(contents))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(api.conf.RedditClientID, api.conf.RedditSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Unable to complate refreshing token: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read response body: %v\n", err)
		return nil, err
	}

	authResponse := &RedditAuthResponse{}
	err = json.Unmarshal(body, authResponse)
	if err != nil {
		log.Printf("Unable to parse response from reddit: %v\n", err)
		return nil, err
	}

	auth := &AuthRequest{BearerToken: authResponse.AccessToken, RefreshToken: refreshToken}
	return auth, nil
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
