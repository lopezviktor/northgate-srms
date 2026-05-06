package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPort   = "8080"
	DefaultDBPath = "northgate.db"
)

type Config struct {
	Port   string
	DBPath string
}

func Load() (Config, error) {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = DefaultPort
	}

	if err := validatePort(port); err != nil {
		return Config{}, fmt.Errorf("invalid PORT: %w", err)
	}

	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	if err := validateDBPath(dbPath); err != nil {
		return Config{}, fmt.Errorf("invalid DB_PATH: %w", err)
	}

	return Config{
		Port:   port,
		DBPath: dbPath,
	}, nil
}

func (c Config) ServerAddress() string {
	return ":" + c.Port
}

func validatePort(port string) error {
	value, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("must be numeric")
	}

	if value < 1024 || value > 65535 {
		return fmt.Errorf("must be between 1024 and 65535")
	}

	return nil
}

func validateDBPath(dbPath string) error {
	if dbPath == "" {
		return fmt.Errorf("must not be empty")
	}

	if strings.Contains(dbPath, "\x00") {
		return fmt.Errorf("must not contain null bytes")
	}

	if strings.HasPrefix(dbPath, "/etc/") ||
		strings.HasPrefix(dbPath, "/bin/") ||
		strings.HasPrefix(dbPath, "/usr/") ||
		strings.HasPrefix(dbPath, "/System/") {
		return fmt.Errorf("must not point to a sensitive system directory")
	}

	return nil
}
