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

type StatusResponse struct {
	Paging struct {
		Offset  int `json:"offset"`
		PerPage int `json:"perPage"`
		Total   int `json:"total"`
	} `json:"paging"`
	Entry []struct {
		Acl struct {
			TTL        string `json:"ttl"`
			CanWrite   bool   `json:"can_write"`
			App        string `json:"app"`
			Sharing    string `json:"sharing"`
			Modifiable bool   `json:"modifiable"`
			Owner      string `json:"owner"`
			Perms      struct {
				Write []string `json:"write"`
				Read  []string `json:"read"`
			} `json:"perms"`
		} `json:"acl"`
		Content struct {
			RemoteSearchLogs                  []string               `json:"remoteSearchLogs"`
			IsTimeCursored                    bool                   `json:"isTimeCursored"`
			IsSavedSearch                     bool                   `json:"isSavedSearch"`
			IsSaved                           bool                   `json:"isSaved"`
			IsRemoteTimeline                  bool                   `json:"isRemoteTimeline"`
			IsRealTimeSearch                  bool                   `json:"isRealTimeSearch"`
			IsPreviewEnabled                  bool                   `json:"isPreviewEnabled"`
			IsPaused                          bool                   `json:"isPaused"`
			IsFinalized                       bool                   `json:"isFinalized"`
			IsFailed                          bool                   `json:"isFailed"`
			IsEventsPreviewEnabled            bool                   `json:"isEventsPreviewEnabled"`
			IsDone                            bool                   `json:"isDone"`
			IsBatchModeSearch                 bool                   `json:"isBatchModeSearch"`
			IndexLatestTime                   int                    `json:"indexLatestTime"`
			IndexEarliestTime                 int                    `json:"indexEarliestTime"`
			EventSorting                      string                 `json:"eventSorting"`
			EventSearch                       string                 `json:"eventSearch"`
			DispatchState                     string                 `json:"dispatchState"`
			DiskUsage                         int                    `json:"diskUsage"`
			Delegate                          string                 `json:"delegate"`
			DefaultTTL                        string                 `json:"defaultTTL"`
			DefaultSaveTTL                    string                 `json:"defaultSaveTTL"`
			CursorTime                        string                 `json:"cursorTime"`
			CanSummarize                      bool                   `json:"canSummarize"`
			BundleVersion                     string                 `json:"bundleVersion"`
			DoneProgress                      float64                `json:"doneProgress"`
			DropCount                         int                    `json:"dropCount"`
			EarliestTime                      string                 `json:"earliestTime"`
			EventAvailableCount               int                    `json:"eventAvailableCount"`
			EventCount                        int                    `json:"eventCount"`
			EventFieldCount                   int                    `json:"eventFieldCount"`
			EventIsStreaming                  bool                   `json:"eventIsStreaming"`
			EventIsTruncated                  bool                   `json:"eventIsTruncated"`
			IsZombie                          bool                   `json:"isZombie"`
			Keywords                          string                 `json:"keywords"`
			Label                             string                 `json:"label"`
			LatestTime                        string                 `json:"latestTime"`
			NormalizedSearch                  string                 `json:"normalizedSearch"`
			NumPreviews                       int                    `json:"numPreviews"`
			OptimizedSearch                   string                 `json:"optimizedSearch"`
			Pid                               string                 `json:"pid"`
			Priority                          int                    `json:"priority"`
			Provenance                        string                 `json:"provenance"`
			RemoteSearch                      string                 `json:"remoteSearch"`
			ReportSearch                      string                 `json:"reportSearch"`
			ResultCount                       int                    `json:"resultCount"`
			ResultIsStreaming                 bool                   `json:"resultIsStreaming"`
			ResultPreviewCount                int                    `json:"resultPreviewCount"`
			RunDuration                       float64                `json:"runDuration"`
			SampleRatio                       string                 `json:"sampleRatio"`
			SampleSeed                        string                 `json:"sampleSeed"`
			ScanCount                         int                    `json:"scanCount"`
			SearchCanBeEventType              bool                   `json:"searchCanBeEventType"`
			SearchEarliestTime                int                    `json:"searchEarliestTime"`
			SearchLatestTime                  float64                `json:"searchLatestTime"`
			SearchTotalBucketsCount           int                    `json:"searchTotalBucketsCount"`
			SearchTotalEliminatedBucketsCount int                    `json:"searchTotalEliminatedBucketsCount"`
			Sid                               string                 `json:"sid"`
			StatusBuckets                     int                    `json:"statusBuckets"`
			TTL                               int                    `json:"ttl"`
			Performance                       map[string]interface{} `json:"performance"`
			Messages                          []struct {
				Text string `json:"text"`
				Type string `json:"type"`
			} `json:"messages"`
			Request struct {
				Search string `json:"search"`
			} `json:"request"`
			Runtime struct {
				AutoPause  string `json:"auto_pause"`
				AutoCancel string `json:"auto_cancel"`
			} `json:"runtime"`
			SearchProviders []string `json:"searchProviders"`
		} `json:"content"`
		Author    string `json:"author"`
		Published string `json:"published"`
		Links     struct {
			Control        string `json:"control"`
			Summary        string `json:"summary"`
			Timeline       string `json:"timeline"`
			ResultsPreview string `json:"results_preview"`
			Results        string `json:"results"`
			Events         string `json:"events"`
			SearchLog      string `json:"search.log"`
			Alternate      string `json:"alternate"`
		} `json:"links"`
		Updated string `json:"updated"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	} `json:"entry"`
	Generator struct {
		Version string `json:"version"`
		Build   string `json:"build"`
	} `json:"generator"`
	Updated string `json:"updated"`
	Origin  string `json:"origin"`
	Links   struct {
	} `json:"links"`
}

type PastSearch struct {
	SearchID string
	Search   string
	// Date time executed
}

type Client struct {
	Username  string       `json:"username"`
	SessionID string       `json:"session_id"`
	Addr      string       `json:"addr"`
	Searches  []PastSearch `json:"searches"`
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
	ret := Client{}

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

func New(addr string) *Client { return &Client{Addr: addr} }

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
	c.Searches = append(c.Searches, PastSearch{
		SearchID: exp.SearchID,
		Search:   search,
	})
	ret.SearchID = exp.SearchID
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

	// TODO ./s status 4 | jq '.entry | .[] | .content.isDone'
	// .entry.[].content.isDone

	if ret.AuthFailed() {
		return ret, ErrAuth
	}

	return ret, nil
}

func (c *Client) ClearKnownSearches() error {
	var save []PastSearch
	for _, s := range c.Searches {
		r, err := c.GetSearchStatus(s.SearchID)
		if err == ErrAuth {
			// dont clear if we can't communicate
			return ErrAuth
		}
		if err != nil {
			return err
		}
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			save = append(save, s)
		}

	}
	c.Searches = save
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
