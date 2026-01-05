// Package config provides configuration management for the a9s application.
// It uses Viper for configuration loading and supports YAML files, environment
// variables, and hot-reloading.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Configuration Structures
// =============================================================================

// Config represents the complete application configuration.
type Config struct {
	AWS         AWSConfig         `mapstructure:"aws"`
	TUI         TUIConfig         `mapstructure:"tui"`
	Services    ServicesConfig    `mapstructure:"services"`
	Keybindings KeybindingsConfig `mapstructure:"keybindings"`
	Plugins     PluginsConfig     `mapstructure:"plugins"`
	Hooks       HooksConfig       `mapstructure:"hooks"`
	API         APIConfig         `mapstructure:"api"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	Themes      map[string]Theme  `mapstructure:"themes"`
}

// AWSConfig holds AWS connection settings.
type AWSConfig struct {
	Profile string        `mapstructure:"profile"`
	Region  string        `mapstructure:"region"`
	Timeout time.Duration `mapstructure:"timeout"`
	Retry   RetryConfig   `mapstructure:"retry"`
}

// ToCore converts AWSConfig to core.AWSConfig.
func (c *AWSConfig) ToCore() *core.AWSConfig {
	return &core.AWSConfig{
		Profile: c.Profile,
		Region:  c.Region,
		Timeout: c.Timeout,
		Retry: core.RetryConfig{
			MaxAttempts:    c.Retry.MaxAttempts,
			InitialBackoff: c.Retry.InitialBackoff,
		},
	}
}

// RetryConfig configures AWS API retry behavior.
type RetryConfig struct {
	MaxAttempts    int           `mapstructure:"max_attempts"`
	InitialBackoff time.Duration `mapstructure:"initial_backoff"`
}

// TUIConfig holds terminal UI settings.
type TUIConfig struct {
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	Theme           string        `mapstructure:"theme"`
	MouseEnabled    bool          `mapstructure:"mouse_enabled"`
	ShowHelpOnStart bool          `mapstructure:"show_help_on_start"`
	AltScreen       bool          `mapstructure:"alt_screen"`
}

// ServicesConfig configures which services are enabled.
type ServicesConfig struct {
	Enabled []string               `mapstructure:"enabled"`
	EC2     map[string]any         `mapstructure:"ec2"`
	IAM     map[string]any         `mapstructure:"iam"`
	S3      map[string]any         `mapstructure:"s3"`
	Custom  map[string]map[string]any `mapstructure:"custom"`
}

// KeybindingsConfig holds keyboard shortcuts.
type KeybindingsConfig struct {
	Global   GlobalKeybindings `mapstructure:"global"`
	Services map[string]string `mapstructure:"services"`
}

// GlobalKeybindings holds global keyboard shortcuts.
type GlobalKeybindings struct {
	Quit    []string `mapstructure:"quit"`
	Help    []string `mapstructure:"help"`
	Refresh []string `mapstructure:"refresh"`
}

// PluginsConfig configures the plugin system.
type PluginsConfig struct {
	Directory string   `mapstructure:"directory"`
	Enabled   []string `mapstructure:"enabled"`
	HotReload bool     `mapstructure:"hot_reload"`
}

// HooksConfig configures the hook system.
type HooksConfig struct {
	Audit         AuditHookConfig `mapstructure:"audit"`
	Notifications NotifyConfig    `mapstructure:"notifications"`
}

// AuditHookConfig configures the audit hook.
type AuditHookConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	LogFile string `mapstructure:"log_file"`
}

// NotifyConfig configures notifications.
type NotifyConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	SlackWebhook string `mapstructure:"slack_webhook"`
}

// APIConfig configures the REST API server.
type APIConfig struct {
	Enabled bool       `mapstructure:"enabled"`
	Address string     `mapstructure:"address"`
	Auth    AuthConfig `mapstructure:"auth"`
	CORS    CORSConfig `mapstructure:"cors"`
}

// AuthConfig configures API authentication.
type AuthConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Type     string `mapstructure:"type"` // basic, bearer, api-key
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	APIKey   string `mapstructure:"api_key"`
}

// CORSConfig configures CORS settings.
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

// LoggingConfig configures logging.
type LoggingConfig struct {
	Level  string `mapstructure:"level"` // debug, info, warn, error
	Format string `mapstructure:"format"` // text, json
	File   string `mapstructure:"file"`
}

// Theme defines color scheme for the TUI.
type Theme struct {
	Primary    string `mapstructure:"primary"`
	Secondary  string `mapstructure:"secondary"`
	Success    string `mapstructure:"success"`
	Warning    string `mapstructure:"warning"`
	Error      string `mapstructure:"error"`
	Background string `mapstructure:"background"`
	Foreground string `mapstructure:"foreground"`
	Border     string `mapstructure:"border"`
	Muted      string `mapstructure:"muted"`
}

// =============================================================================
// Configuration Loader
// =============================================================================

// Loader handles configuration loading and watching.
type Loader struct {
	mu        sync.RWMutex
	v         *viper.Viper
	config    *Config
	watchers  []func(*Config)
	stopWatch chan struct{}
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	v := viper.New()
	v.SetConfigName("a9s")
	v.SetConfigType("yaml")

	// Search paths (in order of priority)
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "a9s"))
		v.AddConfigPath(filepath.Join(home, ".a9s"))
	}
	v.AddConfigPath("/etc/a9s")

	// Environment variable support
	v.SetEnvPrefix("A9S")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &Loader{
		v:         v,
		stopWatch: make(chan struct{}),
	}
}

// Load loads the configuration from file and environment.
func (l *Loader) Load(path string) (*Config, error) {
	// Set defaults
	l.setDefaults()

	// Override config file path if provided
	if path != "" {
		l.v.SetConfigFile(path)
	}

	// Read config file
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("%w: %v", core.ErrConfigReadFailed, err)
		}
		// Config file not found is OK - use defaults
	}

	// Unmarshal into struct
	var cfg Config
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrConfigInvalid, err)
	}

	// Validate configuration
	if err := l.validate(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrConfigInvalid, err)
	}

	// Expand paths (~ to home directory)
	l.expandPaths(&cfg)

	l.mu.Lock()
	l.config = &cfg
	l.mu.Unlock()

	return &cfg, nil
}

// Get returns the current configuration.
func (l *Loader) Get() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// ConfigFile returns the path to the loaded config file.
func (l *Loader) ConfigFile() string {
	return l.v.ConfigFileUsed()
}

// Watch enables configuration file watching and hot-reload.
func (l *Loader) Watch(callback func(*Config)) {
	l.mu.Lock()
	l.watchers = append(l.watchers, callback)
	l.mu.Unlock()

	l.v.OnConfigChange(func(e fsnotify.Event) {
		l.mu.Lock()
		defer l.mu.Unlock()

		var cfg Config
		if err := l.v.Unmarshal(&cfg); err != nil {
			fmt.Printf("Config reload error: %v\n", err)
			return
		}

		if err := l.validate(&cfg); err != nil {
			fmt.Printf("Config validation error: %v\n", err)
			return
		}

		l.expandPaths(&cfg)
		l.config = &cfg

		for _, w := range l.watchers {
			go w(&cfg)
		}
	})

	l.v.WatchConfig()
}

// Stop stops the configuration watcher.
func (l *Loader) Stop() {
	close(l.stopWatch)
}

// setDefaults sets default configuration values.
func (l *Loader) setDefaults() {
	// AWS defaults
	l.v.SetDefault("aws.region", "us-east-1")
	l.v.SetDefault("aws.timeout", "30s")
	l.v.SetDefault("aws.retry.max_attempts", 3)
	l.v.SetDefault("aws.retry.initial_backoff", "1s")

	// TUI defaults
	l.v.SetDefault("tui.refresh_interval", "5s")
	l.v.SetDefault("tui.theme", "default")
	l.v.SetDefault("tui.mouse_enabled", true)
	l.v.SetDefault("tui.show_help_on_start", false)
	l.v.SetDefault("tui.alt_screen", true)

	// Services defaults
	l.v.SetDefault("services.enabled", []string{"ec2", "iam", "s3"})

	// Keybindings defaults
	l.v.SetDefault("keybindings.global.quit", []string{"q", "ctrl+c"})
	l.v.SetDefault("keybindings.global.help", []string{"?", "h"})
	l.v.SetDefault("keybindings.global.refresh", []string{"r"})
	l.v.SetDefault("keybindings.services.ec2", "1")
	l.v.SetDefault("keybindings.services.iam", "2")
	l.v.SetDefault("keybindings.services.s3", "3")

	// Plugins defaults
	l.v.SetDefault("plugins.directory", "~/.config/a9s/plugins")
	l.v.SetDefault("plugins.hot_reload", true)

	// Hooks defaults
	l.v.SetDefault("hooks.audit.enabled", false)
	l.v.SetDefault("hooks.audit.log_file", "~/.config/a9s/audit.log")
	l.v.SetDefault("hooks.notifications.enabled", false)

	// API defaults
	l.v.SetDefault("api.enabled", false)
	l.v.SetDefault("api.address", "127.0.0.1:8080")
	l.v.SetDefault("api.auth.enabled", true)
	l.v.SetDefault("api.auth.type", "basic")
	l.v.SetDefault("api.cors.enabled", false)
	l.v.SetDefault("api.cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE"})

	// Logging defaults
	l.v.SetDefault("logging.level", "info")
	l.v.SetDefault("logging.format", "text")

	// Theme defaults
	l.v.SetDefault("themes.default.primary", "#FF79C6")
	l.v.SetDefault("themes.default.secondary", "#BD93F9")
	l.v.SetDefault("themes.default.success", "#50FA7B")
	l.v.SetDefault("themes.default.warning", "#FFB86C")
	l.v.SetDefault("themes.default.error", "#FF5555")
	l.v.SetDefault("themes.default.background", "#282A36")
	l.v.SetDefault("themes.default.foreground", "#F8F8F2")
	l.v.SetDefault("themes.default.border", "#6272A4")
	l.v.SetDefault("themes.default.muted", "#6272A4")
}

// validate checks configuration values.
func (l *Loader) validate(cfg *Config) error {
	// Validate AWS config
	if cfg.AWS.Timeout < 0 {
		return fmt.Errorf("aws.timeout must be positive")
	}

	// Validate TUI config
	if cfg.TUI.RefreshInterval < time.Second {
		return fmt.Errorf("tui.refresh_interval must be at least 1s")
	}

	// Validate API config
	if cfg.API.Enabled && cfg.API.Address == "" {
		return fmt.Errorf("api.address required when api.enabled is true")
	}

	// Validate auth type
	validAuthTypes := map[string]bool{"basic": true, "bearer": true, "api-key": true}
	if cfg.API.Auth.Enabled && !validAuthTypes[cfg.API.Auth.Type] {
		return fmt.Errorf("invalid api.auth.type: %s", cfg.API.Auth.Type)
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid logging.level: %s", cfg.Logging.Level)
	}

	// Validate logging format
	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[cfg.Logging.Format] {
		return fmt.Errorf("invalid logging.format: %s", cfg.Logging.Format)
	}

	return nil
}

// expandPaths expands ~ to home directory in paths.
func (l *Loader) expandPaths(cfg *Config) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	cfg.Plugins.Directory = expandPath(cfg.Plugins.Directory, home)
	cfg.Hooks.Audit.LogFile = expandPath(cfg.Hooks.Audit.LogFile, home)
	cfg.Logging.File = expandPath(cfg.Logging.File, home)
}

// expandPath expands ~ to home directory.
func expandPath(path, home string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		return home
	}
	return path
}

// =============================================================================
// Configuration Provider Implementation
// =============================================================================

// Provider implements core.ConfigProvider using Viper.
type Provider struct {
	v *viper.Viper
}

// NewProvider creates a new configuration provider.
func NewProvider(v *viper.Viper) *Provider {
	return &Provider{v: v}
}

// Get returns a configuration value.
func (p *Provider) Get(key string) any {
	return p.v.Get(key)
}

// GetString returns a string configuration value.
func (p *Provider) GetString(key string) string {
	return p.v.GetString(key)
}

// GetInt returns an integer configuration value.
func (p *Provider) GetInt(key string) int {
	return p.v.GetInt(key)
}

// GetBool returns a boolean configuration value.
func (p *Provider) GetBool(key string) bool {
	return p.v.GetBool(key)
}

// GetDuration returns a duration configuration value.
func (p *Provider) GetDuration(key string) time.Duration {
	return p.v.GetDuration(key)
}

// GetStringSlice returns a string slice configuration value.
func (p *Provider) GetStringSlice(key string) []string {
	return p.v.GetStringSlice(key)
}

// GetStringMap returns a string map configuration value.
func (p *Provider) GetStringMap(key string) map[string]any {
	return p.v.GetStringMap(key)
}

// Sub returns a new ConfigProvider scoped to a sub-key.
func (p *Provider) Sub(key string) core.ConfigProvider {
	sub := p.v.Sub(key)
	if sub == nil {
		return &Provider{v: viper.New()}
	}
	return &Provider{v: sub}
}

// IsSet returns whether a key is set.
func (p *Provider) IsSet(key string) bool {
	return p.v.IsSet(key)
}
