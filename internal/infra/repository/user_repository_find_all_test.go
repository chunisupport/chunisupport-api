package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

func TestUserRepository_FindAllWithPlayer_公開範囲(t *testing.T) {
	tests := []struct {
		name string
		// Given
		find       func(repo *userRepository, db *sqlx.DB) ([]entity.UserWithPlayer, error)
		searchName string
		// Then
		expectedUsernames []string
	}{
		{
			name: "通常版は非公開ユーザーとプレイヤー未連携ユーザーを除外する",
			find: func(repo *userRepository, db *sqlx.DB) ([]entity.UserWithPlayer, error) {
				return repo.FindAllWithPlayer(context.Background(), db, 10, 0, "")
			},
			expectedUsernames: []string{"publicuser", "publicuser2"},
		},
		{
			name: "管理者版は非公開ユーザーとプレイヤー未連携ユーザーを含む",
			find: func(repo *userRepository, db *sqlx.DB) ([]entity.UserWithPlayer, error) {
				return repo.FindAllWithPlayerForAdmin(context.Background(), db, 10, 0, "")
			},
			expectedUsernames: []string{"publicuser", "privateuser", "unlinked", "publicuser2"},
		},
		{
			name: "通常版はユーザー名とプレイヤー名を前方一致で検索する",
			find: func(repo *userRepository, db *sqlx.DB) ([]entity.UserWithPlayer, error) {
				return repo.FindAllWithPlayer(context.Background(), db, 10, 0, "こう")
			},
			expectedUsernames: []string{"publicuser"},
		},
		{
			name: "管理者版はユーザー名とプレイヤー名を前方一致で検索する",
			find: func(repo *userRepository, db *sqlx.DB) ([]entity.UserWithPlayer, error) {
				return repo.FindAllWithPlayerForAdmin(context.Background(), db, 10, 0, "priv")
			},
			expectedUsernames: []string{"privateuser"},
		},
		{
			name: "LIMITとOFFSETでページングする",
			find: func(repo *userRepository, db *sqlx.DB) ([]entity.UserWithPlayer, error) {
				return repo.FindAllWithPlayerForAdmin(context.Background(), db, 2, 1, "")
			},
			expectedUsernames: []string{"privateuser", "unlinked"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupUserWithPlayerTestDB(t)
			repo := &userRepository{db: db}

			// When
			results, err := tt.find(repo, db)

			// Then
			require.NoError(t, err)
			usernames := make([]string, 0, len(results))
			for _, result := range results {
				usernames = append(usernames, result.User.Username.String())
			}
			assert.Equal(t, tt.expectedUsernames, usernames)
		})
	}
}

func TestUserRepository_FindAllWithPlayer_ユーザーとプレイヤーを組み立てる(t *testing.T) {
	// Given
	db := setupUserWithPlayerTestDB(t)
	repo := &userRepository{db: db}

	// When
	results, err := repo.FindAllWithPlayerForAdmin(context.Background(), db, 10, 0, "")

	// Then
	require.NoError(t, err)
	require.Len(t, results, 4)

	private := results[1]
	assert.Equal(t, 2, private.User.ID)
	assert.Equal(t, 3, private.User.AccountTypeID)
	assert.True(t, private.User.IsPrivate)
	assert.True(t, private.User.IsSuspicious)
	assert.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), private.User.CreatedAt.UTC())
	require.NotNil(t, private.User.FirebaseUID)
	assert.Equal(t, "uid-2", *private.User.FirebaseUID)
	require.NotNil(t, private.Player)
	assert.Equal(t, 102, private.Player.ID)
	assert.Equal(t, "ひこうかい", private.Player.Name.String())
	require.NotNil(t, private.Player.CalculatedRating)
	assert.InDelta(t, 17.25, *private.Player.CalculatedRating, 0.0001)
	require.NotNil(t, private.Player.OverpowerValue)
	assert.InDelta(t, 12345.5, *private.Player.OverpowerValue, 0.0001)

	unlinked := results[2]
	assert.Nil(t, unlinked.User.PlayerID)
	assert.Nil(t, unlinked.Player)
}

func setupUserWithPlayerTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			firebase_uid TEXT UNIQUE,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			account_type_id INTEGER NOT NULL,
			player_id INTEGER,
			is_private INTEGER NOT NULL,
			is_suspicious INTEGER NOT NULL
		);
		CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			player_name TEXT NOT NULL,
			calculated_player_rating REAL,
			overpower_value REAL
		);
		INSERT INTO players (id, player_name, calculated_player_rating, overpower_value) VALUES
			(101, 'こうかい', 16.5, 10000.0),
			(102, 'ひこうかい', 17.25, 12345.5),
			(104, 'べつのひと', 15.0, 9000.0);
		INSERT INTO users (id, username, firebase_uid, created_at, updated_at, account_type_id, player_id, is_private, is_suspicious) VALUES
			(1, 'publicuser', 'uid-1', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', 1, 101, 0, 0),
			(2, 'privateuser', 'uid-2', '2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z', 3, 102, 1, 1),
			(3, 'unlinked', NULL, '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z', 1, NULL, 0, 0),
			(4, 'publicuser2', 'uid-4', '2026-01-04T00:00:00Z', '2026-01-04T00:00:00Z', 1, 104, 0, 0);
	`)
	require.NoError(t, err)
	return db
}
