package assistant_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/HexPande/alice-go/internal/alice"
	"github.com/HexPande/alice-go/internal/assistant"
)

func TestSkill(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, input, want                                             string
		newSession, disabled, missingKey, apiError, wantCall, wantEnd bool
	}{
		{name: "welcome", newSession: true, want: "Задайте вопрос."},
		{name: "question", input: "Почему небо голубое?", want: "Ответ модели.", wantCall: true},
		{name: "question on launch", input: "Расскажи о космосе", newSession: true, want: "Ответ модели.", wantCall: true},
		{name: "help from model", input: "помощь", want: "Ответ модели.", wantCall: true},
		{name: "exit", input: " Выход ", want: "До встречи!", wantEnd: true},
		{name: "disabled", input: "привет", disabled: true, want: "Привет! Рада вас слышать."},
		{name: "missing key", input: "Вопрос", missingKey: true, want: "Повторите вопрос."},
		{name: "network error", input: "Вопрос", apiError: true, wantCall: true, want: "Повторите вопрос."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := settings()
			cfg.Enabled = !tt.disabled
			if tt.missingKey {
				cfg.APIKey = ""
			}
			var logs bytes.Buffer
			called := false
			skill := assistant.Skill{
				LoadSettings: func() (assistant.Settings, error) { return cfg, nil },
				Logger:       slog.New(slog.NewTextHandler(&logs, nil)),
				Client: doerFunc(func(_ *http.Request) (*http.Response, error) {
					called = true
					if tt.apiError {
						return nil, errors.New("test-secret")
					}
					return response(`{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Ответ модели."}]}]}`), nil
				}),
			}
			got := skill.Handle(t.Context(), alice.Request{Session: alice.Session{New: tt.newSession}, Request: alice.Input{Type: "SimpleUtterance", Command: tt.input}})
			if got.Version != "1.0" || got.Response.Text != tt.want || got.Response.EndSession != tt.wantEnd || called != tt.wantCall {
				t.Fatalf("unexpected response: %+v, API called: %v", got, called)
			}
			if strings.Contains(logs.String(), "test-secret") {
				t.Fatal("logs leaked API credentials")
			}
		})
	}
}

func TestSkillReplyLimit(t *testing.T) {
	t.Parallel()
	skill := assistant.Skill{
		LoadSettings: func() (assistant.Settings, error) { return settings(), nil },
		Logger:       slog.Default(),
		Client: doerFunc(func(_ *http.Request) (*http.Response, error) {
			return response(`{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"` + strings.Repeat("Я", 1100) + `"}]}]}`), nil
		}),
	}
	got := skill.Handle(t.Context(), alice.Request{Request: alice.Input{Type: "SimpleUtterance", Command: "Вопрос"}})
	if !utf8.ValidString(got.Response.Text) || utf8.RuneCountInString(got.Response.Text) != 1024 || !strings.HasSuffix(got.Response.Text, "…") {
		t.Fatal("response must fit Alice's 1024-character limit without breaking UTF-8")
	}
}
