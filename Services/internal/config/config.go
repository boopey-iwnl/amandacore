package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type ServiceConfig struct {
	ServiceName       string
	Port              string
	Environment       string
	StorePath         string
	LocalSeedFile     string
	AdminSeedUsername string
	AdminSeedPassword string
	BuildID           string
	WorldEndpoint     string
}

func Load(serviceName string, defaultPort string) ServiceConfig {
	localSeedFile := valueOrDefault("AMANDACORE_LOCAL_SEED_FILE", filepath.Clean(".secrets/amandacore.dev.env"))
	loadEnvFileIfPresent(localSeedFile)

	return ServiceConfig{
		ServiceName:       serviceName,
		Port:              valueOrDefault("AMANDACORE_SERVICE_PORT", defaultPort),
		Environment:       valueOrDefault("AMANDACORE_ENVIRONMENT", "development"),
		StorePath:         valueOrDefault("AMANDACORE_STORE_PATH", defaultStorePath()),
		LocalSeedFile:     localSeedFile,
		AdminSeedUsername: valueOrDefault("AMANDACORE_ADMIN_SEED_USERNAME", "amanda"),
		AdminSeedPassword: os.Getenv("AMANDACORE_ADMIN_SEED_PASSWORD"),
		BuildID:           valueOrDefault("AMANDACORE_BUILD_ID", "amandacore-local-0.2.0"),
		WorldEndpoint:     valueOrDefault("AMANDACORE_WORLD_ENDPOINT", "http://localhost:8085"),
	}
}

func loadEnvFileIfPresent(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

func valueOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func defaultStorePath() string {
	configRoot, err := os.UserConfigDir()
	if err != nil || configRoot == "" {
		return filepath.Clean(filepath.Join(os.TempDir(), "amandacore", "platform-state.json"))
	}

	return filepath.Join(configRoot, "amandacore", "platform-state.json")
}
