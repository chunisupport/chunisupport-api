package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment/profilecommenttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserProfile(t *testing.T) {
	tests := []struct {
		name    string
		userID  int
		empty   bool
		wantErr error
	}{
		{name: "有効なプロフィール", userID: 7},
		{name: "ユーザーIDが0", wantErr: ErrUserProfileUserIDInvalid},
		{name: "ユーザーIDが負", userID: -1, wantErr: ErrUserProfileUserIDInvalid},
		{name: "ゼロ値のコメントは拒否", userID: 7, empty: true, wantErr: profilecomment.ErrEmpty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			comment := profilecommenttest.New(t, "よろしく")
			if tt.empty {
				comment = profilecomment.ProfileComment{}
			}
			// When
			got, err := NewUserProfile(tt.userID, comment)
			// Then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.userID, got.UserID())
			assert.Equal(t, comment, got.Comment())
		})
	}
}
