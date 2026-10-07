// Package server registers the skill's HTTP routes in PocketBase.
package server

import (
	"net/http"
	"time"

	"github.com/HexPande/alice-go/internal/alice"
	"github.com/HexPande/alice-go/internal/assistant"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Register attaches public webhook routes to the application's serving hook.
func Register(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		skill := &assistant.Skill{
			LoadSettings: func() (assistant.Settings, error) { return assistant.Load(app) },
			Client: &http.Client{
				Timeout:       3500 * time.Millisecond,
				CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
			},
			Logger: app.Logger(),
		}
		e.Router.GET("/healthz", func(e *core.RequestEvent) error {
			return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
		e.Router.POST("/alice", func(e *core.RequestEvent) error { return webhook(e, skill) }).Bind(apis.BodyLimit(1 << 20))
		return e.Next()
	})
}

func webhook(e *core.RequestEvent, skill *assistant.Skill) error {
	var req alice.Request
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body.", err)
	}
	if req.Version != "1.0" || req.Session.SessionID == "" || req.Request.Type == "" {
		return e.BadRequestError("Version 1.0, session.session_id and request.type are required.", nil)
	}
	if req.Request.Type != "SimpleUtterance" && req.Request.Type != "ButtonPressed" {
		return e.BadRequestError("Unsupported request type.", nil)
	}
	return e.JSON(http.StatusOK, skill.Handle(e.Request.Context(), req))
}
