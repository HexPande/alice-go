package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase"
	pbcmd "github.com/pocketbase/pocketbase/cmd"
	"github.com/spf13/cobra"
)

func TestServeConfig(t *testing.T) {
	for _, tt := range []struct {
		name string
		http string
		want string
	}{
		{"yaml port", "", "0.0.0.0:9000"},
		{"explicit flag", "127.0.0.1:9001", "127.0.0.1:9001"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte("port: 9000\n"), 0600); err != nil {
				t.Fatal(err)
			}
			app := pocketbase.New()
			configure(app)
			serve := pbcmd.NewServeCommand(app, false)
			var got string
			serve.RunE = func(cmd *cobra.Command, _ []string) error {
				var err error
				got, err = cmd.Flags().GetString("http")
				return err
			}
			app.RootCmd.AddCommand(serve)
			args := []string{"serve", "--config", path}
			if tt.http != "" {
				args = append(args, "--http", tt.http)
			}
			app.RootCmd.SetArgs(args)
			if err := app.RootCmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("address = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigPort(t *testing.T) {
	app := pocketbase.New()
	configure(app)
	var out bytes.Buffer
	app.RootCmd.SetOut(&out)
	app.RootCmd.SetArgs([]string{"config-port"})
	if err := app.RootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "8090\n" {
		t.Fatalf("unexpected config-port output: %q", out.String())
	}
	if app.IsBootstrapped() {
		t.Fatal("config inspection must not bootstrap the database")
	}
}
