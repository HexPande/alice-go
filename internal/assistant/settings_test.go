package assistant_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HexPande/alice-go/internal/assistant"
	_ "github.com/HexPande/alice-go/migrations"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestSettingsMigrationAndAccess(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	cfg, err := assistant.Load(app)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || cfg.APIKey != "" || cfg.SystemPrompt != assistant.DefaultPrompt || cfg.Model == "" {
		t.Fatalf("unexpected seeded defaults: %+v", cfg)
	}
	record, err := app.FindFirstRecordByData(assistant.Collection, "name", "default")
	if err != nil {
		t.Fatal(err)
	}
	record.Set("api_key", "test-secret")
	record.Set("system_prompt", "Ты — помощник по программированию.")
	record.Set("enabled", true)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	cfg, err = assistant.Load(app)
	if err != nil || cfg.SystemPrompt != "Ты — помощник по программированию." || !cfg.Enabled {
		t.Fatalf("settings changes were not preserved: %+v, %v", cfg, err)
	}
	collection, err := app.FindCollectionByNameOrId(assistant.Collection)
	if err != nil {
		t.Fatal(err)
	}
	if collection.ListRule != nil || collection.ViewRule != nil || collection.CreateRule != nil || collection.UpdateRule != nil || collection.DeleteRule != nil {
		t.Fatal("settings API must be superuser-only")
	}
	duplicate := core.NewRecord(collection)
	for key, value := range record.FieldsData() {
		if key != "id" {
			duplicate.Set(key, value)
		}
	}
	if err := app.Save(duplicate); err == nil {
		t.Fatal("duplicate settings record accepted")
	}
	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := r.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/collections/assistant_settings/records", "/api/collections/assistant_settings/records/" + record.Id} {
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusForbidden {
			t.Fatalf("anonymous access returned %d: %s", res.Code, res.Body.String())
		}
	}
	record.Set("timeout_ms", 4500)
	if err := app.Save(record); err == nil {
		t.Fatal("timeout exceeding Alice's budget accepted")
	}
}
