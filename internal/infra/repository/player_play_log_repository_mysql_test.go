package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplacePlayLogsMySQL_失敗時に同一トランザクションの削除をロールバックしプレイヤー削除で連鎖削除する(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	_, err := db.Exec(`INSERT INTO users (id, username) VALUES (20, 'play-log-test')`)
	require.NoError(t, err)
	seedPlayerWithHonors(t, db, 1, false)
	playedAt := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	var masterID int
	require.NoError(t, db.Get(&masterID, `SELECT id FROM difficulties WHERE name = 'MASTER'`))
	ctx := context.Background()
	repo := NewPlayerDataRepository(db)
	seedTx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, repo.ReplacePlayLogs(ctx, seedTx, 1, []entity.PlayerPlayLog{{
		Jacket: new("saved-jacket"), DifficultyID: &masterID, Score: 900000,
		ClearLampID: 1, ComboLampID: 1, FullChainID: 1, PlayedAt: playedAt,
	}, {
		Jacket: new("worldsend-jacket"), Score: 910000,
		ClearLampID: 1, ComboLampID: 1, FullChainID: 1, PlayedAt: playedAt,
	}}))
	require.NoError(t, seedTx.Commit())

	// When
	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	invalidLogs := []entity.PlayerPlayLog{{
		Jacket: new("invalid-jacket"), DifficultyID: &masterID, Score: 950000,
		ClearLampID: 255, ComboLampID: 1, FullChainID: 1, PlayedAt: playedAt,
	}}
	err = repo.ReplacePlayLogs(ctx, tx, 1, invalidLogs)

	// Then: INSERT失敗後にトランザクションを戻すと削除前の行が残る。
	require.Error(t, err)
	require.NoError(t, tx.Rollback())
	var saved []struct {
		Jacket       string `db:"jacket_img_id"`
		DifficultyID *int   `db:"difficulty_id"`
	}
	require.NoError(t, db.Select(&saved, `SELECT jacket_img_id, difficulty_id FROM player_play_logs WHERE player_id = 1 ORDER BY play_order`))
	require.Len(t, saved, 2)
	assert.Equal(t, "saved-jacket", saved[0].Jacket)
	assert.Equal(t, &masterID, saved[0].DifficultyID)
	assert.Equal(t, "worldsend-jacket", saved[1].Jacket)
	assert.Nil(t, saved[1].DifficultyID)

	// When: プレイヤーを削除する。
	_, err = db.Exec(`DELETE FROM players WHERE id = 1`)
	require.NoError(t, err)

	// Then: プレイヤー外部キーのCASCADEで履歴も削除される。
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM player_play_logs WHERE player_id = 1`))
	assert.Zero(t, count)
}
