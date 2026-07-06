package service

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForbiddenUsernamePolicy_Validate(t *testing.T) {
	tests := []struct {
		name     string
		exact    []string
		contains []string
		username string
		wantErr  bool
	}{
		{
			name:     "完全一致の禁止語を拒否する",
			exact:    []string{"admin"},
			username: "admin",
			wantErr:  true,
		},
		{
			name:     "完全一致だけなら一部に含むユーザー名は許可する",
			exact:    []string{"admin"},
			username: "iamadmin",
			wantErr:  false,
		},
		{
			name:     "部分一致の禁止語を含むユーザー名を拒否する",
			contains: []string{"admin"},
			username: "iamadmin",
			wantErr:  true,
		},
		{
			name:     "禁止語を含まないユーザー名を許可する",
			exact:    []string{"admin"},
			contains: []string{"system"},
			username: "normaluser",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := NewForbiddenUsernamePolicy(tt.exact, tt.contains)
			require.NoError(t, err)
			userName, err := username.NewUserName(tt.username)
			require.NoError(t, err)

			err = policy.Validate(userName)

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrUsernameForbidden)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewForbiddenUsernamePolicy_空の禁止語を拒否する(t *testing.T) {
	tests := []struct {
		name     string
		exact    []string
		contains []string
	}{
		{name: "空文字", exact: []string{""}},
		{name: "部分一致の空文字", contains: []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewForbiddenUsernamePolicy(tt.exact, tt.contains)

			assert.Error(t, err)
		})
	}
}

func TestNewForbiddenUsernamePolicy_任意の非空文字列を許可する(t *testing.T) {
	_, err := NewForbiddenUsernamePolicy(
		[]string{"Admin", "運営"},
		[]string{"admin-", "管理者"},
	)

	assert.NoError(t, err)
}
