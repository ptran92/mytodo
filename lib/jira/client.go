package jira

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL   string
	AuthToken string
	HTTP      *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: baseURL, AuthToken: token, HTTP: &http.Client{}}
}

// generic GET request helper
func (c *Client) get(path string, query url.Values, result interface{}) error {
	u := c.BaseURL + path
	if query != nil {
		u += "?" + query.Encode()
	}
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(result)
}

// Search JQL
func (c *Client) Search(jql string) ([]map[string]interface{}, error) {
	var data struct {
		Issues []map[string]interface{} `json:"issues"`
	}
	err := c.get("/rest/api/3/search", url.Values{"jql": {jql}}, &data)
	return data.Issues, err
}

// Create an issue
func (c *Client) Create(summary, desc, issueType string, tags []string) (string, error) {
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project":     map[string]string{"key": "PROJ"},
			"summary":     summary,
			"description": desc,
			"issuetype":   map[string]string{"name": issueType},
		},
	}
	if len(tags) > 0 {
		payload["fields"].(map[string]interface{})["labels"] = tags
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.BaseURL+"/rest/api/3/issue", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res["key"].(string), nil
}
