package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env  string     `yaml:"env" env:"ENV" env-default:"local"`
	HTTP HTTPConfig `yaml:"http"`
	DB   DBConfig   `yaml:"db"`
	Log  LogConfig  `yaml:"log"`
}

type HTTPConfig struct {
	Host            string        `yaml:"host" env:"HTTP_HOST" env-default:"0.0.0.0"`
	Port            string        `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type DBConfig struct {
	Host            string        `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port            string        `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User            string        `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password        string        `yaml:"password" env:"DB_PASSWORD" env-default:"postgres"`
	Name            string        `yaml:"name" env:"DB_NAME" env-default:"subscriptions"`
	SSLMode         string        `yaml:"sslmode" env:"DB_SSLMODE" env-default:"disable"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" env-default:"10"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" env-default:"30m"`
	MigrationsPath  string        `yaml:"migrations_path" env:"DB_MIGRATIONS_PATH" env-default:"file://migrations"`
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type LogConfig struct {
	Level  string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
	Format string `yaml:"format" env:"LOG_FORMAT" env-default:"json"`
}

// Load reads config in the following order of precedence (highest wins):
//   1. environment variables
//   2. yaml file at CONFIG_PATH (or ./config.yaml as fallback)
//   3. defaults from struct tags
func Load() (*Config, error) {
	var cfg Config

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "./config.yaml"
	}

	if _, err := os.Stat(path); err == nil {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		// No yaml — fall back to env + defaults.
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("read env: %w", err)
		}
	} else {
		return nil, fmt.Errorf("stat config %s: %w", path, err)
	}

	return &cfg, nil
}
