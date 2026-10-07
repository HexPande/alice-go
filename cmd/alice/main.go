package main

import (
	"log"
	"os"

	"github.com/HexPande/alice-go/internal/server"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"
)

func main() {
	app := pocketbase.New()
	configure(app)
	// Configuration inspection must not bootstrap or migrate a database.
	if len(os.Args) > 1 && os.Args[1] == "config-port" {
		if err := app.RootCmd.Execute(); err != nil {
			log.Fatal(err)
		}
		return
	}
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})
	server.Register(app)
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
