package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ServiceConfig struct {
	ServiceName       string
	Host              string
	Port              string
	Environment       string
	StoreBackend      string
	StorePath         string
	SQLitePath        string
	LocalSeedFile     string
	AdminSeedUsername string
	AdminSeedPassword string
	AdminToolsEnabled bool
	BuildID           string
	WorldEndpoint     string
}

func Load(serviceName string, defaultPort string) ServiceConfig {
	environment := valueOrDefault("AMANDACORE_ENVIRONMENT", "development")
	localSeedFile := valueOrDefault("AMANDACORE_LOCAL_SEED_FILE", filepath.Clean(".secrets/amandacore.dev.env"))
	if !isProductionEnvironment(environment) {
		loadEnvFileIfPresent(localSeedFile)
		environment = valueOrDefault("AMANDACORE_ENVIRONMENT", environment)
	}

	return ServiceConfig{
		ServiceName:       serviceName,
		Host:              valueOrDefault("AMANDACORE_SERVICE_HOST", "127.0.0.1"),
		Port:              valueOrDefault("AMANDACORE_SERVICE_PORT", defaultPort),
		Environment:       environment,
		StoreBackend:      valueOrDefault("AMANDACORE_STORE_BACKEND", "file"),
		StorePath:         valueOrDefault("AMANDACORE_STORE_PATH", defaultStorePath()),
		SQLitePath:        os.Getenv("AMANDACORE_SQLITE_PATH"),
		LocalSeedFile:     localSeedFile,
		AdminSeedUsername: valueOrDefault("AMANDACORE_ADMIN_SEED_USERNAME", "amanda"),
		AdminSeedPassword: os.Getenv("AMANDACORE_ADMIN_SEED_PASSWORD"),
		AdminToolsEnabled: adminToolsEnabled(environment),
		BuildID:           valueOrDefault("AMANDACORE_BUILD_ID", "amandacore-alpha-0.1-local"),
		WorldEndpoint:     valueOrDefault("AMANDACORE_WORLD_ENDPOINT", "http://127.0.0.1:8085"),
	}
}

func LoadValidated(serviceName string, defaultPort string) (ServiceConfig, error) {
	cfg := Load(serviceName, defaultPort)
	if err := cfg.Validate(); err != nil {
		return ServiceConfig{}, err
	}
	return cfg, nil
}

func (c ServiceConfig) Validate() error {
	var validationErrors []error

	if strings.TrimSpace(c.ServiceName) == "" {
		validationErrors = append(validationErrors, errors.New("service name is required"))
	}
	if strings.TrimSpace(c.Environment) == "" {
		validationErrors = append(validationErrors, errors.New("environment is required"))
	} else if !isKnownEnvironment(c.Environment) {
		validationErrors = append(validationErrors, fmt.Errorf("unsupported environment %q", c.Environment))
	}
	if strings.ContainsAny(c.Host, " \t\r\n") {
		validationErrors = append(validationErrors, fmt.Errorf("service host %q must not contain whitespace", c.Host))
	}
	if _, err := strconv.Atoi(strings.TrimSpace(c.Port)); err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("service port %q must be numeric", c.Port))
	} else {
		port, _ := strconv.Atoi(strings.TrimSpace(c.Port))
		if port < 0 || port > 65535 {
			validationErrors = append(validationErrors, fmt.Errorf("service port %d is outside 0-65535", port))
		}
	}

	switch strings.ToLower(strings.TrimSpace(c.StoreBackend)) {
	case "file":
		if strings.TrimSpace(c.StorePath) == "" {
			validationErrors = append(validationErrors, errors.New("AMANDACORE_STORE_PATH is required for file store backend"))
		}
	case "sqlite":
		if strings.TrimSpace(c.SQLitePath) == "" {
			validationErrors = append(validationErrors, errors.New("AMANDACORE_SQLITE_PATH is required for sqlite store backend"))
		}
	default:
		validationErrors = append(validationErrors, fmt.Errorf("unsupported store backend %q", c.StoreBackend))
	}

	if strings.TrimSpace(c.WorldEndpoint) == "" {
		validationErrors = append(validationErrors, errors.New("world endpoint is required"))
	} else if parsed, err := url.Parse(c.WorldEndpoint); err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		validationErrors = append(validationErrors, fmt.Errorf("world endpoint %q must be an absolute http or https URL", c.WorldEndpoint))
	}

	if isProductionEnvironment(c.Environment) {
		if c.AdminToolsEnabled {
			validationErrors = append(validationErrors, errors.New("admin tools must be disabled in production"))
		}
		if strings.TrimSpace(c.LocalSeedFile) == "" || strings.Contains(filepath.ToSlash(c.LocalSeedFile), ".secrets/") {
			validationErrors = append(validationErrors, errors.New("production must not rely on the local dev seed file"))
		}
		if strings.TrimSpace(c.AdminSeedPassword) != "" && isWeakSecret(c.AdminSeedPassword) {
			validationErrors = append(validationErrors, errors.New("production admin seed password is too weak"))
		}
		if strings.EqualFold(strings.TrimSpace(c.AdminSeedUsername), "amanda") && strings.TrimSpace(c.AdminSeedPassword) != "" {
			validationErrors = append(validationErrors, errors.New("production admin seed username must not use the local default"))
		}
	}

	return errors.Join(validationErrors...)
}

func (c ServiceConfig) ListenAddress() string {
	host := strings.TrimSpace(c.Host)
	port := strings.TrimSpace(c.Port)
	if host == "" {
		return ":" + port
	}

	return net.JoinHostPort(host, port)
}

func adminToolsEnabled(environment string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("AMANDACORE_ADMIN_TOOLS_ENABLED")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}

	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "dev", "local", "test", "testing":
		return true
	default:
		return false
	}
}

func isKnownEnvironment(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "dev", "local", "test", "testing", "staging", "stage", "production", "prod":
		return true
	default:
		return false
	}
}

func isProductionEnvironment(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func isWeakSecret(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 16 {
		return true
	}
	lower := strings.ToLower(trimmed)
	return strings.Contains(lower, "password") || strings.Contains(lower, "changeme") || strings.Contains(lower, "amanda")
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
