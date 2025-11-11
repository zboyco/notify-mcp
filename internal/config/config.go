package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotConfigured indicates that the configuration file does not exist yet.
var ErrNotConfigured = errors.New("notification configuration not found")

// MethodType enumerates supported notification methods.
type MethodType string

const (
	MethodTelegram MethodType = "telegram"
	MethodOS       MethodType = "os"

	// defaultNotificationMessage 是通知内容的默认值。
	defaultNotificationMessage = "即将进行汇报，请注意查看..."

	// DefaultTelegramAPIBaseURL 是 Telegram 官方 API 地址。
	DefaultTelegramAPIBaseURL = "https://api.telegram.org"
)

// Settings holds the notification methods configuration.
type Settings struct {
	Methods             []Method `json:"methods"`
	NotificationMessage string   `json:"notificationMessage,omitempty"`
}

// Method represents a single notification method configuration.
type Method struct {
	Type    MethodType      `json:"type"`
	Config  json.RawMessage `json:"config,omitempty"`
	tgCache *TelegramConfig `json:"-"`
}

// TelegramConfig holds the Telegram related configuration values.
type TelegramConfig struct {
	APIBaseURL string `json:"apiBaseUrl"`
	ChatID     string `json:"chatId"`
	Token      string `json:"token"`
}

// OSConfig holds configuration for OS level notifications.

// Validate ensures the complete settings are ready to use.
func (s Settings) Validate() error {
	for i := range s.Methods {
		if err := s.Methods[i].validate(); err != nil {
			return fmt.Errorf("validate method[%d]: %w", i, err)
		}
	}
	return nil
}

// EffectiveNotificationMessage 返回配置化后的通知内容，若为空则回退到默认文案。
func (s Settings) EffectiveNotificationMessage() string {
	if strings.TrimSpace(s.NotificationMessage) == "" {
		return defaultNotificationMessage
	}
	return s.NotificationMessage
}

func (s Settings) withDefaults() Settings {
	if strings.TrimSpace(s.NotificationMessage) == "" {
		s.NotificationMessage = defaultNotificationMessage
	}
	return s
}

func (m *Method) validate() error {
	if m.Type == "" {
		return errors.New("missing method type")
	}
	switch m.Type {
	case MethodTelegram:
		cfg, err := decodeTelegramConfig(m.Config)
		if err != nil {
			return err
		}
		m.tgCache = &cfg
	case MethodOS:

	default:
		return fmt.Errorf("unsupported method type %q", m.Type)
	}
	return nil
}

func decodeTelegramConfig(data json.RawMessage) (TelegramConfig, error) {
	var cfg TelegramConfig
	if len(data) == 0 {
		return cfg, errors.New("missing telegram config")
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("decode telegram config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// TelegramConfig extracts the Telegram configuration for the method.
func (m Method) TelegramConfig() (TelegramConfig, error) {
	if m.Type != MethodTelegram {
		return TelegramConfig{}, errors.New("notification method is not telegram")
	}
	if m.tgCache != nil {
		return *m.tgCache, nil
	}
	return decodeTelegramConfig(m.Config)
}

// OSConfig extracts the OS notification configuration for the method.
func (m Method) OSConfig() error {
	if m.Type != MethodOS {
		return errors.New("notification method is not os")
	}

	return nil
}

// Validate ensures all required settings are present.
func (c TelegramConfig) Validate() error {
	if c.APIBaseURL == "" {
		return errors.New("missing telegram api base url")
	}
	if c.ChatID == "" {
		return errors.New("missing telegram chat id")
	}
	if c.Token == "" {
		return errors.New("missing telegram token")
	}
	return nil
}

// Path returns the absolute path to the configuration file.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "notify-mcp", "config.json"), nil
}

// Load reads the settings from disk.
func Load() (Settings, error) {
	cfgPath, err := Path()
	if err != nil {
		return Settings{}, err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, ErrNotConfigured
		}
		return Settings{}, fmt.Errorf("read config: %w", err)
	}

	settings, err := decodeSettings(data)
	if err != nil {
		return Settings{}, err
	}
	return settings.withDefaults(), nil
}

func decodeSettings(data []byte) (Settings, error) {
	var marker map[string]json.RawMessage
	if err := json.Unmarshal(data, &marker); err != nil {
		return Settings{}, fmt.Errorf("decode config: %w", err)
	}

	if _, ok := marker["methods"]; ok {
		var settings Settings
		if err := json.Unmarshal(data, &settings); err != nil {
			return Settings{}, fmt.Errorf("decode config: %w", err)
		}
		if err := settings.Validate(); err != nil {
			return Settings{}, err
		}
		return settings, nil
	}

	var legacy legacySettings
	if err := json.Unmarshal(data, &legacy); err != nil {
		return Settings{}, fmt.Errorf("decode legacy config: %w", err)
	}

	settings, err := legacy.toSettings()
	if err != nil {
		return Settings{}, err
	}
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

type legacySettings struct {
	APIBaseURL string `json:"apiBaseUrl"`
	ChatID     string `json:"chatId"`
	Token      string `json:"token"`
}

func (l legacySettings) toSettings() (Settings, error) {
	cfg := TelegramConfig{
		APIBaseURL: l.APIBaseURL,
		ChatID:     l.ChatID,
		Token:      l.Token,
	}
	if err := cfg.Validate(); err != nil {
		return Settings{}, err
	}
	method, err := NewTelegramMethod(cfg)
	if err != nil {
		return Settings{}, err
	}
	return Settings{Methods: []Method{method}}, nil
}

// Save persists the provided settings to disk.
func Save(settings Settings) error {
	settings = settings.withDefaults()
	if err := settings.Validate(); err != nil {
		return err
	}

	cfgPath, err := Path()
	if err != nil {
		return err
	}

	fmt.Println(cfgPath)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// NewTelegramMethod builds a Method entry for Telegram configuration.
func NewTelegramMethod(cfg TelegramConfig) (Method, error) {
	if err := cfg.Validate(); err != nil {
		return Method{}, err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return Method{}, fmt.Errorf("encode telegram config: %w", err)
	}
	return Method{
		Type:    MethodTelegram,
		Config:  data,
		tgCache: &cfg,
	}, nil
}

// NewOSMethod builds a Method entry for OS configuration.
func NewOSMethod() (Method, error) {
	return Method{
		Type: MethodOS,
	}, nil
}
