package config

import "testing"

func intPtr(v int) *int {
	return &v
}

func TestNormalizeAndValidateDatabasePoolConfig_ValidValues(t *testing.T) {
	pool := DatabasePoolConfig{
		MaxOpenConns:       intPtr(25),
		MaxIdleConns:       intPtr(25),
		ConnMaxLifetimeSec: intPtr(300),
		ConnMaxIdleTimeSec: intPtr(60),
	}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	pool := DatabasePoolConfig{
		MaxOpenConns:       intPtr(-1),
		MaxIdleConns:       intPtr(1),
		ConnMaxLifetimeSec: intPtr(1),
		ConnMaxIdleTimeSec: intPtr(1),
	}

	if err := normalizeAndValidateDatabasePoolConfig(&pool); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNormalizeAndValidateDatabasePoolConfig_IdleGreaterThanOpen(t *testing.T) {
	pool := DatabasePoolConfig{MaxOpenConns: intPtr(10), MaxIdleConns: intPtr(11), ConnMaxLifetimeSec: intPtr(300), ConnMaxIdleTimeSec: intPtr(60)}

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
		{name: "max_open_conns", pool: DatabasePoolConfig{MaxIdleConns: intPtr(1), ConnMaxLifetimeSec: intPtr(1), ConnMaxIdleTimeSec: intPtr(1)}},
		{name: "max_idle_conns", pool: DatabasePoolConfig{MaxOpenConns: intPtr(1), ConnMaxLifetimeSec: intPtr(1), ConnMaxIdleTimeSec: intPtr(1)}},
		{name: "conn_max_lifetime_sec", pool: DatabasePoolConfig{MaxOpenConns: intPtr(1), MaxIdleConns: intPtr(1), ConnMaxIdleTimeSec: intPtr(1)}},
		{name: "conn_max_idle_time_sec", pool: DatabasePoolConfig{MaxOpenConns: intPtr(1), MaxIdleConns: intPtr(1), ConnMaxLifetimeSec: intPtr(1)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := normalizeAndValidateDatabasePoolConfig(&tc.pool); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
