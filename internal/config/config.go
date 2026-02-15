package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/joho/godotenv"
)

type Auth struct {
	JWTExpirationHour     int    `json:"jwt_expiration_hour"`
	SessionExpirationHour int    `json:"session_expiration_hour"`
	CookieSecure          bool   `json:"cookie_secure"`
	CookieSameSite        string `json:"cookie_same_site"`
}

// CORS はCORS設定を表します
type CORS struct {
	AllowOrigins     []string `json:"allow_origins"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

type LogPaths struct {
	App  string `json:"app"`
	Echo string `json:"echo"`
}

type Config struct {
	AppPort  int      `json:"app_port"`
	LogLevel string   `json:"log_level"`
	LogPaths LogPaths `json:"log_paths"`
	// StaticDBPath は静的データ用SQLiteのファイルパスです
	StaticDBPath string `json:"static_db_path"`
	// ShutdownTimeoutSeconds はシャットダウンのタイムアウト秒数
	ShutdownTimeoutSeconds int    `json:"shutdown_timeout_seconds"`
	PwPepper               string // 環境変数から読み込み
	// JWTSecret は環境変数から読み込む機密値であり、命名は役割明示のため維持する。
	// #nosec G117
	JWTSecret string
	Auth      Auth     `json:"auth"`
	CORS      CORS     `json:"cors"`
	Database  Database // 環境変数から読み込み
}

type DbConfig struct {
	DbName             string
	DbHost             string
	DbPort             int
	DbUser             string
	DbPass             string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeSec int
	ConnMaxIdleTimeSec int
}

type DatabasePoolConfig struct {
	MaxOpenConns       *int `json:"max_open_conns"`
	MaxIdleConns       *int `json:"max_idle_conns"`
	ConnMaxLifetimeSec *int `json:"conn_max_lifetime_sec"`
	ConnMaxIdleTimeSec *int `json:"conn_max_idle_time_sec"`
}

type Database struct {
	Pool     DatabasePoolConfig `json:"pool"`
	DbConfig DbConfig
}

// LoadConfig は環境変数から環境を読み取り、対応する設定を読み込みます
func LoadConfig() (Config, error) {
	var config Config

	// .envファイルを読み込み(存在しない場合はスキップ)
	_ = godotenv.Load()

	// 環境変数APP_ENVから環境名を取得
	env := os.Getenv("APP_ENV")
	if env == "" {
		return config, fmt.Errorf("APP_ENV environment variable is required (e.g., develop, staging, production)")
	}

	if err := validateEnv(env); err != nil {
		return config, err
	}

	// JSONファイルから基本設定を読み込み
	path := filepath.Join(info.ConfigDir, env+".settings.json")
	configFile, err := os.Open(path) // #nosec G703 G304 APP_ENVはvalidateEnvで許可値に限定済み
	if err != nil {
		return config, fmt.Errorf("failed to open config file for environment '%s': %w", env, err)
	}
	defer configFile.Close()

	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		return config, fmt.Errorf("failed to decode config file: %w", err)
	}

	if config.ShutdownTimeoutSeconds <= 0 {
		return config, fmt.Errorf("shutdown_timeout_seconds must be greater than 0")
	}

	if strings.TrimSpace(config.StaticDBPath) == "" {
		return config, fmt.Errorf("static_db_path is required")
	}

	if err := normalizeAndValidateDatabasePoolConfig(&config.Database.Pool); err != nil {
		return config, err
	}

	// 環境変数から秘密情報を取得
	config.JWTSecret = os.Getenv("JWT_SECRET")
	if config.JWTSecret == "" {
		return config, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	config.PwPepper = os.Getenv("PW_PEPPER")
	if config.PwPepper == "" {
		return config, fmt.Errorf("PW_PEPPER environment variable is required")
	}

	// データベース設定を環境変数から取得
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return config, fmt.Errorf("DB_NAME environment variable is required")
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return config, fmt.Errorf("DB_HOST environment variable is required")
	}

	dbPortStr := os.Getenv("DB_PORT")
	if dbPortStr == "" {
		return config, fmt.Errorf("DB_PORT environment variable is required")
	}
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		return config, fmt.Errorf("DB_PORT must be a valid integer: %w", err)
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return config, fmt.Errorf("DB_USER environment variable is required")
	}

	dbPass := os.Getenv("DB_PASS")
	if dbPass == "" {
		return config, fmt.Errorf("DB_PASS environment variable is required")
	}

	config.Database.DbConfig = DbConfig{
		DbName:             dbName,
		DbHost:             dbHost,
		DbPort:             dbPort,
		DbUser:             dbUser,
		DbPass:             dbPass,
		MaxOpenConns:       *config.Database.Pool.MaxOpenConns,
		MaxIdleConns:       *config.Database.Pool.MaxIdleConns,
		ConnMaxLifetimeSec: *config.Database.Pool.ConnMaxLifetimeSec,
		ConnMaxIdleTimeSec: *config.Database.Pool.ConnMaxIdleTimeSec,
	}

	return config, nil
}

func normalizeAndValidateDatabasePoolConfig(pool *DatabasePoolConfig) error {
	maxOpenConns := info.DefaultDBMaxOpenConns
	if pool.MaxOpenConns != nil {
		maxOpenConns = *pool.MaxOpenConns
	}

	maxIdleConns := info.DefaultDBMaxIdleConns
	if pool.MaxIdleConns != nil {
		maxIdleConns = *pool.MaxIdleConns
	}

	connMaxLifetimeSec := info.DefaultDBConnMaxLifetimeSec
	if pool.ConnMaxLifetimeSec != nil {
		connMaxLifetimeSec = *pool.ConnMaxLifetimeSec
	}

	connMaxIdleTimeSec := info.DefaultDBConnMaxIdleTimeSec
	if pool.ConnMaxIdleTimeSec != nil {
		connMaxIdleTimeSec = *pool.ConnMaxIdleTimeSec
	}

	if maxOpenConns < 0 {
		return fmt.Errorf("database.pool.max_open_conns must be 0 or greater")
	}
	if maxIdleConns < 0 {
		return fmt.Errorf("database.pool.max_idle_conns must be 0 or greater")
	}
	if connMaxLifetimeSec < 0 {
		return fmt.Errorf("database.pool.conn_max_lifetime_sec must be 0 or greater")
	}
	if connMaxIdleTimeSec < 0 {
		return fmt.Errorf("database.pool.conn_max_idle_time_sec must be 0 or greater")
	}

	if maxOpenConns > 0 && maxIdleConns > maxOpenConns {
		maxIdleConns = maxOpenConns
	}

	pool.MaxOpenConns = &maxOpenConns
	pool.MaxIdleConns = &maxIdleConns
	pool.ConnMaxLifetimeSec = &connMaxLifetimeSec
	pool.ConnMaxIdleTimeSec = &connMaxIdleTimeSec

	return nil
}

func validateEnv(env string) error {
	for _, r := range env {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return fmt.Errorf("invalid APP_ENV: %q", env)
		}
	}
	return nil
}
