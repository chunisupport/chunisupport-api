package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepositorySaveProtectsFirebaseUIDFromPartialEntity(t *testing.T) {
	// Given
	db := setupUserRepositoryTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO users (id, username, firebase_uid, account_type_id, is_private, is_suspicious)
		VALUES (1, 'user01', 'linked-uid', 1, 0, 0)
	`)
	require.NoError(t, err)

	user := newUserForRepositorySaveTest(t, 1, "user01")
	user.IsPrivate = true
	user.UpdatedAt = time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	repo := &userRepository{db: db}

	// When
	err = repo.Save(ctx, db, user)

	// Then
	require.ErrorIs(t, err, domainrepo.ErrUserConflict)

	var saved struct {
		FirebaseUID *string `db:"firebase_uid"`
		IsPrivate   bool    `db:"is_private"`
	}
	err = db.Get(&saved, `SELECT firebase_uid, is_private FROM users WHERE id = ?`, 1)
	require.NoError(t, err)
	require.NotNil(t, saved.FirebaseUID)
	assert.Equal(t, "linked-uid", *saved.FirebaseUID)
	assert.False(t, saved.IsPrivate)
}

func TestUserRepositorySaveUpdatesMutableFieldsWhenFirebaseUIDMatches(t *testing.T) {
	// Given
	db := setupUserRepositoryTestDB(t)
	defer db.Close()
	ctx := context.Background()
	existingUID := "linked-uid"

	_, err := db.Exec(`
		INSERT INTO users (id, username, firebase_uid, account_type_id, is_private, is_suspicious)
		VALUES (1, 'user01', 'linked-uid', 1, 0, 0)
	`)
	require.NoError(t, err)

	user := newUserForRepositorySaveTest(t, 1, "user01")
	user.FirebaseUID = &existingUID
	user.IsPrivate = true
	user.IsSuspicious = true
	user.UpdatedAt = time.Date(2026, 4, 5, 12, 30, 0, 0, time.UTC)

	repo := &userRepository{db: db}

	// When
	err = repo.Save(ctx, db, user)

	// Then
	require.NoError(t, err)

	var saved struct {
		FirebaseUID  *string `db:"firebase_uid"`
		IsPrivate    bool    `db:"is_private"`
		IsSuspicious bool    `db:"is_suspicious"`
	}
	err = db.Get(&saved, `SELECT firebase_uid, is_private, is_suspicious FROM users WHERE id = ?`, 1)
	require.NoError(t, err)
	require.NotNil(t, saved.FirebaseUID)
	assert.Equal(t, existingUID, *saved.FirebaseUID)
	assert.True(t, saved.IsPrivate)
	assert.True(t, saved.IsSuspicious)
}

func TestUserRepositorySaveProtectsAccountTypeIDFromPartialEntity(t *testing.T) {
	// Given
	db := setupUserRepositoryTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO users (id, username, firebase_uid, account_type_id, is_private, is_suspicious)
		VALUES (1, 'user01', NULL, 1, 0, 0)
	`)
	require.NoError(t, err)

	user := newUserForRepositorySaveTest(t, 1, "user01")
	user.AccountTypeID = 0
	user.IsPrivate = true
	user.UpdatedAt = time.Date(2026, 4, 5, 12, 45, 0, 0, time.UTC)

	repo := &userRepository{db: db}

	// When
	err = repo.Save(ctx, db, user)

	// Then
	require.ErrorIs(t, err, domainrepo.ErrUserConflict)

	var saved struct {
		IsPrivate bool `db:"is_private"`
	}
	err = db.Get(&saved, `SELECT is_private FROM users WHERE id = ?`, 1)
	require.NoError(t, err)
	assert.False(t, saved.IsPrivate)
}

func TestUserRepositorySaveReturnsErrUserNotFoundWhenTargetMissing(t *testing.T) {
	// Given
	db := setupUserRepositoryTestDB(t)
	defer db.Close()
	ctx := context.Background()

	user := newUserForRepositorySaveTest(t, 999, "user01")
	user.UpdatedAt = time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)

	repo := &userRepository{db: db}

	// When
	err := repo.Save(ctx, db, user)

	// Then
	require.ErrorIs(t, err, domainrepo.ErrUserNotFound)
}

