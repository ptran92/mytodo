package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// -----------------------------------------------------------------------------
// LlmResponse
// -----------------------------------------------------------------------------
type LlmResponse struct {
	raw json.RawMessage // raw JSON response from the LLM server
}

// String returns a pretty‑printed string representation of the response.
func (l *LlmResponse) String() (string, error) {
	if l.raw == nil {
		return "", fmt.Errorf("no data")
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, l.raw, "", "  "); err != nil {
		return "", err
	}
	return pretty.String(), nil
}

// Json returns the raw JSON payload.
func (l *LlmResponse) Json() (json.RawMessage, error) {
	if l.raw == nil {
		return nil, fmt.Errorf("no data")
	}
	return l.raw, nil
}

func (l *LlmResponse) GetResponse() (string, error) {
	// Parse the JSON into the struct.
	type responseData struct {
		Response string `json:"response"`
	}

	var data responseData
	if err := json.Unmarshal([]byte(l.raw), &data); err != nil {
		// If the JSON itself is malformed, fall back to the default.
		return "", err
	}

	// If the field was absent, data.Response will still be the zero value.
	if data.Response == "" {
		data.Response = "No response"
	}

	return strings.Trim(strings.TrimSpace(data.Response), "```"), nil

}

// -----------------------------------------------------------------------------
// LlmAgent interface
// -----------------------------------------------------------------------------
type LlmAgent interface {
	Prompt(prompt string) (*LlmResponse, error)
}

// -----------------------------------------------------------------------------
// NetClient interface
// -----------------------------------------------------------------------------
type NetClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// -----------------------------------------------------------------------------
// OllamaAgent implementation
// -----------------------------------------------------------------------------
type OllamaAgent struct {
	client  NetClient
	baseURL string
}

func (oa *OllamaAgent) Prompt(prompt string) (*LlmResponse, error) {
	// Build request payload
	payload := map[string]interface{}{
		"model":  "gpt-oss:20b",
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", oa.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := oa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non‑OK HTTP status: %s, body: %s", resp.Status, string(respBody))
	}

	// Wrap into LlmResponse
	return &LlmResponse{raw: respBody}, nil
}

// -----------------------------------------------------------------------------
// Factory function
// -----------------------------------------------------------------------------
func CreateLlmAgent(client NetClient) LlmAgent {
	return &OllamaAgent{
		client:  client,
		baseURL: "http://localhost:11434", // Ollama default local URL
	}
}

func CreateLlmAgentDefault() LlmAgent {
	// Use the standard http client; you can replace it with a custom implementation.
	httpClient := &http.Client{}
	return CreateLlmAgent(httpClient)
}
