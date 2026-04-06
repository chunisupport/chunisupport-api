package entity

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	// Given
	userName, err := username.NewUserName("testuser")
	require.NoError(t, err)
	hash, err := passwordhash.NewPasswordHash("hashed-password")
	require.NoError(t, err)

	// When
	user := NewUser(userName, hash, info.AccountTypePlayer)

	// Then
	require.NotNil(t, user)
	assert.Equal(t, userName, user.Username)
	assert.Equal(t, hash, user.PasswordHash)
	assert.Equal(t, info.AccountTypePlayer, user.AccountTypeID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.True(t, user.CreatedAt.Equal(user.UpdatedAt))
	assert.Zero(t, user.ID)
	assert.Nil(t, user.FirebaseUID)
	assert.Nil(t, user.PlayerID)
	assert.False(t, user.IsSuspicious)
	assert.False(t, user.IsPrivate)
}

func TestUser_LinkFirebaseUID(t *testing.T) {
	baseTime := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		uid            string
		expectedLinked bool
		expectedUID    *string
	}{
		{
			name:           "UID を設定すると連携済みになる",
			uid:            "firebase-uid-1",
			expectedLinked: true,
			expectedUID:    new("firebase-uid-1"),
		},
		{
			name:           "空白だけのUIDは未連携として扱う",
			uid:            "  ",
			expectedLinked: false,
			expectedUID:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			user := &User{UpdatedAt: baseTime}

			// When
			user.LinkFirebaseUID(tt.uid)

			// Then
			assert.Equal(t, tt.expectedLinked, user.HasLinkedFirebase())
			assert.Equal(t, tt.expectedUID, user.FirebaseUID)
			require.False(t, user.UpdatedAt.IsZero())
			assert.NotEqual(t, baseTime, user.UpdatedAt)
		})
	}
}
