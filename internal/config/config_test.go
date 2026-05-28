package config

import (
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndValidateDatabasePoolConfig_ValidValues(t *testing.T) {
	pool := DatabasePoolConfig{
		MaxOpenConns:       new(25),
		MaxIdleConns:       new(25),
		ConnMaxLifetimeSec: new(300),
		ConnMaxIdleTimeSec: new(60),
	}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_ZeroValues(t *testing.T) {
	pool := DatabasePoolConfig{
		MaxOpenConns:       new(0),
		MaxIdleConns:       new(0),
		ConnMaxLifetimeSec: new(0),
		ConnMaxIdleTimeSec: new(0),
	}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *pool.MaxOpenConns != 0 {
		t.Fatalf("MaxOpenConns = %d, want 0", *pool.MaxOpenConns)
	}
	if *pool.MaxIdleConns != 0 {
		t.Fatalf("MaxIdleConns = %d, want 0", *pool.MaxIdleConns)
	}
	if *pool.ConnMaxLifetimeSec != 0 {
		t.Fatalf("ConnMaxLifetimeSec = %d, want 0", *pool.ConnMaxLifetimeSec)
	}
	if *pool.ConnMaxIdleTimeSec != 0 {
		t.Fatalf("ConnMaxIdleTimeSec = %d, want 0", *pool.ConnMaxIdleTimeSec)
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_InvalidValue(t *testing.T) {
	pool := DatabasePoolConfig{
		MaxOpenConns:       new(-1),
		MaxIdleConns:       new(1),
		ConnMaxLifetimeSec: new(1),
		ConnMaxIdleTimeSec: new(1),
	}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_IdleGreaterThanOpen(t *testing.T) {
	pool := DatabasePoolConfig{MaxOpenConns: new(10), MaxIdleConns: new(11), ConnMaxLifetimeSec: new(300), ConnMaxIdleTimeSec: new(60)}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *pool.MaxIdleConns != 10 {
		t.Fatalf("MaxIdleConns = %d, want 10", *pool.MaxIdleConns)
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_MissingValues(t *testing.T) {
	tests := []struct {
		name string
		pool DatabasePoolConfig
	}{
		{name: "max_open_conns", pool: DatabasePoolConfig{MaxIdleConns: new(1), ConnMaxLifetimeSec: new(1), ConnMaxIdleTimeSec: new(1)}},
		{name: "max_idle_conns", pool: DatabasePoolConfig{MaxOpenConns: new(1), ConnMaxLifetimeSec: new(1), ConnMaxIdleTimeSec: new(1)}},
		{name: "conn_max_lifetime_sec", pool: DatabasePoolConfig{MaxOpenConns: new(1), MaxIdleConns: new(1), ConnMaxIdleTimeSec: new(1)}},
		{name: "conn_max_idle_time_sec", pool: DatabasePoolConfig{MaxOpenConns: new(1), MaxIdleConns: new(1), ConnMaxLifetimeSec: new(1)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := normalizeAndValidateDatabasePoolConfig(&tc.pool); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_MultipleErrors(t *testing.T) {
	// 複数のフィールドが欠けている場合、すべてのエラーが報告されることを検証
	pool := DatabasePoolConfig{}

	err := normalizeAndValidateDatabasePoolConfig(&pool)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()

	// すべての必須フィールドのエラーメッセージが含まれているか確認
	expectedErrors := []string{
		"max_open_conns is required",
		"max_idle_conns is required",
		"conn_max_lifetime_sec is required",
		"conn_max_idle_time_sec is required",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("error message should contain %q, but got: %s", expected, errMsg)
		}
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_MultipleInvalidValues(t *testing.T) {
	// 複数のフィールドが無効な値の場合、すべてのエラーが報告されることを検証
	pool := DatabasePoolConfig{
		MaxOpenConns:       new(-1),
		MaxIdleConns:       new(-2),
		ConnMaxLifetimeSec: new(-3),
		ConnMaxIdleTimeSec: new(-4),
	}

	err := normalizeAndValidateDatabasePoolConfig(&pool)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()

	// すべてのバリデーションエラーが含まれているか確認
	expectedErrors := []string{
		"max_open_conns must be 0 or greater",
		"max_idle_conns must be 0 or greater",
		"conn_max_lifetime_sec must be 0 or greater",
		"conn_max_idle_time_sec must be 0 or greater",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("error message should contain %q, but got: %s", expected, errMsg)
		}
	}
}

func TestNormalizeAndValidateFirebaseConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Firebase
		wantErr bool
	}{
		{
			name:    "資格情報ファイルがなければエラーになる",
			config:  Firebase{},
			wantErr: true,
		},
		{
			name:    "資格情報ファイルがあれば通る",
			config:  Firebase{CredentialsFile: "  /tmp/firebase.json  "},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizeAndValidateFirebaseConfig(&tt.config)
			assert.Equal(t, tt.wantErr, err != nil)
			if !tt.wantErr {
				assert.Equal(t, strings.TrimSpace(tt.config.CredentialsFile), tt.config.CredentialsFile)
			}
		})
	}
}

func TestNormalizeAndValidateTurnstileConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Turnstile
		wantErr bool
	}{
		{
			name:    "シークレットキーがなければエラーになる",
			config:  Turnstile{},
			wantErr: true,
		},
		{
			name:    "シークレットキーがあれば通る",
			config:  Turnstile{SecretKey: "  turnstile-secret  "},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizeAndValidateTurnstileConfig(&tt.config)
			assert.Equal(t, tt.wantErr, err != nil)
			if !tt.wantErr {
				assert.Equal(t, strings.TrimSpace(tt.config.SecretKey), tt.config.SecretKey)
			}
		})
	}
}

func TestNormalizeAndValidateDatabaseStartupConfig(t *testing.T) {
	tests := []struct {
		name                string
		config              DatabaseStartupConfig
		wantErr             bool
		expectedMaxWaitSec  int
		expectedIntervalSec int
	}{
		{
			name:                "未設定の場合はデフォルト値が入る",
			config:              DatabaseStartupConfig{},
			wantErr:             false,
			expectedMaxWaitSec:  info.DefaultDBStartupMaxWaitSec,
			expectedIntervalSec: info.DefaultDBStartupIntervalSec,
		},
		{
			name: "明示した待機設定があればその値を使う",
			config: DatabaseStartupConfig{
				MaxWaitSec:  intPtr(60),
				IntervalSec: intPtr(2),
			},
			wantErr:             false,
			expectedMaxWaitSec:  60,
			expectedIntervalSec: 2,
		},
		{
			name: "最大待機秒数は0なら即時失敗として許可する",
			config: DatabaseStartupConfig{
				MaxWaitSec:  intPtr(0),
				IntervalSec: intPtr(1),
			},
			wantErr:             false,
			expectedMaxWaitSec:  0,
			expectedIntervalSec: 1,
		},
		{
			name: "最大待機秒数が負数ならエラーになる",
			config: DatabaseStartupConfig{
				MaxWaitSec:  intPtr(-1),
				IntervalSec: intPtr(1),
			},
			wantErr: true,
		},
		{
			name: "再試行間隔が0ならエラーになる",
			config: DatabaseStartupConfig{
				MaxWaitSec:  intPtr(60),
				IntervalSec: intPtr(0),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizeAndValidateDatabaseStartupConfig(&tt.config)

			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				return
			}

			require.NotNil(t, tt.config.MaxWaitSec)
			require.NotNil(t, tt.config.IntervalSec)
			assert.Equal(t, tt.expectedMaxWaitSec, *tt.config.MaxWaitSec)
			assert.Equal(t, tt.expectedIntervalSec, *tt.config.IntervalSec)
		})
	}
}

func intPtr(v int) *int {
	return &v
}
