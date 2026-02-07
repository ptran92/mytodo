package quip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	AccessToken string
	BaseURL     string
	HTTP        *http.Client
}

func NewClient(accessToken string) *Client {
	return &Client{
		AccessToken: accessToken,
		BaseURL:     "https://platform.quip.com/1",
		HTTP:        &http.Client{},
	}
}

// CreateDocument creates a new Quip document with the given content
func (c *Client) CreateDocument(title, content string) (string, error) {
	payload := map[string]interface{}{
		"title":   title,
		"content": content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/threads/new-document", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	threadID, ok := result["thread"].(map[string]interface{})["id"].(string)
	if !ok {
		return "", fmt.Errorf("failed to extract thread ID from response")
	}

	return fmt.Sprintf("https://quip.com/%s", threadID), nil
}

// UpdateDocument updates an existing Quip document
func (c *Client) UpdateDocument(threadID, content string) error {
	payload := map[string]interface{}{
		"content": content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/threads/edit-document", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	query := req.URL.Query()
	query.Add("thread_id", threadID)
	req.URL.RawQuery = query.Encode()

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	return nil
}

// AppendToDocument appends content to an existing Quip document
func (c *Client) AppendToDocument(threadID, content string) error {
	payload := map[string]interface{}{
		"content":   content,
		"operation": "append",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/threads/edit-document", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	query := req.URL.Query()
	query.Add("thread_id", threadID)
	req.URL.RawQuery = query.Encode()

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	return nil
}
