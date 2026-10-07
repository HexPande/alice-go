// Package config reads the application's YAML settings.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Port int
}

// Load uses defaults when path is empty; an explicitly selected file must exist.
func Load(path string) (Config, error) {
	v := viper.New()
	v.SetDefault("port", 8090)
	if path != "" {
		v.SetConfigFile(path)
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}
	port, ok := v.Get("port").(int)
	if !ok || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("config port must be an integer between 1 and 65535")
	}
	return Config{Port: port}, nil
}
