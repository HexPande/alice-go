package migrations

import (
	"github.com/HexPande/alice-go/internal/assistant"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection(assistant.Collection)
		// Nil API rules keep all operations restricted to PocketBase superusers.
		collection.Fields.Add(
			&core.TextField{Name: "name", Required: true, Pattern: "^default$", Presentable: true, Help: "Единственная запись настроек навыка. Оставьте default."},
			&core.BoolField{Name: "enabled", Help: "Включить ответы через OpenAI API."},
			&core.TextField{Name: "api_key", Max: 1024, Help: "Ключ OpenAI API. Доступен только администраторам. Не публикуйте его."},
			&core.TextField{Name: "model", Required: true, Max: 100, Help: "Имя модели OpenAI, например gpt-4.1-mini. Выбирайте быструю текстовую модель."},
			&core.TextField{Name: "system_prompt", Required: true, Max: 16000, Help: "Базовая инструкция: роль, стиль и правила ответов. Можно редактировать без перезапуска."},
			&core.TextField{Name: "welcome_message", Required: true, Max: 1024, Help: "Приветствие при запуске навыка без вопроса."},
			&core.TextField{Name: "error_message", Required: true, Max: 1024, Help: "Ответ при таймауте или ошибке OpenAI."},
			&core.NumberField{Name: "timeout_ms", Required: true, OnlyInt: true, Min: types.Pointer(500.0), Max: types.Pointer(3500.0), Help: "Таймаут OpenAI в миллисекундах. Оставляет время на доставку ответа Алисе."},
			&core.NumberField{Name: "max_output_tokens", Required: true, OnlyInt: true, Min: types.Pointer(16.0), Max: types.Pointer(1024.0), Help: "Лимит генерации. Короткие ответы быстрее; голосовой ответ ограничен 1024 символами."},
		)
		collection.AddIndex("idx_assistant_settings_name", true, "name", "")
		if err := app.Save(collection); err != nil {
			return err
		}
		record := core.NewRecord(collection)
		record.Set("name", "default")
		record.Set("enabled", false)
		record.Set("model", "gpt-4.1-mini")
		record.Set("system_prompt", assistant.DefaultPrompt)
		record.Set("welcome_message", assistant.DefaultWelcome)
		record.Set("error_message", assistant.DefaultFallback)
		record.Set("timeout_ms", 3000)
		record.Set("max_output_tokens", 256)
		return app.Save(record)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(assistant.Collection)
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
