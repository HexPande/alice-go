// Package assistant connects Alice's dialogue to OpenAI using PocketBase settings.
package assistant

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const Collection = "assistant_settings"

const DefaultPrompt = `Ты — голосовой помощник внутри навыка Яндекс Алисы.
Отвечай на русском языке, если пользователь не попросил другой язык.
Отвечай по существу, обычно в 1–3 коротких предложениях. Ответ должен быть не длиннее 900 символов.
Пиши обычным текстом без Markdown, таблиц, HTML и специальных тегов озвучивания.
Если вопрос неясен, задай один уточняющий вопрос. Не выдумывай факты и честно говори о неопределённости.
Не утверждай, что управляешь устройствами, выполняешь действия или имеешь доступ к данным, которых тебе не предоставили.`

const DefaultWelcome = "Привет! Задайте вопрос, и я постараюсь помочь."
const DefaultFallback = "Сейчас не получилось получить ответ. Попробуйте задать вопрос ещё раз."

type Settings struct {
	Enabled         bool
	APIKey          string
	Model           string
	SystemPrompt    string
	Welcome         string
	Fallback        string
	Timeout         time.Duration
	MaxOutputTokens int
}

// Load reads fresh values so changes made in the dashboard apply immediately.
func Load(app core.App) (Settings, error) {
	r, err := app.FindFirstRecordByData(Collection, "name", "default")
	if err != nil {
		return Settings{}, fmt.Errorf("load assistant settings: %w", err)
	}
	cfg := Settings{
		Enabled:         r.GetBool("enabled"),
		APIKey:          strings.TrimSpace(r.GetString("api_key")),
		Model:           strings.TrimSpace(r.GetString("model")),
		SystemPrompt:    r.GetString("system_prompt"),
		Welcome:         r.GetString("welcome_message"),
		Fallback:        r.GetString("error_message"),
		Timeout:         time.Duration(r.GetInt("timeout_ms")) * time.Millisecond,
		MaxOutputTokens: r.GetInt("max_output_tokens"),
	}
	if cfg.Timeout < 500*time.Millisecond || cfg.Timeout > 3500*time.Millisecond ||
		cfg.MaxOutputTokens < 16 || cfg.MaxOutputTokens > 1024 || cfg.Model == "" {
		return Settings{}, fmt.Errorf("invalid assistant settings")
	}
	return cfg, nil
}
