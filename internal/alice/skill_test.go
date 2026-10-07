package alice_test

import (
	"testing"

	"github.com/HexPande/alice-go/internal/alice"
)

func TestHandle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		text    string
		end     bool
	}{
		{"greeting normalization", "  ПРИВЕТ  ", "Привет! Рада вас слышать.", false},
		{"help", "помощь", "Я умею здороваться и завершать разговор. Скажите «привет» или «выход».", false},
		{"help alias", "что ты умеешь", "Я умею здороваться и завершать разговор. Скажите «привет» или «выход».", false},
		{"goodbye", "пока", "До встречи!", true},
		{"stop", "хватит", "До встречи!", true},
		{"unknown", "погода", "Пока я только учусь. Скажите «помощь», чтобы узнать, что я умею.", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := alice.Handle(alice.Request{Request: alice.Input{Type: "SimpleUtterance", Command: tt.command}})
			if got.Version != "1.0" || got.Response.Text != tt.text || got.Response.EndSession != tt.end {
				t.Errorf("unexpected response: %+v", got)
			}
		})
	}
}
