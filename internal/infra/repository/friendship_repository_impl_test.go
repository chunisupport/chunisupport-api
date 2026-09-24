package repository

import (
	"context"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFriendshipRepository_ListPendingRequests_非公開ユーザーのプロフィールを返さない(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL,
			player_id INTEGER,
			is_private INTEGER NOT NULL
		);
		CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			player_level INTEGER NOT NULL,
			player_name TEXT NOT NULL,
			calculated_player_rating REAL,
			overpower_value REAL,
			possession_id INTEGER NOT NULL
		);
		CREATE TABLE friendships (
			user_id INTEGER NOT NULL,
			friend_user_id INTEGER NOT NULL,
			status_id INTEGER NOT NULL,
			requested_at DATETIME NOT NULL,
			accepted_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		INSERT INTO users (id, username, player_id, is_private) VALUES
			(1, 'viewer', NULL, 0),
			(2, 'privateuser', 20, 1),
			(3, 'publicuser', 30, 0);
		INSERT INTO players (
			id, player_level, player_name, calculated_player_rating,
			overpower_value, possession_id
		) VALUES
			(20, 50, 'PRIVATE', 16.00, 20000.50, 3),
			(30, 40, 'PUBLIC', 15.00, 18000.25, 2);
		INSERT INTO friendships (
			user_id, friend_user_id, status_id, requested_at, accepted_at, created_at, updated_at
		) VALUES
			(1, 2, 1, CURRENT_TIMESTAMP, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(1, 3, 1, CURRENT_TIMESTAMP, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(2, 1, 1, CURRENT_TIMESTAMP, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(3, 1, 1, CURRENT_TIMESTAMP, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`)
	require.NoError(t, err)
	repo := NewFriendshipRepository()

	// When
	sent, err := repo.ListSentRequests(context.Background(), db, 1)
	require.NoError(t, err)
	received, err := repo.ListReceivedRequests(context.Background(), db, 1)

	// Then
	require.NoError(t, err)
	for _, requests := range [][]*domainFriendshipSummary{toDomainFriendshipSummaries(sent), toDomainFriendshipSummaries(received)} {
		require.Len(t, requests, 2)
		assert.Equal(t, "privateuser", requests[0].username)
		assert.True(t, requests[0].isPrivate)
		assert.Nil(t, requests[0].playerName)
		assert.Nil(t, requests[0].playerLevel)
		assert.Nil(t, requests[0].rating)
		assert.Nil(t, requests[0].overpowerValue)
		assert.Nil(t, requests[0].possessionID)
		assert.Equal(t, "publicuser", requests[1].username)
		assert.False(t, requests[1].isPrivate)
		assert.Equal(t, "PUBLIC", *requests[1].playerName)
		assert.Equal(t, 40, *requests[1].playerLevel)
		assert.Equal(t, 15.0, *requests[1].rating)
		assert.Equal(t, 18000.25, *requests[1].overpowerValue)
		assert.Equal(t, 2, *requests[1].possessionID)
	}
}

func TestFriendshipRepository_ListFriends_非公開ユーザーでも承認済みフレンドにはプロフィールを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL,
			player_id INTEGER,
			is_private INTEGER NOT NULL
		);
		CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			player_level INTEGER NOT NULL,
			player_name TEXT NOT NULL,
			calculated_player_rating REAL,
			overpower_value REAL,
			possession_id INTEGER NOT NULL
		);
		CREATE TABLE friendships (
			user_id INTEGER NOT NULL,
			friend_user_id INTEGER NOT NULL,
			status_id INTEGER NOT NULL,
			requested_at DATETIME NOT NULL,
			accepted_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		INSERT INTO users (id, username, player_id, is_private) VALUES
			(1, 'viewer', NULL, 0),
			(2, 'privateuser', 20, 1);
		INSERT INTO players (
			id, player_level, player_name, calculated_player_rating,
			overpower_value, possession_id
		) VALUES
			(20, 50, 'PRIVATE', 16.00, 20000.50, 3);
		INSERT INTO friendships (
			user_id, friend_user_id, status_id, requested_at, accepted_at, created_at, updated_at
		) VALUES
			(1, 2, 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(2, 1, 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`)
	require.NoError(t, err)
	repo := NewFriendshipRepository()

	// When
	friends, err := repo.ListFriends(context.Background(), db, 1)

	// Then
	require.NoError(t, err)
	got := toDomainFriendshipSummaries(friends)
	require.Len(t, got, 1)
	assert.Equal(t, "privateuser", got[0].username)
	assert.True(t, got[0].isPrivate)
	assert.Equal(t, "PRIVATE", *got[0].playerName)
	assert.Equal(t, 20000.5, *got[0].overpowerValue)
	assert.Equal(t, 3, *got[0].possessionID)
}

type domainFriendshipSummary struct {
	username       string
	isPrivate      bool
	playerName     *string
	playerLevel    *int
	rating         *float64
	overpowerValue *float64
	possessionID   *int
}

func toDomainFriendshipSummaries(items []*domainrepo.FriendshipWithUserSummary) []*domainFriendshipSummary {
	result := make([]*domainFriendshipSummary, 0, len(items))
	for _, item := range items {
		result = append(result, &domainFriendshipSummary{
			username:       item.User.Username,
			isPrivate:      item.User.IsPrivate,
			playerName:     item.User.PlayerName,
			playerLevel:    item.User.PlayerLevel,
			rating:         item.User.Rating,
			overpowerValue: item.User.OverpowerValue,
			possessionID:   item.User.PossessionID,
		})
	}
	return result
}
