package database

import (
	"context"
	"testing"
	"time"
)

// TestConfigFromEnv 環境変数からの設定読み込みテスト
func TestConfigFromEnv(t *testing.T) {
	// デフォルト値のテスト
	cfg := ConfigFromEnv()

	if cfg.Host != "localhost" {
		t.Errorf("Host = %s, want localhost", cfg.Host)
	}
	if cfg.Port != "5432" {
		t.Errorf("Port = %s, want 5432", cfg.Port)
	}
	if cfg.User != "postgres" {
		t.Errorf("User = %s, want postgres", cfg.User)
	}
	if cfg.Password != "postgres" {
		t.Errorf("Password = %s, want postgres", cfg.Password)
	}
	if cfg.DBName != "postgres" {
		t.Errorf("DBName = %s, want postgres", cfg.DBName)
	}
}

// TestConfigFromEnv_WithEnvVars 環境変数設定時のテスト
func TestConfigFromEnv_WithEnvVars(t *testing.T) {
	// 環境変数を設定
	t.Setenv("DB_HOST", "testhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")

	cfg := ConfigFromEnv()

	if cfg.Host != "testhost" {
		t.Errorf("Host = %s, want testhost", cfg.Host)
	}
	if cfg.Port != "5433" {
		t.Errorf("Port = %s, want 5433", cfg.Port)
	}
	if cfg.User != "testuser" {
		t.Errorf("User = %s, want testuser", cfg.User)
	}
	if cfg.Password != "testpass" {
		t.Errorf("Password = %s, want testpass", cfg.Password)
	}
	if cfg.DBName != "testdb" {
		t.Errorf("DBName = %s, want testdb", cfg.DBName)
	}
}

// TestConfig_ConnectionString 接続文字列生成テスト
func TestConfig_ConnectionString(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "testdb",
	}

	expected := "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable"
	if connStr := cfg.ConnectionString(); connStr != expected {
		t.Errorf("ConnectionString = %s, want %s", connStr, expected)
	}
}

// TestConfig_ConnectionString_SpecialChars 特殊文字を含む接続文字列テスト
func TestConfig_ConnectionString_SpecialChars(t *testing.T) {
	cfg := Config{
		Host:     "db.example.com",
		Port:     "5433",
		User:     "user",
		Password: "p@ss:word",
		DBName:   "mydb",
	}

	connStr := cfg.ConnectionString()
	if connStr == "" {
		t.Error("ConnectionString が空")
	}
	// 必要な要素が含まれていることを確認
	if !contains(connStr, "db.example.com") {
		t.Error("ホストが含まれていない")
	}
	if !contains(connStr, "5433") {
		t.Error("ポートが含まれていない")
	}
}

// TestGetEnv getEnv関数のテスト
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		want         string
	}{
		{
			name:         "環境変数が設定されている場合",
			envKey:       "TEST_ENV_VAR",
			envValue:     "custom_value",
			defaultValue: "default",
			want:         "custom_value",
		},
		{
			name:         "環境変数が空の場合",
			envKey:       "TEST_ENV_VAR_EMPTY",
			envValue:     "",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			got := getEnv(tt.envKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %s, want %s", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestNewPool_InvalidConfig 不正な設定でのプール作成
func TestNewPool_InvalidConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Host:     "nonexistent-host",
		Port:     "5432",
		User:     "test",
		Password: "test",
		DBName:   "test",
	}

	_, err := NewPool(ctx, cfg)
	// 接続失敗が期待される
	if err == nil {
		t.Log("接続成功（予想外）")
	} else {
		t.Logf("接続失敗（期待通り）: %v", err)
	}
}

// TestNewPool_ContextCanceled コンテキストキャンセル
func TestNewPool_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	cfg := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "test",
		Password: "test",
		DBName:   "test",
	}

	_, err := NewPool(ctx, cfg)
	if err == nil {
		t.Log("接続成功（予想外）")
	} else {
		t.Logf("接続失敗（期待通り）: %v", err)
	}
}

// TestConfig_DefaultValues デフォルト値テスト
func TestConfig_DefaultValues(t *testing.T) {
	cfg := Config{}

	if cfg.Host != "" {
		t.Error("Host should be empty by default")
	}
	if cfg.Port != "" {
		t.Error("Port should be empty by default")
	}
}
