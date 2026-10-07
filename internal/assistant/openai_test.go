package assistant_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/HexPande/alice-go/internal/assistant"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func settings() assistant.Settings {
	return assistant.Settings{
		Enabled: true, APIKey: "test-secret", Model: "gpt-4.1-mini", SystemPrompt: "Говори кратко.",
		Welcome: "Задайте вопрос.", Fallback: "Повторите вопрос.", Timeout: time.Second, MaxOutputTokens: 256,
	}
}

func response(text string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(text))}
}

func TestGenerateContract(t *testing.T) {
	t.Parallel()
	client := doerFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.openai.com/v1/responses" || r.Method != http.MethodPost {
			t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-secret" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing authentication or JSON header")
		}
		var body struct {
			Model           string `json:"model"`
			Input           string `json:"input"`
			Instructions    string `json:"instructions"`
			Store           *bool  `json:"store"`
			MaxOutputTokens int    `json:"max_output_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "gpt-4.1-mini" || body.Input != "Что такое Go?" || body.Instructions != "Говори кратко." ||
			body.Store == nil || *body.Store || body.MaxOutputTokens != 256 {
			t.Errorf("unexpected request: %+v", body)
		}
		return response(`{"status":"completed","output":[{"type":"reasoning"},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Go — язык программирования."}]}]}`), nil
	})
	got, err := assistant.Generate(t.Context(), client, settings(), "Что такое Go?")
	if err != nil || got != "Go — язык программирования." {
		t.Fatalf("Generate = %q, %v", got, err)
	}
}

func TestGenerateResponses(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body, want string
		status           int
		fail             bool
	}{
		{"invalid key", `{"error":{"message":"test-secret"}}`, "", 401, true},
		{"rate limit", `{}`, "", 429, true},
		{"upstream error", `{}`, "", 503, true},
		{"invalid JSON", `{`, "", 200, true},
		{"empty output", `{"status":"completed","output":[]}`, "", 200, true},
		{"failed response", `{"status":"failed"}`, "", 200, true},
		{"oversized body", strings.Repeat("x", (1<<20)+1), "", 200, true},
		{"refusal", `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"refusal","refusal":"Не могу помочь с этим."}]}]}`, "Не могу помочь с этим.", 200, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := doerFunc(func(_ *http.Request) (*http.Response, error) {
				r := response(tt.body)
				r.StatusCode = tt.status
				return r, nil
			})
			got, err := assistant.Generate(t.Context(), client, settings(), "Вопрос")
			if (err != nil) != tt.fail || got != tt.want {
				t.Fatalf("Generate = %q, %v", got, err)
			}
			if err != nil && strings.Contains(err.Error(), "test-secret") {
				t.Fatal("error leaked sensitive upstream data")
			}
		})
	}
}

func TestGenerateTimeout(t *testing.T) {
	t.Parallel()
	cfg := settings()
	cfg.Timeout = time.Millisecond
	client := doerFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	_, err := assistant.Generate(t.Context(), client, cfg, "Вопрос")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
}
