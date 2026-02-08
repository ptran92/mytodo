package quip

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
)

const (
	BASE_API_URL = "https://platform.quip.com"
)

type Client struct {
	accessToken  string
	clientId     string
	clientSecret string
	redirectUri  string
}
type Thread struct {
	ExpandedUserIds []string `mapstructure:"expanded_user_ids"`
	UserIds         []string `mapstructure:"user_ids"`
	SharedFolderIds []string `mapstructure:"shared_folder_ids"`
	Html            string
	Thread          map[string]string
}

type GetRecentThreadsParams struct {
	Count          int
	MaxUpdatedUsec int
}

type NewDocumentParams struct {
	Content   string
	Format    string
	Title     string
	MemberIds []string
}

type LocationEnum int

const (
	APPEND LocationEnum = iota
	PREPEND
	AFTER_SECTION
	BEFORE_SECTION
	REPLACE_SECTION
	DELETE_SECTION
	AFTER_DOCUMENT_RANGE
	BEFORE_DOCUMENT_RANGE
	REPLACE_DOCUMENT_RANGE
	DELETE_DOCUMENT_RANGE
)

type EditDocumentParams struct {
	ThreadId  string
	Content   string
	Format    string
	Location  LocationEnum
	SectionId string
}

type AddMembersParams struct {
	ThreadId  string
	MemberIds []string
}

type RemoveMembersParams struct {
	ThreadId  string
	MemberIds []string
}

func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
	}
}

func (q *Client) postJson(resource string, params map[string]interface{}) []byte {
	req, err := http.NewRequest("POST", resource, mapToQueryString(params))
	if err != nil {
		log.Fatal("Bad url: " + resource)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return q.doRequest(req)
}

func (q *Client) getJson(resource string, params map[string]interface{}) []byte {
	qs, err := ioutil.ReadAll(mapToQueryString(params))
	if err != nil {
		log.Fatal("Malformed query params %v", params)
	}

	queryString := string(qs)
	if queryString != "" {
		resource = resource + "?" + queryString
	}

	req, err := http.NewRequest("GET", resource, nil)
	if err != nil {
		log.Fatal("Bad url: " + resource)
	}

	return q.doRequest(req)
}

func (q *Client) doRequest(req *http.Request) []byte {
	client := &http.Client{}
	req.Header.Set("Authorization", "Bearer "+q.accessToken)
	res, err := client.Do(req)
	if err != nil {
		// TODO: handle API errors here
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	return body
}

// normalizeThreadID normalizes a thread ID by removing common prefixes
func normalizeThreadID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "https://quip.com/")
	id = strings.TrimPrefix(id, "http://quip.com/")
	return id
}

// hydrateThread converts a response interface into a Thread struct
func hydrateThread(resp interface{}) *Thread {
	var thread Thread
	mapstructure.Decode(resp, &thread)
	return &thread
}

// hydrateThreads converts a map of responses into Thread structs
func hydrateThreads(resp map[string]interface{}) []*Thread {
	threads := make([]*Thread, 0, len(resp))

	for _, body := range resp {
		threads = append(threads, hydrateThread(body))
	}

	return threads
}

func mapToQueryString(params map[string]interface{}) *strings.Reader {
	body := url.Values{}
	for k, v := range params {
		switch val := v.(type) {
		case string:
			body.Set(k, val)
		case int:
			body.Set(k, fmt.Sprintf("%d", val))
		case LocationEnum:
			body.Set(k, fmt.Sprintf("%d", val))
		case []string:
			body.Set(k, strings.Join(val, ","))
		default:
			body.Set(k, fmt.Sprintf("%v", val))
		}
	}
	return strings.NewReader(body.Encode())
}

func apiUrlResource(resource string) string {
	return BASE_API_URL + "/1/" + resource
}

func required(val interface{}, message string) {
	switch val := val.(type) {
	case string:
		if val == "" {
			log.Fatal(message)
		}
	case []string:
		if len(val) == 0 {
			log.Fatal(message)
		}
	}
}

func setOptional(val interface{}, key string, params *map[string]interface{}) {
	switch val := val.(type) {
	case string:
		if val != "" {
			(*params)[key] = val
		}
	case []string:
		if len(val) != 0 {
			(*params)[key] = val
		}
	case LocationEnum:
		(*params)[key] = val
	case int:
		(*params)[key] = val
	}
}

func setRequired(val interface{}, key string, params *map[string]interface{}, message string) {
	required(val, message)
	setOptional(val, key, params)
}

func parseJsonObject(b []byte) map[string]interface{} {
	var val map[string]interface{}
	json.Unmarshal(b, &val)
	return val
}

func parseJsonArray(b []byte) []interface{} {
	var val []interface{}
	json.Unmarshal(b, &val)
	return val
}
