package configs

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Address string `env:"HTTP_ADDRESS" env-default:"192.168.0.102:8080"`
}

type Config struct {
	Env    string `env:"ENV" env-default:"dev"`
	DBPath string `env:"DB_PATH" env-default:"postgresql/dev"`
	DBName string `env:"DB_NAME" env-default:"db.dev"`
	HTTPServer
	JWTSecretKey string `env:"JWT_SECRET_KEY" env-default:"sha25612864321684210"`
}

func LoadConfig() *Config {
	var cfg Config

	var envPath string

	flag.StringVar(&envPath, "config", "", "path to .env file")

	if envPath == "" {
		envPath = os.Getenv("CONFIG_PATH")
	}

	if envPath == "" {
		envPath = "config/dev.env"
	}

	err := cleanenv.ReadConfig(envPath, &cfg)
	if err != nil {
		log.Fatalf("Failed to read the .env properties from %s: %v", envPath, cfg)
	}

	return &cfg
}
