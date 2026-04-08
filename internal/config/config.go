package config

import (
	"errors"
	"os"
)

type Config struct {
	MongoURI  string
	MongoDB   string
	JWTSecret string
	HTTPAddr  string
}

func Load() (Config, error) {
	loadDotEnv()

	cfg := Config{
		MongoURI:  os.Getenv("MONGO_URI"),
		MongoDB:   os.Getenv("MONGO_DB"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		HTTPAddr:  os.Getenv("HTTP_ADDR"),
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.MongoURI == "" {
		return Config{}, errors.New("MONGO_URI is required")
	}
	if cfg.MongoDB == "" {
		return Config{}, errors.New("MONGO_DB is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	return cfg, nil
}
