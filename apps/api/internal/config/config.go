package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	CORSOrigin      string
	CookieSecure    bool
	AISessionTTL    time.Duration
	MaxBodyBytes    int64
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		DatabaseURL:     env("DATABASE_URL", "postgres://forma:forma@localhost:5432/forma?sslmode=disable"),
		CORSOrigin:      env("CORS_ORIGIN", "http://localhost:3000"),
		AISessionTTL:    30 * time.Minute,
		MaxBodyBytes:    2 << 20,
		ShutdownTimeout: 10 * time.Second,
	}

	var err error
	if value := os.Getenv("AI_SESSION_TTL"); value != "" {
		cfg.AISessionTTL, err = time.ParseDuration(value)
		if err != nil || cfg.AISessionTTL < time.Minute {
			return Config{}, fmt.Errorf("AI_SESSION_TTL must be a duration of at least 1m")
		}
	}
	if value := os.Getenv("MAX_BODY_BYTES"); value != "" {
		cfg.MaxBodyBytes, err = strconv.ParseInt(value, 10, 64)
		if err != nil || cfg.MaxBodyBytes < 1024 {
			return Config{}, fmt.Errorf("MAX_BODY_BYTES must be an integer of at least 1024")
		}
	}
	if value := os.Getenv("COOKIE_SECURE"); value != "" {
		cfg.CookieSecure, err = strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("COOKIE_SECURE must be true or false")
		}
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
