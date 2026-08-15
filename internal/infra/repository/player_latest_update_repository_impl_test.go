package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestPlayerDataRepository_SaveLatestUpdate_新しい収集結果だけを保存する(t *testing.T) {
	// Given
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`
		CREATE TABLE player_latest_updates (
			player_id INTEGER NOT NULL PRIMARY KEY,
			schema_version INTEGER NOT NULL,
			result_gzip BLOB NOT NULL,
			source_updated_at DATETIME NOT NULL,
			imported_at DATETIME NOT NULL,
			body_hash TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	repo := NewPlayerDataRepository(db)
	baseTime := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	first, err := entity.NewPlayerLatestUpdate(10, 1, []byte("first"), baseTime, baseTime.Add(time.Minute), "hash-1")
	require.NoError(t, err)
	duplicate, err := entity.NewPlayerLatestUpdate(10, 9, []byte("duplicate"), baseTime, baseTime.Add(90*time.Second), "hash-1")
	require.NoError(t, err)
	sameSourceNewer, err := entity.NewPlayerLatestUpdate(10, 2, []byte("same-source-newer"), baseTime, baseTime.Add(2*time.Minute), "hash-2")
	require.NoError(t, err)
	sameSourceDelayed, err := entity.NewPlayerLatestUpdate(10, 3, []byte("same-source-delayed"), baseTime, baseTime.Add(105*time.Second), "hash-3")
	require.NoError(t, err)
	older, err := entity.NewPlayerLatestUpdate(10, 1, []byte("older"), baseTime.Add(-time.Minute), baseTime.Add(2*time.Minute), "hash-2")
	require.NoError(t, err)
	newer, err := entity.NewPlayerLatestUpdate(10, 2, []byte("newer"), baseTime.Add(time.Minute), baseTime.Add(3*time.Minute), "hash-3")
	require.NoError(t, err)

	// When
	require.NoError(t, repo.SaveLatestUpdate(context.Background(), db, first))
	require.NoError(t, repo.SaveLatestUpdate(context.Background(), db, duplicate))
	require.NoError(t, repo.SaveLatestUpdate(context.Background(), db, sameSourceNewer))
	require.NoError(t, repo.SaveLatestUpdate(context.Background(), db, sameSourceDelayed))

	var sameSourceSaved struct {
		ResultGzip []byte    `db:"result_gzip"`
		ImportedAt time.Time `db:"imported_at"`
		BodyHash   string    `db:"body_hash"`
	}
	err = db.Get(&sameSourceSaved, `SELECT result_gzip, imported_at, body_hash FROM player_latest_updates WHERE player_id = ?`, 10)
	require.NoError(t, err)
	assert.Equal(t, []byte("same-source-newer"), sameSourceSaved.ResultGzip)
	assert.Equal(t, baseTime.Add(2*time.Minute), sameSourceSaved.ImportedAt)
	assert.Equal(t, "hash-2", sameSourceSaved.BodyHash)

	require.NoError(t, repo.SaveLatestUpdate(context.Background(), db, older))
	require.NoError(t, repo.SaveLatestUpdate(context.Background(), db, newer))

	// Then
	var saved struct {
		SchemaVersion int       `db:"schema_version"`
		ResultGzip    []byte    `db:"result_gzip"`
		SourceAt      time.Time `db:"source_updated_at"`
		ImportedAt    time.Time `db:"imported_at"`
		BodyHash      string    `db:"body_hash"`
	}
	err = db.Get(&saved, `SELECT schema_version, result_gzip, source_updated_at, imported_at, body_hash FROM player_latest_updates WHERE player_id = ?`, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, saved.SchemaVersion)
	assert.Equal(t, []byte("newer"), saved.ResultGzip)
	assert.Equal(t, baseTime.Add(time.Minute), saved.SourceAt)
	assert.Equal(t, baseTime.Add(3*time.Minute), saved.ImportedAt)
	assert.Equal(t, "hash-3", saved.BodyHash)

	found, err := repo.FindLatestUpdateByPlayerID(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 10, found.PlayerID())
	assert.Equal(t, 2, found.SchemaVersion())
	assert.Equal(t, []byte("newer"), found.ResultGzip())
	assert.Equal(t, baseTime.Add(time.Minute), found.SourceUpdatedAt())
	assert.Equal(t, baseTime.Add(3*time.Minute), found.ImportedAt())
	assert.Equal(t, "hash-3", found.BodyHash())

	_, err = repo.FindLatestUpdateByPlayerID(context.Background(), 999)
	assert.True(t, errors.Is(err, domainrepo.ErrPlayerLatestUpdateNotFound))
}
