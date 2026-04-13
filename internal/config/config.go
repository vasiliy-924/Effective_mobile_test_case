package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// File holds optional non-secret defaults loaded from YAML.
type File struct {
	HTTPAddr   string `yaml:"http_addr"`
	LogLevel   string `yaml:"log_level"`
	LogFormat  string `yaml:"log_format"`
	Migrations string `yaml:"migrations_path"`
}

// Config is the runtime configuration (env overrides file).
type Config struct {
	HTTPAddr    string        `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL string        `env:"DATABASE_URL,required"`
	LogLevel    string        `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat   string        `env:"LOG_FORMAT" envDefault:"text"` // text | json
	Migrations  string        `env:"MIGRATIONS_PATH" envDefault:"file://migrations"`
	DBRetry     int           `env:"DB_CONNECT_RETRIES" envDefault:"30"`
	DBRetryWait time.Duration `env:"DB_CONNECT_RETRY_WAIT" envDefault:"1s"`
}

// Load reads optional .env (non-fatal), optional YAML file, then environment variables.
func Load(yamlPath string) (*Config, error) {
	_ = godotenv.Load()

	fileDefaults := File{
		HTTPAddr:   ":8080",
		LogLevel:   "info",
		LogFormat:  "text",
		Migrations: "file://migrations",
	}
	if yamlPath == "" {
		// skip optional YAML
	} else if raw, err := os.ReadFile(yamlPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config yaml: %w", err)
		}
	} else if err := yaml.Unmarshal(raw, &fileDefaults); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	cfg := Config{
		HTTPAddr:    fileDefaults.HTTPAddr,
		LogLevel:    fileDefaults.LogLevel,
		LogFormat:   fileDefaults.LogFormat,
		Migrations:  fileDefaults.Migrations,
		DBRetry:     30,
		DBRetryWait: time.Second,
	}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	return &cfg, nil
}
