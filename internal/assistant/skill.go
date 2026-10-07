package assistant

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/HexPande/alice-go/internal/alice"
)

type Skill struct {
	LoadSettings func() (Settings, error)
	Client       HTTPDoer
	Logger       *slog.Logger
}

// Handle sends spoken questions to OpenAI while keeping local exit commands available.
func (s *Skill) Handle(ctx context.Context, req alice.Request) alice.Response {
	input := strings.TrimSpace(req.Request.Command)
	if req.Request.Type == "SimpleUtterance" {
		switch strings.ToLower(input) {
		case "выход", "пока", "хватит":
			return alice.Response{Version: "1.0", Response: alice.Reply{Text: "До встречи!", EndSession: true}}
		}
	}
	cfg, err := s.LoadSettings()
	if err != nil {
		s.Logger.Error("assistant settings unavailable")
		return reply(DefaultFallback)
	}
	if !cfg.Enabled {
		return alice.Handle(req)
	}
	if req.Request.Type != "SimpleUtterance" || input == "" {
		return reply(cfg.Welcome)
	}
	if cfg.APIKey == "" {
		s.Logger.Warn("OpenAI API key is not configured")
		return reply(cfg.Fallback)
	}
	if utf8.RuneCountInString(input) > 4096 {
		return reply("Пожалуйста, задайте вопрос короче.")
	}
	text, err := Generate(ctx, s.Client, cfg, input)
	if err != nil {
		s.Logger.Warn("assistant request failed", "error", err)
		return reply(cfg.Fallback)
	}
	return reply(text)
}

func reply(text string) alice.Response {
	text = strings.TrimSpace(text)
	if text == "" {
		text = DefaultFallback
	}
	chars := []rune(text)
	if len(chars) > 1024 {
		text = string(chars[:1023]) + "…"
	}
	return alice.Response{Version: "1.0", Response: alice.Reply{Text: text}}
}
