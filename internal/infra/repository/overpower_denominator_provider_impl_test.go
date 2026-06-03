package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupOverpowerDenominatorProviderSQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	schema := []string{
		`CREATE TABLE songs (
			id INTEGER PRIMARY KEY,
			is_worldsend INTEGER NOT NULL,
			is_deleted INTEGER NOT NULL
		)`,
		`CREATE TABLE difficulties (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE charts (
			id INTEGER PRIMARY KEY,
			song_id INTEGER NOT NULL,
			difficulty_id INTEGER NOT NULL,
			const REAL NOT NULL
		)`,
		`INSERT INTO difficulties (id, name) VALUES (1, 'BASIC'), (5, 'ULTIMA')`,
		`INSERT INTO songs (id, is_worldsend, is_deleted) VALUES
			(10, 0, 0),
			(20, 0, 0),
			(30, 1, 0),
			(40, 0, 1)`,
		`INSERT INTO charts (id, song_id, difficulty_id, const) VALUES
			(101, 10, 1, 14.0),
			(102, 10, 5, 15.0),
			(201, 20, 1, 13.0),
			(301, 30, 1, 16.0),
			(401, 40, 1, 16.5)`,
	}
	for _, stmt := range schema {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	return db
}

func TestOverpowerDenominatorProvider_Snapshot(t *testing.T) {
	db := setupOverpowerDenominatorProviderSQLite(t)
	provider := NewOverpowerDenominatorProviderWithTTL(db, time.Hour)

	snapshot, err := provider.Snapshot(context.Background())

	require.NoError(t, err)
	require.NotNil(t, snapshot)
	assert.InDelta(t, service.CalcSongMaxOP(15.0)+service.CalcSongMaxOP(13.0), snapshot.GlobalTotal, 0.0001)
	assert.InDelta(t, service.CalcSongMaxOP(15.0), snapshot.SongMaxOP[10], 0.0001)
	assert.InDelta(t, service.CalcSongMaxOP(14.0), snapshot.SongMaxOPWithoutUltima[10], 0.0001)
	assert.NotContains(t, snapshot.SongMaxOP, 30)
	assert.NotContains(t, snapshot.SongMaxOP, 40)
}

func TestOverpowerDenominatorProvider_キャッシュとInvalidate(t *testing.T) {
	db := setupOverpowerDenominatorProviderSQLite(t)
	provider := NewOverpowerDenominatorProviderWithTTL(db, time.Hour)

	first, err := provider.Snapshot(context.Background())
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO songs (id, is_worldsend, is_deleted) VALUES (50, 0, 0)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO charts (id, song_id, difficulty_id, const) VALUES (501, 50, 1, 12.0)`)
	require.NoError(t, err)

	cached, err := provider.Snapshot(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, first.GlobalTotal, cached.GlobalTotal, 0.0001)
	assert.NotContains(t, cached.SongMaxOP, 50)

	provider.Invalidate(context.Background())
	rebuilt, err := provider.Snapshot(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, first.GlobalTotal+service.CalcSongMaxOP(12.0), rebuilt.GlobalTotal, 0.0001)
	assert.Contains(t, rebuilt.SongMaxOP, 50)
}

func TestOverpowerDenominatorProvider_Snapshotはコピーを返す(t *testing.T) {
	db := setupOverpowerDenominatorProviderSQLite(t)
	provider := NewOverpowerDenominatorProviderWithTTL(db, time.Hour)

	first, err := provider.Snapshot(context.Background())
	require.NoError(t, err)
	first.SongMaxOP[10] = 0

	second, err := provider.Snapshot(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, service.CalcSongMaxOP(15.0), second.SongMaxOP[10], 0.0001)
}
