package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Server serverConfig
	DB     databaseConfig
	OIDC   oidcConfig
}

type serverConfig struct {
	Port           int      `env:"SERVER_PORT" envDefault:"8080"`
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envDefault:"http://localhost:3000" envSeparator:","`
}

type oidcConfig struct {
	ClientID string `env:"OIDC_CLIENT_ID,required"`
	Issuer   string `env:"OIDC_ISSUER" envDefault:"https://login.maxstash.io/application/o/olympics/"`
}

type databaseConfig struct {
	Host     string `env:"DB_HOST,required"`
	Port     int    `env:"DB_PORT,required"`
	User     string `env:"DB_USER,required"`
	Password string `env:"DB_PASSWORD,required"`
	Name     string `env:"DB_NAME,required"`
	SSLMode  string `env:"DB_SSL_MODE,required"`
}

func LoadEnv() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return cfg, nil
}
