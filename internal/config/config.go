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

	var errors []string

	// 設定ファイルの検証
	if config.ShutdownTimeoutSeconds <= 0 {
		errors = append(errors, "shutdown_timeout_seconds must be greater than 0")
	}

	if strings.TrimSpace(config.StaticDBPath) == "" {
		errors = append(errors, "static_db_path is required")
	}

	if err := normalizeAndValidateDatabasePoolConfig(&config.Database.Pool); err != nil {
		// normalizeAndValidateDatabasePoolConfigが返すエラーからプレフィックスを削除して個別のエラーとして追加
		errMsg := err.Error()
		prefix := "configuration validation failed: "
		if strings.HasPrefix(errMsg, prefix) {
			errMsg = strings.TrimPrefix(errMsg, prefix)
			// セミコロンで分割して個別のエラーとして追加
			for _, msg := range strings.Split(errMsg, "; ") {
				errors = append(errors, strings.TrimSpace(msg))
			}
		} else {
			errors = append(errors, errMsg)
		}
	}

	// 環境変数から秘密情報を取得
	config.JWTSecret = os.Getenv("JWT_SECRET")
	if config.JWTSecret == "" {
		errors = append(errors, "JWT_SECRET environment variable is required")
	}

	config.PwPepper = os.Getenv("PW_PEPPER")
	if config.PwPepper == "" {
		errors = append(errors, "PW_PEPPER environment variable is required")
	}

	// データベース設定を環境変数から取得
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		errors = append(errors, "DB_NAME environment variable is required")
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		errors = append(errors, "DB_HOST environment variable is required")
	}

	dbPortStr := os.Getenv("DB_PORT")
	var dbPort int
	if dbPortStr == "" {
		errors = append(errors, "DB_PORT environment variable is required")
	} else {
		var err error
		dbPort, err = strconv.Atoi(dbPortStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("DB_PORT must be a valid integer: %v", err))
		}
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		errors = append(errors, "DB_USER environment variable is required")
	}

	dbPass := os.Getenv("DB_PASS")
	if dbPass == "" {
		errors = append(errors, "DB_PASS environment variable is required")
	}

	// すべてのエラーをまとめて返す
	if len(errors) > 0 {
		return config, fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
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

// normalizeAndValidateDatabasePoolConfig はデータベースプール設定の検証と正規化を行います。
// エラーがある場合は、すべてのエラーをまとめて返します。
func normalizeAndValidateDatabasePoolConfig(pool *DatabasePoolConfig) error {
	var errors []string

	// 必須フィールドのチェック
	if pool.MaxOpenConns == nil {
		errors = append(errors, "database.pool.max_open_conns is required")
	}
	if pool.MaxIdleConns == nil {
		errors = append(errors, "database.pool.max_idle_conns is required")
	}
	if pool.ConnMaxLifetimeSec == nil {
		errors = append(errors, "database.pool.conn_max_lifetime_sec is required")
	}
	if pool.ConnMaxIdleTimeSec == nil {
		errors = append(errors, "database.pool.conn_max_idle_time_sec is required")
	}

	// 必須フィールドが欠けている場合はここで返す
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	maxOpenConns := *pool.MaxOpenConns
	maxIdleConns := *pool.MaxIdleConns
	connMaxLifetimeSec := *pool.ConnMaxLifetimeSec
	connMaxIdleTimeSec := *pool.ConnMaxIdleTimeSec

	// 値の範囲チェック
	if maxOpenConns < 0 {
		errors = append(errors, "database.pool.max_open_conns must be 0 or greater")
	}
	if maxIdleConns < 0 {
		errors = append(errors, "database.pool.max_idle_conns must be 0 or greater")
	}
	if connMaxLifetimeSec < 0 {
		errors = append(errors, "database.pool.conn_max_lifetime_sec must be 0 or greater")
	}
	if connMaxIdleTimeSec < 0 {
		errors = append(errors, "database.pool.conn_max_idle_time_sec must be 0 or greater")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	// MaxIdleがMaxOpenより大きい場合の調整
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
