package config

import "testing"

import "github.com/chunisupport/chunisupport-api/internal/info"

func intPtr(v int) *int {
	return &v
}

func TestNormalizeAndValidateDatabasePoolConfig_DefaultValues(t *testing.T) {
	pool := DatabasePoolConfig{}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *pool.MaxOpenConns != info.DefaultDBMaxOpenConns {
		t.Fatalf("MaxOpenConns = %d, want %d", *pool.MaxOpenConns, info.DefaultDBMaxOpenConns)
	}
	if *pool.MaxIdleConns != info.DefaultDBMaxIdleConns {
		t.Fatalf("MaxIdleConns = %d, want %d", *pool.MaxIdleConns, info.DefaultDBMaxIdleConns)
	}
	if *pool.ConnMaxLifetimeSec != info.DefaultDBConnMaxLifetimeSec {
		t.Fatalf("ConnMaxLifetimeSec = %d, want %d", *pool.ConnMaxLifetimeSec, info.DefaultDBConnMaxLifetimeSec)
	}
	if *pool.ConnMaxIdleTimeSec != info.DefaultDBConnMaxIdleTimeSec {
		t.Fatalf("ConnMaxIdleTimeSec = %d, want %d", *pool.ConnMaxIdleTimeSec, info.DefaultDBConnMaxIdleTimeSec)
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_ZeroValues(t *testing.T) {
	pool := DatabasePoolConfig{
		MaxOpenConns:       intPtr(0),
		MaxIdleConns:       intPtr(0),
		ConnMaxLifetimeSec: intPtr(0),
		ConnMaxIdleTimeSec: intPtr(0),
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
	pool := DatabasePoolConfig{MaxOpenConns: intPtr(-1)}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_IdleGreaterThanOpen(t *testing.T) {
	pool := DatabasePoolConfig{MaxOpenConns: intPtr(10), MaxIdleConns: intPtr(11)}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *pool.MaxIdleConns != 10 {
		t.Fatalf("MaxIdleConns = %d, want 10", *pool.MaxIdleConns)
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_IdleDefaultCappedByOpen(t *testing.T) {
	pool := DatabasePoolConfig{MaxOpenConns: intPtr(10)}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *pool.MaxIdleConns != 10 {
		t.Fatalf("MaxIdleConns = %d, want 10", *pool.MaxIdleConns)
	}
}
