package splunk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Client struct {
	Username  string            `json:"username"`
	SessionID string            `json:"session_id"`
	Addr      string            `json:"addr"`
	Searches  map[string]string `json:"searches"`
	httpcli   http.Client
}

func (c *Client) ToJSON() string {
	byts, _ := json.MarshalIndent(c, "", "    ")
	return string(byts)
}

func (c *Client) SaveTo(fileloc string) error {
	err := ioutil.WriteFile(fileloc, []byte(c.ToJSON()), 0644)
	if err != nil {
		return err
	}
	return nil
}

func LoadClient(fileloc string) (*Client, error) {
	ret := Client{Searches: make(map[string]string)}

	// open and parse json settings file
	file, err := os.Open(fileloc)
	if err != nil {
		return &ret, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ret)
	if err != nil {
		return &ret, err
	}
	return &ret, nil
}

func New(addr string) *Client { return &Client{Addr: addr, Searches: make(map[string]string)} }

//  NewSessionID attempts to authenticate with `addr` using `username`
//  and `password` and returns a sessionID if successful. An optional
//  `*http.Client` `cli` may be passed in. If nil, a default one will
//  be used
func NewSessionID(addr, username, password string, httpClient *http.Client) (string, error) {
	// curl https://splunk.sendgrid.net:8089/services/auth/login -d username=$SPLUNK_USER -d password="$SPLUNK_PASS"
	cli := http.Client{}
	if httpClient != nil {
		cli = *httpClient
	}
	urlstr := fmt.Sprintf("%s/services/auth/login", addr)
	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	data.Set("output_mode", "json")
	req, err := http.NewRequest("POST", urlstr, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	byt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("non-200 return code: %d, response: %s", resp.StatusCode, string(byt))
	}
	type expectedResposne struct {
		SessionKey string `json:"sessionKey"`
	}
	var exp expectedResposne
	err = json.Unmarshal(byt, &exp)
	if err != nil {
		return "", err
	}
	return exp.SessionKey, nil
}

func (c *Client) RenewSessionID(username, password string) (string, error) {
	sid, err := NewSessionID(c.Addr, username, password, &c.httpcli)
	if err != nil {
		return "", err
	}
	c.SessionID = sid
	c.Username = username
	return sid, nil
}

// TODO gotta be a better way
// func (c *Client) AuthCheck(search string) ( error) {
type SearchResponse struct {
	Response
	SearchID string
}

// TODO ensure search has a date
func (c *Client) Search(search string, opts ...Option) (SearchResponse, error) {
	// curl -H "Authorization: Splunk $SPLUNK_SESSION"
	//      https://splunk.sendgrid.net:8089/services/search/jobs
	//      -d output_mode=json
	//      -d search='search earliest=-4h event=processed | eval l=len(subject) | where l > 3000'
	var ret SearchResponse
	urlstr := fmt.Sprintf("%s/services/search/jobs", c.Addr)
	data := url.Values{}
	for _, opt := range opts {
		opt(data)
	}
	data.Set("output_mode", "json")
	data.Set("search", search)
	req, err := http.NewRequest("POST", urlstr, strings.NewReader(data.Encode()))
	if err != nil {
		return ret, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", fmt.Sprintf("Splunk %s", c.SessionID))
	resp, err := c.httpcli.Do(req)
	if err != nil {
		return ret, err
	}
	ret.Body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return ret, err
	}
	ret.StatusCode = resp.StatusCode
	if ret.AuthFailed() {
		return ret, ErrAuth
	}

	type expectedResposne struct {
		SearchID string `json:"sid"`
	}
	var exp expectedResposne
	err = json.Unmarshal(ret.Body, &exp)
	if err != nil {
		return ret, err
	}
	c.Searches[exp.SearchID] = search
	ret.SearchID = search
	return ret, nil
}

func WithParam(key string, value string) Option {
	return func(v url.Values) { v.Set(key, value) }
}

type Option func(url.Values)

func (c *Client) GetSearchResults(searchID string, opts ...Option) (Response, error) {
	// curl -H "Authorization: Splunk $SPLUNK_SESSION" -X GET https://splunk.sendgrid.net:8089/services/search/jobs/$SEARCH_ID/results -d output_mode=json

	var ret Response

	urlstr := fmt.Sprintf("%s/services/search/jobs/%s/results", c.Addr, searchID)
	req, err := http.NewRequest("GET", urlstr, nil)
	if err != nil {
		return ret, err
	}

	data := url.Values{}
	for _, opt := range opts {
		opt(data)
	}
	if _, ok := data["output_mode"]; !ok {
		data.Set("output_mode", "json")
	}

	req.URL.RawQuery = data.Encode()
	req.Header.Add("Authorization", fmt.Sprintf("Splunk %s", c.SessionID))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpcli.Do(req)
	if err != nil {
		return ret, err
	}
	ret.Body, err = ioutil.ReadAll(resp.Body)
	ret.StatusCode = resp.StatusCode
	if err != nil {
		return ret, err
	}
	return ret, nil
}

type Response struct {
	Body       []byte
	StatusCode int
}

var ErrAuth = fmt.Errorf("call not properly authenticated")

func (r *Response) AuthFailed() bool {
	return r.StatusCode == 401 && bytes.Index(r.Body, []byte(ErrAuth.Error())) >= 0 // TODO this is a weak check
}

// TODO have --raw feature to give back what splunk gives you, and also have expected struct here
func (c *Client) GetSearchStatus(searchID string) (Response, error) {
	// # check status of search
	// curl -H "Authorization: Splunk $SPLUNK_SESSION"  https://splunk.sendgrid.net:8089/services/search/jobs/$SEARCH_ID -d output_mode=json

	var ret Response

	urlstr := fmt.Sprintf("%s/services/search/jobs/%s", c.Addr, searchID)
	data := url.Values{}
	data.Set("output_mode", "json")
	req, err := http.NewRequest("GET", urlstr, strings.NewReader(data.Encode()))
	if err != nil {
		return ret, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", fmt.Sprintf("Splunk %s", c.SessionID))
	resp, err := c.httpcli.Do(req)
	if err != nil {
		return ret, err
	}
	ret.Body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return ret, err
	}
	ret.StatusCode = resp.StatusCode

	if ret.AuthFailed() {
		return ret, ErrAuth
	}

	return ret, nil
}

func (c *Client) ClearKnownSearches() error {
	var rm []string
	for sid, _ := range c.Searches {
		_, err := c.GetSearchStatus(sid)
		if err == ErrAuth {
			// dont clear if we can't communicate
			return ErrAuth
		}
		if err != nil {
			rm = append(rm, sid)
		}
	}
	for _, sid := range rm {
		delete(c.Searches, sid)
	}
	return nil
}

// TODO
// func (s *Splunk) GetSearchStatuses() ([]byte, error) {
// TODO

func submitRequest(req *http.Request) ([]byte, error) {
	cli := http.Client{}
	resp, err := cli.Do(req)

	if err != nil {
		return nil, err
	}

	byt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		err = fmt.Errorf("non-200 return code: %d, response: %s", resp.StatusCode, string(byt))
	}

	return byt, err
}
