package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequester(t *testing.T) {
	user := &entity.User{ID: 7}
	var nilUser *entity.User

	tests := []struct {
		name string
		// Given
		set   bool
		value any
		// Then
		want *entity.User
	}{
		{
			name:  "設定されたユーザーを返す",
			set:   true,
			value: user,
			want:  user,
		},
		{
			name: "未設定はnil",
		},
		{
			name:  "型が異なる値はnil",
			set:   true,
			value: "invalid",
		},
		{
			name:  "nilのユーザーはnil",
			set:   true,
			value: nilUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			c := newRequesterContext(t, tt.set, tt.value)

			// When
			got := Requester(c)

			// Then
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.Same(t, tt.want, got)
		})
	}
}

func TestRequireRequester(t *testing.T) {
	user := &entity.User{ID: 7}
	var nilUser *entity.User

	tests := []struct {
		name string
		// Given
		set   bool
		value any
		// Then
		wantUser *entity.User
		wantErr  bool
	}{
		{
			name:     "設定されたユーザーを返す",
			set:      true,
			value:    user,
			wantUser: user,
		},
		{
			name:    "未設定は401",
			wantErr: true,
		},
		{
			name:    "型が異なる値は401",
			set:     true,
			value:   "invalid",
			wantErr: true,
		},
		{
			// 型アサーションは成功するため、ok だけを見て 401 にしている箇所と同じくエラーにしない。
			name:  "nilのユーザーはエラーにしない",
			set:   true,
			value: nilUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			c := newRequesterContext(t, tt.set, tt.value)

			// When
			got, apiErr := RequireRequester(c)

			// Then
			if tt.wantErr {
				assert.Nil(t, got)
				require.NotNil(t, apiErr)
				assert.Same(t, apierror.ErrUnauthorized, apiErr)
				assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
				return
			}
			assert.Nil(t, apiErr)
			if tt.wantUser == nil {
				assert.Nil(t, got)
				return
			}
			assert.Same(t, tt.wantUser, got)
		})
	}
}

func TestGetRequesterAccountTypeID(t *testing.T) {
	user := &entity.User{AccountTypeID: 3}
	var nilUser *entity.User

	tests := []struct {
		name string
		// Given
		set   bool
		value any
		// Then
		want    int
		wantNil bool
	}{
		{
			name:  "ログインユーザーのAccountTypeIDを返す",
			set:   true,
			value: user,
			want:  3,
		},
		{
			name:    "未設定はnil",
			wantNil: true,
		},
		{
			name:    "型が異なる値はnil",
			set:     true,
			value:   "invalid",
			wantNil: true,
		},
		{
			name:    "nilのユーザーはnil",
			set:     true,
			value:   nilUser,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			c := newRequesterContext(t, tt.set, tt.value)

			// When
			got := GetRequesterAccountTypeID(c)

			// Then
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}

func newRequesterContext(t *testing.T, set bool, value any) *echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())
	if set {
		c.Set(contextKeyUserEntity, value)
	}
	return c
}
