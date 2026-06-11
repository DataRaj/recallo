package configs

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// HTTPServer holds HTTP server binding configuration.
type HTTPServer struct {
	Address string `env:"HTTP_ADDRESS" env-default:"0.0.0.0:8080"`
}

// Config is the full application configuration loaded from an .env file.
type Config struct {
	Env          string `env:"ENV"            env-default:"dev"`
	DatabaseURL  string `env:"DATABASE_URL"   env-default:""`
	JWTSecretKey string `env:"JWT_SECRET_KEY" env-default:""`
	HTTPServer
}

// LoadConfig resolves the config file path in this order:
//  1. -config <path> CLI flag
//  2. CONFIG_PATH environment variable
//  3. Default: config/dev.env
func LoadConfig() *Config {
	var cfg Config
	var envPath string

	flag.StringVar(&envPath, "config", "", "path to .env config file")
	flag.Parse()

	if envPath == "" {
		envPath = os.Getenv("CONFIG_PATH")
	}
	if envPath == "" {
		envPath = "config/dev.env"
	}

	if err := cleanenv.ReadConfig(envPath, &cfg); err != nil {
		log.Fatalf("[config] failed to read config from %s: %v", envPath, err)
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("[config] DATABASE_URL must not be empty")
	}
	if cfg.JWTSecretKey == "" {
		log.Fatal("[config] JWT_SECRET_KEY must not be empty")
	}

	log.Printf("[config] loaded env=%s addr=%s", cfg.Env, cfg.Address)
	return &cfg
}
