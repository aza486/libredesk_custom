package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zerodha/logf"
)

type OpenAIClient struct {
	apikey string
	lo     *logf.Logger
	client *http.Client
}

func NewOpenAIClient(apiKey string, lo *logf.Logger) *OpenAIClient {
	return &OpenAIClient{
		apikey: apiKey,
		lo:     lo,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendPrompt sends a prompt to the OpenAI API and returns the response text.
func (o *OpenAIClient) SendPrompt(payload PromptPayload) (string, error) {
	if o.apikey == "" {
		return "", fmt.Errorf("OpenAI API key is not set, Please ask your administrator to set the key")
	}

	apiURL := "https://api.openai.com/v1/chat/completions"
	requestBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": payload.SystemPrompt},
			{"role": "user", "content": payload.UserPrompt},
		},
		"max_tokens":  1024,
		"temperature": 0.7,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		o.lo.Error("error marshalling request body", "error", err)
		return "", fmt.Errorf("marshalling request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		o.lo.Error("error creating request", "error", err)
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apikey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		o.lo.Error("error making HTTP request", "error", err)
		return "", fmt.Errorf("making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("OpenAI API key is invalid, Please ask your administrator to update the key")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s, body: %s", resp.Status, body)
	}

	var responseBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("decoding response body: %w", err)
	}

	if len(responseBody.Choices) > 0 {
		return responseBody.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response found")
}
