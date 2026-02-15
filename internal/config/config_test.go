package config

import "testing"

import "github.com/chunisupport/chunisupport-api/internal/info"

func TestNormalizeAndValidateDatabasePoolConfig_DefaultValues(t *testing.T) {
	pool := DatabasePoolConfig{}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pool.MaxOpenConns != info.DefaultDBMaxOpenConns {
		t.Fatalf("MaxOpenConns = %d, want %d", pool.MaxOpenConns, info.DefaultDBMaxOpenConns)
	}
	if pool.MaxIdleConns != info.DefaultDBMaxIdleConns {
		t.Fatalf("MaxIdleConns = %d, want %d", pool.MaxIdleConns, info.DefaultDBMaxIdleConns)
	}
	if pool.ConnMaxLifetimeSec != info.DefaultDBConnMaxLifetimeSec {
		t.Fatalf("ConnMaxLifetimeSec = %d, want %d", pool.ConnMaxLifetimeSec, info.DefaultDBConnMaxLifetimeSec)
	}
	if pool.ConnMaxIdleTimeSec != info.DefaultDBConnMaxIdleTimeSec {
		t.Fatalf("ConnMaxIdleTimeSec = %d, want %d", pool.ConnMaxIdleTimeSec, info.DefaultDBConnMaxIdleTimeSec)
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_InvalidValue(t *testing.T) {
	pool := DatabasePoolConfig{MaxOpenConns: -1}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_IdleGreaterThanOpen(t *testing.T) {
	pool := DatabasePoolConfig{MaxOpenConns: 10, MaxIdleConns: 11}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err == nil {
		t.Fatal("expected error, got nil")
	}
}