func TestUserRepositorySaveCreatesUserWithAggregateTimestamps(t *testing.T) {
	// Given
	db := setupUserRepositoryTestDB(t)
	defer db.Close()
	ctx := context.Background()

	user := newUserForRepositorySaveTest(t, 0, "user01")
	user.CreatedAt = time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC)
	user.UpdatedAt = time.Date(2026, 4, 5, 15, 1, 0, 0, time.UTC)
	playerID := 123
	user.PlayerID = &playerID
	user.IsSuspicious = true

	repo := &userRepository{db: db}

	// When
	err := repo.Save(ctx, db, user)

	// Then
	require.NoError(t, err)
	assert.NotZero(t, user.ID)

	var savedCount int
	err = db.Get(&savedCount, `SELECT COUNT(1) FROM users WHERE id = ? AND username = ? AND created_at = ? AND updated_at = ? AND player_id = ? AND account_type_id = ? AND is_suspicious = 1`, user.ID, user.Username.String(), user.CreatedAt, user.UpdatedAt, playerID, user.AccountTypeID)
	require.NoError(t, err)
	assert.Equal(t, 1, savedCount)
}

func TestUserRepositoryLinkFirebaseUID(t *testing.T) {
	tests := []struct {
		name         string
		initialUID   *string
		currentUID   *string
		newUID       string
		wantErr      error
		wantSavedUID *string
	}{
		{
			name:         "未連携ユーザーにUIDを設定できる",
			initialUID:   nil,
			currentUID:   nil,
			newUID:       "firebase-uid",
			wantSavedUID: stringPtrForUserSaveTest("firebase-uid"),
		},
		{
			name:         "現在値が一致すれば既存UIDを置き換えられる",
			initialUID:   stringPtrForUserSaveTest("old-uid"),
			currentUID:   stringPtrForUserSaveTest("old-uid"),
			newUID:       "new-uid",
			wantSavedUID: stringPtrForUserSaveTest("new-uid"),
		},
		{
			name:         "現在値が一致しなければ更新しない",
			initialUID:   stringPtrForUserSaveTest("persisted-uid"),
			currentUID:   nil,
			newUID:       "new-uid",
			wantErr:      domainrepo.ErrUserConflict,
			wantSavedUID: stringPtrForUserSaveTest("persisted-uid"),
		},
		{
			name:         "対象ユーザーが存在しなければErrUserNotFoundを返す",
			initialUID:   nil,
			currentUID:   nil,
			newUID:       "new-uid",
			wantErr:      domainrepo.ErrUserNotFound,
			wantSavedUID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupUserRepositoryTestDB(t)
			defer db.Close()
			ctx := context.Background()
			var err error

			if tt.wantErr != domainrepo.ErrUserNotFound {
				_, err := db.Exec(`
					INSERT INTO users (id, username, firebase_uid, account_type_id, is_private, is_suspicious)
					VALUES (?, 'user01', ?, 1, 0, 0)
				`, 1, tt.initialUID)
				require.NoError(t, err)
			}

			repo := &userRepository{db: db}

			// When
			err = repo.LinkFirebaseUID(ctx, db, 1, tt.currentUID, tt.newUID, time.Date(2026, 4, 5, 13, 0, 0, 0, time.UTC))

			// Then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantErr == domainrepo.ErrUserNotFound {
				var count int
				err = db.Get(&count, `SELECT COUNT(1) FROM users WHERE id = ?`, 1)
				require.NoError(t, err)
				assert.Zero(t, count)
				return
			}

			var savedUID *string
			err = db.Get(&savedUID, `SELECT firebase_uid FROM users WHERE id = ?`, 1)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSavedUID, savedUID)
		})
	}
}

func setupUserRepositoryTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := setupTestDB(t)
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS account_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO account_types (id, name) VALUES (1, 'PLAYER');

		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			firebase_uid TEXT UNIQUE,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			account_type_id INTEGER NOT NULL DEFAULT 1,
			player_id INTEGER,
			is_private INTEGER NOT NULL DEFAULT 0,
			is_suspicious INTEGER NOT NULL DEFAULT 0
		);
	`)
	require.NoError(t, err)

	return db
}

func newUserForRepositorySaveTest(t *testing.T, id int, name string) *entity.User {
	t.Helper()

	userName, err := username.NewUserName(name)
	require.NoError(t, err)

	user := entity.NewUser(userName, info.AccountTypePlayer)
	user.ID = id

	return user
}

func stringPtrForUserSaveTest(v string) *string {
	return &v
}
