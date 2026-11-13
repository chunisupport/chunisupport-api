package entity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewXAccount(t *testing.T) {
	tests := []struct {
		name     string
		userID   int
		xUserID  string
		username string
		wantErr  error
	}{
		{name: "数値IDと英数字・アンダースコアのユーザー名は生成できる", userID: 1, xUserID: "1234567890123456789", username: "Chuni_Support1"},
		{name: "XのユーザーIDが20桁は生成できる", userID: 1, xUserID: strings.Repeat("1", 20), username: "chuni"},
		{name: "ユーザー名が50文字は生成できる", userID: 1, xUserID: "123", username: strings.Repeat("a", 50)},
		{name: "ユーザーIDが0以下はエラー", userID: 0, xUserID: "123", username: "chuni", wantErr: ErrXAccountUserIDInvalid},
		{name: "XのユーザーIDが空はエラー", userID: 1, xUserID: "", username: "chuni", wantErr: ErrXAccountXUserIDInvalid},
		{name: "XのユーザーIDに数字以外を含むとエラー", userID: 1, xUserID: "12a", username: "chuni", wantErr: ErrXAccountXUserIDInvalid},
		{name: "XのユーザーIDが21桁以上はエラー", userID: 1, xUserID: strings.Repeat("1", 21), username: "chuni", wantErr: ErrXAccountXUserIDInvalid},
		{name: "ユーザー名が空はエラー", userID: 1, xUserID: "123", username: "", wantErr: ErrXAccountUsernameInvalid},
		{name: "ユーザー名に@を含むとエラー", userID: 1, xUserID: "123", username: "@chuni", wantErr: ErrXAccountUsernameInvalid},
		{name: "ユーザー名が51文字以上はエラー", userID: 1, xUserID: "123", username: strings.Repeat("a", 51), wantErr: ErrXAccountUsernameInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			account, err := NewXAccount(tt.userID, tt.xUserID, tt.username)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, account)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.userID, account.UserID())
			assert.Equal(t, tt.xUserID, account.XUserID())
			assert.Equal(t, tt.username, account.Username())
		})
	}
}
