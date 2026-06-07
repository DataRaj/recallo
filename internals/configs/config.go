package configs

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Address string `env:"HTTP_ADDRESS" env-default:"0.0.0.0:8080"`
}

type Config struct {
	Env         string `env:"ENV"          env-default:"dev"`
	DatabaseURL string `env:"DATABASE_URL" env-default:"postgres://postgres:password@localhost:5432/gotel?sslmode=disable"`
	HTTPServer
	JWTSecretKey string `env:"JWT_SECRET_KEY" env-default:""`
}

func LoadConfig() *Config {
	var cfg Config

	var envPath string

	flag.StringVar(&envPath, "config", "", "path to .env file")
	flag.Parse()

	if envPath == "" {
		envPath = os.Getenv("CONFIG_PATH")
	}

	if envPath == "" {
		envPath = "config/dev.env"
	}

	err := cleanenv.ReadConfig(envPath, &cfg)
	if err != nil {
		log.Fatalf("Failed to read config from %s: %v", envPath, err)
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL must not be empty")
	}

	if cfg.JWTSecretKey == "" {
		log.Fatal("JWT_SECRET_KEY must not be empty")
	}

	return &cfg
}
