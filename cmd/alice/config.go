package main

import (
	"fmt"

	"github.com/HexPande/alice-go/internal/config"
	"github.com/pocketbase/pocketbase"
	"github.com/spf13/cobra"
)

func configure(app *pocketbase.PocketBase) {
	var path string
	app.RootCmd.PersistentFlags().StringVar(&path, "config", "", "Path to the YAML configuration file")
	app.RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Name() != "serve" {
			return nil
		}
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}
		// Preserve explicit PocketBase flags and domain-based automatic TLS.
		if !cmd.Flags().Changed("http") && len(args) == 0 {
			return cmd.Flags().Set("http", fmt.Sprintf("0.0.0.0:%d", cfg.Port))
		}
		return nil
	}
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "config-port",
		Short: "Validate configuration and print the HTTP port",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), cfg.Port)
			return err
		},
	})
}
