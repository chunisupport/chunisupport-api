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

// CORS はCORS設定を表します
type CORS struct {
	AllowOrigins     []string `json:"allow_origins"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

type Logging struct {
	Level      string `json:"level"`
	AppFile    string `json:"app_file"`
	AccessFile string `json:"access_file"`
	Stdout     bool   `json:"stdout"`
	stdoutSet  bool
}

func (l *Logging) UnmarshalJSON(data []byte) error {
	var raw struct {
		Level      string `json:"level"`
		AppFile    string `json:"app_file"`
		AccessFile string `json:"access_file"`
		Stdout     *bool  `json:"stdout"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	l.Level = raw.Level
	l.AppFile = raw.AppFile
	l.AccessFile = raw.AccessFile
	if raw.Stdout != nil {
		l.Stdout = *raw.Stdout
		l.stdoutSet = true
	}

	return nil
}

type Firebase struct {
	CredentialsFile string
}

type Turnstile struct {
	SecretKey string
}

// TempData は一時プレイヤーデータ保存設定です。
type TempData struct {
	MaxTotalMB int `json:"max_total_mb"`
}

type Config struct {
	AppPort int     `json:"app_port"`
	Logging Logging `json:"logging"`
	// StaticDBPath は静的データ用SQLiteのファイルパスです
	StaticDBPath string `json:"static_db_path"`
	// SmallDataDBPath は小規模なユーザー補助データ用SQLiteのファイルパスです。
	SmallDataDBPath string `json:"smalldata_db_path"`
	// ShutdownTimeoutSeconds はシャットダウンのタイムアウト秒数
	ShutdownTimeoutSeconds int       `json:"shutdown_timeout_seconds"`
	CORS                   CORS      `json:"cors"`
	TempData               TempData  `json:"temp_data"`
	Firebase               Firebase  // 環境変数から読み込み
	Turnstile              Turnstile // 環境変数から読み込み
	Database               Database  // 環境変数から読み込み
	loggingSet             bool
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var raw struct {
		AppPort                int      `json:"app_port"`
		Logging                *Logging `json:"logging"`
		StaticDBPath           string   `json:"static_db_path"`
		SmallDataDBPath        string   `json:"smalldata_db_path"`
		ShutdownTimeoutSeconds int      `json:"shutdown_timeout_seconds"`
		CORS                   CORS     `json:"cors"`
		TempData               TempData `json:"temp_data"`
		Database               Database `json:"database"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.AppPort = raw.AppPort
	if raw.Logging != nil {
		c.Logging = *raw.Logging
		c.loggingSet = true
	}
	c.StaticDBPath = raw.StaticDBPath
	c.SmallDataDBPath = raw.SmallDataDBPath
	c.ShutdownTimeoutSeconds = raw.ShutdownTimeoutSeconds
	c.CORS = raw.CORS
	c.TempData = raw.TempData
	c.Database = raw.Database

	return nil
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
	StartupMaxWaitSec  int
	StartupIntervalSec int
}

type DatabasePoolConfig struct {
	MaxOpenConns       *int `json:"max_open_conns"`
	MaxIdleConns       *int `json:"max_idle_conns"`
	ConnMaxLifetimeSec *int `json:"conn_max_lifetime_sec"`
	ConnMaxIdleTimeSec *int `json:"conn_max_idle_time_sec"`
}

type DatabaseStartupConfig struct {
	MaxWaitSec  *int `json:"max_wait_sec"`
	IntervalSec *int `json:"interval_sec"`
}

type Database struct {
	Pool     DatabasePoolConfig    `json:"pool"`
	Startup  DatabaseStartupConfig `json:"startup"`
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
	if strings.TrimSpace(config.SmallDataDBPath) == "" {
		errors = append(errors, "smalldata_db_path is required")
	}

	if err := normalizeAndValidateLoggingConfig(&config.Logging, config.loggingSet); err != nil {
		errMsg := strings.TrimPrefix(err.Error(), "configuration validation failed: ")
		for _, msg := range strings.Split(errMsg, "; ") {
			errors = append(errors, strings.TrimSpace(msg))
		}
	}

	if config.TempData.MaxTotalMB <= 0 {
		config.TempData.MaxTotalMB = info.DefaultTempDataMaxTotalMB
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

	if err := normalizeAndValidateDatabaseStartupConfig(&config.Database.Startup); err != nil {
		// normalizeAndValidateDatabaseStartupConfigが返すエラーからプレフィックスを削除して個別のエラーとして追加
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

	config.Firebase.CredentialsFile = strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_FILE"))
	if err := normalizeAndValidateFirebaseConfig(&config.Firebase); err != nil {
		errors = append(errors, err.Error())
	}

	config.Turnstile.SecretKey = strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY"))
	if err := normalizeAndValidateTurnstileConfig(&config.Turnstile); err != nil {
		errors = append(errors, err.Error())
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
		StartupMaxWaitSec:  *config.Database.Startup.MaxWaitSec,
		StartupIntervalSec: *config.Database.Startup.IntervalSec,
	}

	return config, nil
}

func normalizeAndValidateLoggingConfig(logging *Logging, loggingSet bool) error {
	var errors []string

	if !loggingSet {
		errors = append(errors, "logging section is required")
	}
	if !logging.stdoutSet {
		errors = append(errors, "logging.stdout is required")
	}

	logging.Level = strings.TrimSpace(logging.Level)
	switch logging.Level {
	case "debug", "info", "warn", "error":
	default:
		errors = append(errors, "logging.level must be one of debug, info, warn, error")
	}

	logging.AppFile = strings.TrimSpace(logging.AppFile)
	logging.AccessFile = strings.TrimSpace(logging.AccessFile)

	if !logging.Stdout && logging.AppFile == "" {
		errors = append(errors, "logging.app_file is required when logging.stdout is false")
	}
	if !logging.Stdout && logging.AccessFile == "" {
		errors = append(errors, "logging.access_file is required when logging.stdout is false")
	}

	if logging.AppFile != "" {
		logging.AppFile = filepath.Clean(logging.AppFile)
	}
	if logging.AccessFile != "" {
		logging.AccessFile = filepath.Clean(logging.AccessFile)
	}
	if logging.AppFile != "" && logging.AccessFile != "" && sameLogPath(logging.AppFile, logging.AccessFile) {
		errors = append(errors, "logging.app_file and logging.access_file must be different paths")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

func sameLogPath(a, b string) bool {
	return canonicalLogPath(a) == canonicalLogPath(b)
}

func canonicalLogPath(path string) string {
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return cleanPath
	}

	if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
		return resolvedPath
	}

	dir := filepath.Dir(absPath)
	if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Join(resolvedDir, filepath.Base(absPath))
	}

	return absPath
}

func normalizeAndValidateFirebaseConfig(firebase *Firebase) error {
	if firebase == nil {
		return fmt.Errorf("firebase configuration is required")
	}

	firebase.CredentialsFile = strings.TrimSpace(firebase.CredentialsFile)
	if firebase.CredentialsFile == "" {
		return fmt.Errorf("FIREBASE_CREDENTIALS_FILE environment variable is required")
	}

	return nil
}

func normalizeAndValidateTurnstileConfig(turnstile *Turnstile) error {
	if turnstile == nil {
		return fmt.Errorf("turnstile configuration is required")
	}

	turnstile.SecretKey = strings.TrimSpace(turnstile.SecretKey)
	if turnstile.SecretKey == "" {
		return fmt.Errorf("TURNSTILE_SECRET_KEY environment variable is required")
	}

	return nil
}

// normalizeAndValidateDatabaseStartupConfig は起動時のDB接続待機設定の検証と正規化を行います。
func normalizeAndValidateDatabaseStartupConfig(startup *DatabaseStartupConfig) error {
	if startup.MaxWaitSec == nil {
		defaultMaxWaitSec := info.DefaultDBStartupMaxWaitSec
		startup.MaxWaitSec = &defaultMaxWaitSec
	}
	if startup.IntervalSec == nil {
		defaultIntervalSec := info.DefaultDBStartupIntervalSec
		startup.IntervalSec = &defaultIntervalSec
	}

	var errors []string
	if *startup.MaxWaitSec < 0 {
		errors = append(errors, "database.startup.max_wait_sec must be 0 or greater")
	}
	if *startup.IntervalSec <= 0 {
		errors = append(errors, "database.startup.interval_sec must be greater than 0")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
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
