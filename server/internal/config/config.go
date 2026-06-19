package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type AuthConfig struct {
	JWTSecret string       `mapstructure:"jwt_secret"`
	WeChat    WeChatConfig `mapstructure:"wechat"`
}

type WeChatConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	LLM      LLMConfig      `mapstructure:"llm"`
}

func (c *Config) setDefaults() error {
	v := viper.New()
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("llm.max_tokens", 2048)
	v.SetDefault("llm.temperature", 0.7)

	if err := v.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal defaults: %w", err)
	}
	return nil
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("toml")

	v.SetDefault("auth.jwt_secret", "dev-secret-change-in-production")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("llm.max_tokens", 2048)
	v.SetDefault("llm.temperature", 0.7)

	v.SetEnvPrefix("")
	v.BindEnv("database.url", "DATABASE_URL")
	v.BindEnv("llm.api_key", "LLM_API_KEY")
	v.BindEnv("auth.jwt_secret", "JWT_SECRET")
	v.BindEnv("auth.wechat.app_id", "WECHAT_APP_ID")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.setZeroValuesFromEnv(v)

	return &cfg, nil
}

func (c *Config) setZeroValuesFromEnv(v *viper.Viper) {
	if c.Database.URL == "" {
		c.Database.URL = v.GetString("database.url")
	}
	if c.LLM.APIKey == "" {
		c.LLM.APIKey = v.GetString("llm.api_key")
	}
	if c.Auth.WeChat.AppID == "" {
		c.Auth.WeChat.AppID = v.GetString("auth.wechat.app_id")
	}
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = v.GetString("auth.jwt_secret")
	}
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}
	if c.LLM.APIKey == "" {
		return fmt.Errorf("llm.api_key is required")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required")
	}
	if c.Auth.WeChat.AppID == "" {
		return fmt.Errorf("auth.wechat.app_id is required")
	}
	return nil
}
