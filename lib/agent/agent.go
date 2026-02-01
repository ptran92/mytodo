package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mytodo/lib/utils"
	"net/http"
	"strings"
)

// -----------------------------------------------------------------------------
// LlmResponse
// -----------------------------------------------------------------------------
type LlmResponse struct {
	raw json.RawMessage // raw JSON response from the LLM server
}

type InternalResp struct {
	Response string `json:"response"`
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
	var data InternalResp
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

// ---------------------------------------------------------------------------
// OpenAIAgent implementation
// ---------------------------------------------------------------------------

type OpenAIAgent struct {
	client  NetClient
	baseURL string
	apiKey  string
}

// From official OpenAI document
type OpenAIResponse struct {
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (oa *OpenAIAgent) Prompt(prompt string) (*LlmResponse, error) {
	// Build request payload
	payload := map[string]interface{}{
		"model":             "gpt-4.1",
		"input":             prompt,
		"max_output_tokens": 512,
		"temperature":       0.7,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", oa.baseURL+"/v1/responses", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+oa.apiKey)

	resp, err := oa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non‑OK HTTP status: %s, body: %s", resp.Status, string(respBody))
	}

	return extractOpenAiResponse(respBody)
}

func extractOpenAiResponse(openAiResponse []byte) (*LlmResponse, error) {
	if openAiResponse == nil {
		return nil, fmt.Errorf("Input is empty")
	}

	var openAiResp OpenAIResponse
	if err := json.Unmarshal(openAiResponse, &openAiResp); err != nil {
		return nil, err
	}

	if len(openAiResp.Output) == 0 || len(openAiResp.Output[0].Content) == 0 {
		return nil, fmt.Errorf("no output text in response: %s", string(openAiResponse))
	}

	internalResp := InternalResp{Response: openAiResp.Output[0].Content[0].Text}
	internalRespRaw, _ := json.Marshal(internalResp)

	return &LlmResponse{raw: internalRespRaw}, nil
}

// ---------------------------------------------------------------------------
// Factory functions
// ---------------------------------------------------------------------------

func CreateLlmAgent(client NetClient) LlmAgent {
	return &OllamaAgent{
		client:  client,
		baseURL: "http://localhost:11434",
	}
}

func CreateLlmAgentDefault() LlmAgent {
	httpClient := &http.Client{}
	return CreateLlmAgent(httpClient)
}

// ---------------------------------------------------------------------------
// OpenAI specific factory helpers
// ---------------------------------------------------------------------------

func CreateOpenAIAgent(client NetClient, apiKey string) LlmAgent {
	return &OpenAIAgent{
		client:  client,
		baseURL: "https://api.openai.com",
		apiKey:  apiKey,
	}
}

func CreateOpenAIAgentDefault() (LlmAgent, error) {
	apiKey := utils.GetOpenAIToken()
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPEN_AI_API_KEY not set")
	}
	httpClient := &http.Client{}
	return CreateOpenAIAgent(httpClient, apiKey), nil
}
