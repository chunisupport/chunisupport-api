package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestUserLogAttrs(t *testing.T) {
	tests := []struct {
		name string
		// Given: コンテキストに設定する認証済みユーザー
		user *entity.User
		// Then: 期待するログ属性
		expected []any
	}{
		{
			name:     "未認証の場合は属性を返さない",
			user:     nil,
			expected: nil,
		},
		{
			name:     "プレイヤー未連携の場合はuser_idとusernameを返す",
			user:     &entity.User{ID: 42, Username: usernametest.New(t, "testuser")},
			expected: []any{"user_id", 42, "username", "testuser"},
		},
		{
			name:     "プレイヤー連携済みの場合はplayer_idも返す",
			user:     &entity.User{ID: 42, Username: usernametest.New(t, "testuser"), PlayerID: new(7)},
			expected: []any{"user_id", 42, "username", "testuser", "player_id", 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/me", nil)
			c := e.NewContext(req, httptest.NewRecorder())
			if tt.user != nil {
				c.Set(contextKeyUserEntity, tt.user)
			}

			// When
			result := UserLogAttrs(c)

			// Then
			assert.Equal(t, tt.expected, result)
		})
	}
}
