// Package server registers the skill's HTTP routes in PocketBase.
package server

import (
	"net/http"

	"github.com/HexPande/alice-go/internal/alice"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Register attaches public webhook routes to the application's serving hook.
func Register(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/healthz", func(e *core.RequestEvent) error {
			return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
		e.Router.POST("/alice", webhook).Bind(apis.BodyLimit(1 << 20))
		return e.Next()
	})
}

func webhook(e *core.RequestEvent) error {
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
	return e.JSON(http.StatusOK, alice.Handle(req))
}
