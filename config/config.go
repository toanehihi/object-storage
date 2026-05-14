package config

import (
	"net/url"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   Server
	Postgres Postgres
	MinIO    MinIO
	NATS     NATS
	JWT      JWT
}

type Server struct {
	Port int `env:"SERVER_PORT" envDefault:"8080"`
}

type Postgres struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER" envDefault:"objectstorage"`
	Password string `env:"POSTGRES_PASSWORD" envDefault:"objectstorage"`
	DB       string `env:"POSTGRES_DB" envDefault:"objectstorage"`
	SSLMode  string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`
}

func (p Postgres) DSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.User, p.Password),
		Host:   p.Host + ":" + strconv.Itoa(p.Port),
		Path:   p.DB,
	}
	q := u.Query()
	q.Set("sslmode", p.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

type MinIO struct {
	Endpoint  string `env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	AccessKey string `env:"MINIO_ACCESS_KEY" envDefault:"minioadmin"`
	SecretKey string `env:"MINIO_SECRET_KEY" envDefault:"minioadmin"`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"uploads"`
	UseSSL    bool   `env:"MINIO_USE_SSL" envDefault:"false"`
}

type NATS struct {
	URL string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
}

type JWT struct {
	Secret string        `env:"JWT_SECRET" envDefault:"change-me-in-production"`
	Expiry time.Duration `env:"JWT_EXPIRY" envDefault:"24h"`
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
