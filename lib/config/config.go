package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Logger   LoggerConfig   `yaml:"logger"`
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbName"`
	SSLMode  string `yaml:"sslMode"`
	MaxConns int32  `yaml:"maxConns"`
}

// LoggerConfig holds logging configuration.
type LoggerConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

var globalConfig atomic.Pointer[Config]

// Get returns the current global Config. May return nil if not yet set.
func Get() *Config {
	return globalConfig.Load()
}

// Set atomically replaces the global Config and triggers registered callbacks.
func Set(cfg *Config) {
	globalConfig.Store(cfg)
	callbacksMu.RLock()
	cbs := make([]func(*Config), len(callbacks))
	copy(cbs, callbacks)
	callbacksMu.RUnlock()
	for _, fn := range cbs {
		fn(cfg)
	}
}

var (
	callbacks   []func(*Config)
	callbacksMu sync.RWMutex
)

// RegisterReloadCallback registers a function to be called when configuration is reloaded.
func RegisterReloadCallback(fn func(*Config)) {
	callbacksMu.Lock()
	callbacks = append(callbacks, fn)
	callbacksMu.Unlock()
}

// SetDefaults fills zero-value fields with sensible defaults.
func SetDefaults(cfg *Config) {
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.User == "" {
		cfg.Database.User = "lcp"
	}
	if cfg.Database.Password == "" {
		cfg.Database.Password = "lcp"
	}
	if cfg.Database.DBName == "" {
		cfg.Database.DBName = "lcp"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.MaxConns == 0 {
		cfg.Database.MaxConns = 10
	}
	if cfg.Logger.Level == "" {
		cfg.Logger.Level = "INFO"
	}
	if cfg.Logger.Format == "" {
		cfg.Logger.Format = "default"
	}
}

// LoadFromFile reads and parses a YAML configuration file.
// If the file does not exist, an empty Config with defaults is returned.
func LoadFromFile(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			SetDefaults(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file %q: %w", path, err)
	}
	SetDefaults(cfg)
	return cfg, nil
}

// ApplyEnvOverrides overrides Config fields with environment variable values when set.
func ApplyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = i
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.DBName = v
	}
	if v := os.Getenv("DB_SSL_MODE"); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxConns = int32(i)
		}
	}
}
