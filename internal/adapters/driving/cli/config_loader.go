package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Server          ServerConfig          `yaml:"server"`
	Storage         StorageConfig         `yaml:"storage"`
	NodeDefinitions NodeDefinitionsConfig `yaml:"nodeDefinitions"`
	Auth            AuthConfig            `yaml:"auth"`
	Deploy          DeployConfig          `yaml:"deploy"`
	CommandHistory  CommandHistoryConfig  `yaml:"commandHistory"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
}

// Addr returns the host:port string.
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// StorageConfig holds persistence settings.
type StorageConfig struct {
	Adapter string       `yaml:"adapter"`
	SQLite  SQLiteConfig `yaml:"sqlite"`
}

// SQLiteConfig holds SQLite-specific settings.
type SQLiteConfig struct {
	Path    string `yaml:"path"`
	WALMode bool   `yaml:"walMode"`
}

// NodeDefinitionsConfig holds node definition loader settings.
type NodeDefinitionsConfig struct {
	Path string `yaml:"path"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Provider string        `yaml:"provider"`
	GitHub   GitHubConfig  `yaml:"github"`
	Session  SessionConfig `yaml:"session"`
}

// GitHubConfig holds GitHub OAuth settings.
type GitHubConfig struct {
	ClientID     string   `yaml:"clientId"`
	ClientSecret string   `yaml:"clientSecret"`
	Scopes       []string `yaml:"scopes"`
	AllowedOrgs  []string `yaml:"allowedOrgs"`
}

// SessionConfig holds session management settings.
type SessionConfig struct {
	Secret string `yaml:"secret"`
	MaxAge int    `yaml:"maxAge"` // seconds
	Secure bool   `yaml:"secure"`
}

// DeployConfig holds deploy target settings.
type DeployConfig struct {
	DefaultTarget string                  `yaml:"defaultTarget"`
	HMACSecret    string                  `yaml:"hmacSecret"`
	Targets       map[string]TargetConfig `yaml:"targets"`
}

// TargetConfig holds per-target deploy settings.
type TargetConfig struct {
	WebhookURL    string        `yaml:"webhookURL"`
	StatusBaseURL string        `yaml:"statusBaseURL"` // Optional; empty auto-derives, "-" disables
	Timeout       time.Duration `yaml:"timeout"`
	Retries       int           `yaml:"retries"`
}

// CommandHistoryConfig holds undo/redo settings.
type CommandHistoryConfig struct {
	MaxUndoDepth int `yaml:"maxUndoDepth"`
}

// Load reads and merges config from the given YAML files.
// It loads .env files first (if present), then expands ${VAR} references
// in YAML values from the environment.
func Load(paths ...string) (*Config, error) {
	// Load .env file if present (does not override existing env vars)
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("failed to load .env file", "error", err)
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Storage: StorageConfig{
			Adapter: "sqlite",
			SQLite:  SQLiteConfig{Path: "./data/graphiti.db", WALMode: true},
		},
		NodeDefinitions: NodeDefinitionsConfig{Path: "./config/nodes"},
		Auth: AuthConfig{
			Provider: "fake",
			Session: SessionConfig{
				MaxAge: 86400,
			},
		},
		CommandHistory: CommandHistoryConfig{MaxUndoDepth: 100},
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading config %s: %w", path, err)
		}

		expanded := expandEnvVars(string(data))
		if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
			return nil, fmt.Errorf("parsing config %s: %w", path, err)
		}
	}

	return cfg, nil
}

func expandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		// Keep unexpanded if env var not set
		if strings.HasPrefix(key, "{") && strings.HasSuffix(key, "}") {
			return "${" + key[1:len(key)-1] + "}"
		}
		return ""
	})
}
