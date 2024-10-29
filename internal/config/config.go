package config

import (
	"fmt"
	"os"
)

const (
	defaultServerAddress = ":50051"
	defaultStorageDir    = "./files_storage"
)

type Config struct {
	ServerAddress string
	StorageDir    string
}

func (c *Config) String() string {
	return fmt.Sprintf("ServerAddress: %s, StorageDir: %s", c.ServerAddress, c.StorageDir)
}

func Load() *Config {
	cfg := &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", defaultServerAddress),
		StorageDir:    getEnv("STORAGE_DIR", defaultStorageDir),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Printf("Using default value for %s: %s\n", key, defaultValue)
		return defaultValue
	}
	fmt.Printf("Loaded %s from environment: %s\n", key, value)
	return value
}
