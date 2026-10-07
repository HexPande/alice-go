package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPDoer allows tests to exercise the API contract without paid requests.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Generate calls the Responses API once; retries would exceed Alice's time budget.
func Generate(ctx context.Context, client HTTPDoer, cfg Settings, input string) (string, error) {
	body, err := json.Marshal(struct {
		Model           string `json:"model"`
		Instructions    string `json:"instructions"`
		Input           string `json:"input"`
		MaxOutputTokens int    `json:"max_output_tokens"`
		Store           bool   `json:"store"`
	}{cfg.Model, cfg.SystemPrompt, input, cfg.MaxOutputTokens, false})
	if err != nil {
		return "", errors.New("encode OpenAI request")
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return "", errors.New("create OpenAI request")
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("OpenAI request canceled: %w", ctx.Err())
		}
		// Transport errors may contain sensitive details; do not expose the original error.
		return "", errors.New("OpenAI connection failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI returned HTTP %d", resp.StatusCode)
	}
	const maxBody = 1 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil || len(data) > maxBody {
		return "", errors.New("cannot read OpenAI response")
	}
	var response struct {
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", errors.New("invalid OpenAI response")
	}
	if response.Status != "completed" && response.Status != "incomplete" {
		return "", errors.New("OpenAI did not complete the response")
	}
	var parts []string
	for _, item := range response.Output {
		if item.Type != "message" || item.Role != "assistant" {
			continue
		}
		for _, content := range item.Content {
			switch content.Type {
			case "output_text":
				parts = append(parts, content.Text)
			case "refusal":
				parts = append(parts, content.Refusal)
			}
		}
	}
	text := strings.TrimSpace(strings.Join(parts, "\n"))
	if text == "" {
		return "", errors.New("OpenAI returned no text")
	}
	return text, nil
}
