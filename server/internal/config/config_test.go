package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := &Config{}
	if err := cfg.setDefaults(); err != nil {
		t.Fatalf("setDefaults() error = %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("default Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.LLM.MaxTokens != 2048 {
		t.Errorf("default LLM.MaxTokens = %d, want %d", cfg.LLM.MaxTokens, 2048)
	}
	if cfg.LLM.Temperature != 0.7 {
		t.Errorf("default LLM.Temperature = %f, want %f", cfg.LLM.Temperature, 0.7)
	}
}

func TestLoadFromTOML(t *testing.T) {
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "config.toml")

	content := `
[server]
host = "127.0.0.1"
port = 9090

[database]
url = "postgres://user:pass@localhost/db"

[auth.wechat]
app_id = "my-app-id"
app_secret = "my-secret"

[llm]
provider = "anthropic"
base_url = "https://api.anthropic.com/v1"
api_key = "sk-test"
model = "claude-3"
max_tokens = 4096
temperature = 0.5
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tomlPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}
	if cfg.Database.URL != "postgres://user:pass@localhost/db" {
		t.Errorf("Database.URL = %q, want %q", cfg.Database.URL, "postgres://user:pass@localhost/db")
	}
	if cfg.Auth.WeChat.AppID != "my-app-id" {
		t.Errorf("Auth.WeChat.AppID = %q, want %q", cfg.Auth.WeChat.AppID, "my-app-id")
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "anthropic")
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("LLM.MaxTokens = %d, want %d", cfg.LLM.MaxTokens, 4096)
	}
	if cfg.LLM.Temperature != 0.5 {
		t.Errorf("LLM.Temperature = %f, want %f", cfg.LLM.Temperature, 0.5)
	}
}

func TestEnvOverridesTOML(t *testing.T) {
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "config.toml")

	content := `
[database]
url = "postgres://file:pass@localhost/db"

[llm]
api_key = "file-key"

[auth.wechat]
app_id = "file-app-id"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	os.Setenv("DATABASE_URL", "postgres://env:pass@localhost/envdb")
	os.Setenv("LLM_API_KEY", "env-key")
	os.Setenv("WECHAT_APP_ID", "env-app-id")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("LLM_API_KEY")
		os.Unsetenv("WECHAT_APP_ID")
	}()

	cfg, err := Load(tomlPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.URL != "postgres://env:pass@localhost/envdb" {
		t.Errorf("Database.URL = %q, want env override %q", cfg.Database.URL, "postgres://env:pass@localhost/envdb")
	}
	if cfg.LLM.APIKey != "env-key" {
		t.Errorf("LLM.APIKey = %q, want env override %q", cfg.LLM.APIKey, "env-key")
	}
	if cfg.Auth.WeChat.AppID != "env-app-id" {
		t.Errorf("Auth.WeChat.AppID = %q, want env override %q", cfg.Auth.WeChat.AppID, "env-app-id")
	}
}

func TestValidateMissingDatabaseURL(t *testing.T) {
	cfg := &Config{
		LLM: LLMConfig{
			APIKey: "sk-test",
		},
	}
	cfg.Auth.WeChat.AppID = "app-123"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing Database.URL, got nil")
	}
}

func TestValidateMissingLLMAPIKey(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/db",
		},
	}
	cfg.Auth.WeChat.AppID = "app-123"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing LLM.APIKey, got nil")
	}
}

func TestValidateMissingWeChatAppID(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/db",
		},
		LLM: LLMConfig{
			APIKey: "sk-test",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing Auth.WeChat.AppID, got nil")
	}
}

func TestValidateAllPresent(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/db",
		},
		LLM: LLMConfig{
			APIKey: "sk-test",
		},
	}
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Auth.WeChat.AppID = "app-123"
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}
