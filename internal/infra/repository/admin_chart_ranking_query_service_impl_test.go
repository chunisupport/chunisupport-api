package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminChartRankingQueryService_ListRecords_全ユーザーから指定件数を返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendChartRankingDB(t, db)
	insertFriendChartRankingFixtures(t, db)

	query := NewAdminChartRankingQueryService(db)

	// When
	data, err := query.GetStandard(context.Background(), "0000000000000001", "MASTER", 2)

	// Then
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, 4, data.Total)
	require.Len(t, data.Records, 2)
	assert.Equal(t, []string{"oneway", "friend1"}, []string{data.Records[0].Username, data.Records[1].Username})
	assert.Equal(t, []uint32{1_010_000, 1_009_500}, []uint32{data.Records[0].Score, data.Records[1].Score})
}

func TestAdminChartRankingQueryService_ListWorldsendRecords_全ユーザーから指定件数を返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendChartRankingDB(t, db)
	insertFriendChartRankingFixtures(t, db)
	insertFriendWorldsendRankingFixtures(t, db)

	query := NewAdminChartRankingQueryService(db)

	// When
	data, err := query.GetWorldsend(context.Background(), "0000000000000002", 3)

	// Then
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, 4, data.Total)
	require.Len(t, data.Records, 3)
	assert.Equal(t, []string{"oneway", "friend1", "friend2"}, []string{data.Records[0].Username, data.Records[1].Username, data.Records[2].Username})
	assert.Equal(t, []uint32{1_010_000, 1_009_700, 1_009_700}, []uint32{data.Records[0].Score, data.Records[1].Score, data.Records[2].Score})
	assert.Equal(t, "ALL JUSTICE", data.Records[0].ComboLamp)
}

func TestAdminChartRankingQueryService_GetStandard_100件目と同点でも100件まで返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendChartRankingDB(t, db)
	insertFriendChartRankingFixtures(t, db)
	insertAdminChartRankingBoundaryFixtures(t, db)
	query := NewAdminChartRankingQueryService(db)

	// When
	data, err := query.GetStandard(context.Background(), "0000000000000001", "MASTER", 100)

	// Then
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, 101, data.Total)
	require.Len(t, data.Records, 100)
	assert.Equal(t, "bulk095", data.Records[99].Username)
	assert.NotContains(t, usernamesFromAdminChartRankingRecords(data.Records), "bulk096")
}

func insertAdminChartRankingBoundaryFixtures(t *testing.T, db *sqlx.DB) {
	t.Helper()
	updatedAt := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	for i := range 97 {
		userID := 1_000 + i
		playerID := 2_000 + i
		username := fmt.Sprintf("bulk%03d", i)
		_, err := db.Exec(`INSERT INTO users (id, username, player_id) VALUES (?, ?, ?)`, userID, username, playerID)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO players (id, user_id, player_name) VALUES (?, ?, ?)`, playerID, userID, username)
		require.NoError(t, err)
		_, err = db.Exec(`
			INSERT INTO player_records (
				player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at
			) VALUES (?, 10, 1000000, 1, 1, 1, ?)
		`, playerID, updatedAt)
		require.NoError(t, err)
	}
}

func usernamesFromAdminChartRankingRecords(records []*domainrepo.AdminChartRankingRecord) []string {
	usernames := make([]string, 0, len(records))
	for _, record := range records {
		usernames = append(usernames, record.Username)
	}
	return usernames
}
