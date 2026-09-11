package entity

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	// Given
	userName, err := username.NewUserName("testuser")
	require.NoError(t, err)

	// When
	user := NewUser(userName, info.AccountTypePlayer)

	// Then
	require.NotNil(t, user)
	assert.Equal(t, userName, user.Username)
	assert.Equal(t, info.AccountTypePlayer, user.AccountTypeID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.True(t, user.CreatedAt.Equal(user.UpdatedAt))
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
			expectedUID:    ptr("firebase-uid-1"),
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

func TestUser_ChangeUsername(t *testing.T) {
	// Given
	oldName := username.MustNewUserName("oldname")
	newName := username.MustNewUserName("newname")
	user := &User{Username: oldName, UpdatedAt: time.Now().Add(-time.Hour)}
	oldUpdatedAt := user.UpdatedAt

	// When
	user.ChangeUsername(newName)

	// Then
	assert.Equal(t, newName, user.Username)
	assert.True(t, user.UpdatedAt.After(oldUpdatedAt))
}

func TestUser_ChangeAccountType(t *testing.T) {
	baseTime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name               string
		userID             int
		requesterID        int
		currentAccountType int
		newAccountType     int
		wantErr            error
		wantAccountType    int
	}{
		{
			name:               "ADMINは他者をEDITORへ変更できる",
			userID:             2,
			requesterID:        1,
			currentAccountType: info.AccountTypePlayer,
			newAccountType:     info.AccountTypeEditor,
			wantAccountType:    info.AccountTypeEditor,
		},
		{
			name:               "ADMINは自分をEDITORへ降格できない",
			userID:             1,
			requesterID:        1,
			currentAccountType: info.AccountTypeAdmin,
			newAccountType:     info.AccountTypeEditor,
			wantErr:            ErrCannotDemoteOwnAdmin,
			wantAccountType:    info.AccountTypeAdmin,
		},
		{
			name:               "ADMINは自分をADMINのままにできる",
			userID:             1,
			requesterID:        1,
			currentAccountType: info.AccountTypeAdmin,
			newAccountType:     info.AccountTypeAdmin,
			wantAccountType:    info.AccountTypeAdmin,
		},
		{
			name:               "未知の権限は変更できない",
			userID:             2,
			requesterID:        1,
			currentAccountType: info.AccountTypePlayer,
			newAccountType:     99,
			wantErr:            ErrInvalidAccountType,
			wantAccountType:    info.AccountTypePlayer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			user := &User{ID: tt.userID, AccountTypeID: tt.currentAccountType, UpdatedAt: baseTime}

			// When
			err := user.ChangeAccountType(tt.requesterID, tt.newAccountType)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, baseTime, user.UpdatedAt)
			} else {
				require.NoError(t, err)
				assert.True(t, user.UpdatedAt.After(baseTime))
			}
			assert.Equal(t, tt.wantAccountType, user.AccountTypeID)
		})
	}
}

func ptr(value string) *string {
	return &value
}
