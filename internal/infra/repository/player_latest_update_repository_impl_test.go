package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestPlayerDataRepository_SaveLatestUpdate_最新5件を保持する(t *testing.T) {
	// Given
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`
		CREATE TABLE player_latest_updates (
			player_id INTEGER NOT NULL,
			schema_version INTEGER NOT NULL,
			result_gzip BLOB NOT NULL,
			source_updated_at DATETIME NOT NULL,
			imported_at DATETIME NOT NULL,
			body_hash TEXT NOT NULL,
			PRIMARY KEY (player_id, source_updated_at)
		)
	`)
	require.NoError(t, err)
	repo := NewPlayerDataRepository(db)
	baseTime := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	makeUpdate := func(playerID, minute int, hash string) *entity.PlayerLatestUpdate {
		update, err := entity.NewPlayerLatestUpdate(playerID, 1, []byte(hash), baseTime.Add(time.Duration(minute)*time.Minute), baseTime.Add(time.Duration(minute+1)*time.Minute), hash)
		require.NoError(t, err)
		return update
	}
	ctx := context.Background()

	// When
	for minute := range info.PlayerUpdateHistoryLimit + 1 {
		require.NoError(t, repo.SaveLatestUpdate(ctx, db, makeUpdate(10, minute, fmt.Sprintf("hash-%d", minute))))
	}
	require.NoError(t, repo.SaveLatestUpdate(ctx, db, makeUpdate(20, 0, "other")))
	require.NoError(t, repo.SaveLatestUpdate(ctx, db, makeUpdate(10, 3, "hash-3")))
	assert.ErrorIs(t, repo.SaveLatestUpdate(ctx, db, makeUpdate(10, 3, "conflict")), entity.ErrConflictingPlayerDataBody)
	require.NoError(t, repo.SaveLatestUpdate(ctx, db, makeUpdate(10, -1, "too-old")))

	// Then
	updates, err := repo.FindRecentUpdatesByPlayerID(ctx, 10)
	require.NoError(t, err)
	require.Len(t, updates, info.PlayerUpdateHistoryLimit)
	for index, update := range updates {
		assert.Equal(t, baseTime.Add(time.Duration(info.PlayerUpdateHistoryLimit-index)*time.Minute), update.SourceUpdatedAt())
	}
	latest, err := repo.FindLatestUpdateByPlayerID(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, updates[0].BodyHash(), latest.BodyHash())
	locked, err := repo.FindLatestUpdateByPlayerIDForUpdate(ctx, db, 10)
	require.NoError(t, err)
	assert.Equal(t, latest.BodyHash(), locked.BodyHash())
	other, err := repo.FindRecentUpdatesByPlayerID(ctx, 20)
	require.NoError(t, err)
	require.Len(t, other, 1)
	assert.Equal(t, "other", other[0].BodyHash())
	empty, err := repo.FindRecentUpdatesByPlayerID(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, empty)
	_, err = repo.FindLatestUpdateByPlayerID(ctx, 999)
	assert.True(t, errors.Is(err, domainrepo.ErrPlayerLatestUpdateNotFound))
}
