package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HexPande/alice-go/internal/alice"
	"github.com/HexPande/alice-go/internal/server"
	_ "github.com/HexPande/alice-go/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestRoutes(t *testing.T) {
	t.Parallel()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	server.Register(app)
	r := router.NewRouter(func(w http.ResponseWriter, req *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
		e := &core.RequestEvent{App: app}
		e.Response = w
		e.Request = req
		return e, nil
	})
	if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: r}); err != nil {
		t.Fatal(err)
	}
	mux, err := r.BuildMux()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
		text   string
		end    bool
	}{
		{"health", "GET", "/healthz", "", 200, "", false},
		{"welcome", "POST", "/alice", `{"version":"1.0","session":{"new":true,"session_id":"test"},"request":{"type":"SimpleUtterance"},"meta":{"locale":"ru-RU"}}`, 200, "Привет! Это заготовка навыка Алисы на Go. Скажите «помощь» или «выход».", false},
		{"exit", "POST", "/alice", `{"version":"1.0","session":{"session_id":"test"},"request":{"type":"SimpleUtterance","command":"выход"}}`, 200, "До встречи!", true},
		{"button fallback", "POST", "/alice", `{"version":"1.0","session":{"session_id":"test"},"request":{"type":"ButtonPressed","payload":{"action":"unknown"}}}`, 200, "Пока я только учусь. Скажите «помощь», чтобы узнать, что я умею.", false},
		{"malformed JSON", "POST", "/alice", `{`, 400, "", false},
		{"empty body", "POST", "/alice", "", 400, "", false},
		{"missing fields", "POST", "/alice", `{}`, 400, "", false},
		{"null", "POST", "/alice", `null`, 400, "", false},
		{"unsupported version", "POST", "/alice", `{"version":"2.0","session":{"session_id":"test"},"request":{"type":"SimpleUtterance"}}`, 400, "", false},
		{"unsupported type", "POST", "/alice", `{"version":"1.0","session":{"session_id":"test"},"request":{"type":"Show.Pull"}}`, 400, "", false},
		{"oversized body", "POST", "/alice", strings.Repeat(" ", (1<<20)+1), 413, "", false},
		{"wrong method", "GET", "/alice", "", 404, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			if res.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", res.Code, tt.status, res.Body.String())
			}
			if tt.text == "" {
				return
			}
			if !strings.HasPrefix(res.Header().Get("Content-Type"), "application/json") {
				t.Errorf("unexpected content type: %s", res.Header().Get("Content-Type"))
			}
			var got alice.Response
			if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Version != "1.0" || got.Response.Text != tt.text || got.Response.EndSession != tt.end {
				t.Errorf("unexpected response: %+v", got)
			}
			// A false end_session must still be explicitly present in the wire format.
			var raw struct {
				Response map[string]json.RawMessage `json:"response"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &raw); err != nil {
				t.Fatal(err)
			}
			if _, ok := raw.Response["end_session"]; !ok {
				t.Error("missing required end_session field")
			}
		})
	}
}
