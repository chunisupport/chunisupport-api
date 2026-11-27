package repository

import (
	"os"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullChainRainbowMySQL_既存マスタを維持し参照済みの虹を削除しない(t *testing.T) {
	db := mysqltest.NewDatabase(t, "")
	upSQL, err := os.ReadFile("../../../migration/mysql/000074_add_rainbow_full_chain.up.sql")
	require.NoError(t, err)
	downSQL, err := os.ReadFile("../../../migration/mysql/000074_add_rainbow_full_chain.down.sql")
	require.NoError(t, err)

	var names []string
	require.NoError(t, db.Select(&names, "SELECT name FROM full_chain_types ORDER BY sort_order"))
	assert.Equal(t, []string{"NONE", "FULL CHAIN GOLD", "FULL CHAIN PLATINUM", "FULL CHAIN RAINBOW"}, names)
	var legacyIDs []int
	require.NoError(t, db.Select(&legacyIDs, "SELECT id FROM full_chain_types WHERE sort_order < 3 ORDER BY sort_order"))
	assert.Equal(t, []int{1, 2, 3}, legacyIDs)

	_, err = db.Exec(string(downSQL))
	require.NoError(t, err)
	var count int
	require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM full_chain_types WHERE name = 'FULL CHAIN RAINBOW'"))
	assert.Zero(t, count)
	_, err = db.Exec(string(upSQL))
	require.NoError(t, err)
	var rainbowID int
	require.NoError(t, db.Get(&rainbowID, "SELECT id FROM full_chain_types WHERE name = 'FULL CHAIN RAINBOW'"))

	_, err = db.Exec("INSERT INTO users (id, username) VALUES (20, 'rainbow-test')")
	require.NoError(t, err)
	seedPlayerWithHonors(t, db, 1, false)
	_, err = db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx)
			VALUES (1, '0000000000000001', '虹', '作者', 1, '1');
		INSERT INTO charts (id, song_id, difficulty_id, const) VALUES (10, 1, 4, 15)`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO player_record_histories (player_id, chart_id, updated_at, score, clear_lamp_id, combo_lamp_id, full_chain_id)
			VALUES (1, 10, '2026-11-27 00:00:00', 1009000, 2, 3, ?)`, rainbowID)
	require.NoError(t, err)

	_, err = db.Exec(string(downSQL))
	require.NoError(t, err)
	var retainedID int
	require.NoError(t, db.Get(&retainedID, "SELECT id FROM full_chain_types WHERE name = 'FULL CHAIN RAINBOW'"))
	assert.Equal(t, rainbowID, retainedID)

	_, err = db.Exec(string(upSQL))
	require.NoError(t, err)
	require.NoError(t, db.Get(&retainedID, "SELECT id FROM full_chain_types WHERE name = 'FULL CHAIN RAINBOW'"))
	assert.Equal(t, rainbowID, retainedID)
}
