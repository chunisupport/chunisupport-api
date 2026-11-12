package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIToken_Rename(t *testing.T) {
	token, err := NewAPIToken(10, "CLI", strings.Repeat("a", 64), "abcde", "read_write")
	require.NoError(t, err)

	err = token.Rename("  Discord Bot  ")

	require.NoError(t, err)
	assert.Equal(t, "Discord Bot", token.Name.String())
}

func TestAPIToken_RecordUsage(t *testing.T) {
	token, err := RestoreAPIToken(1, 10, "CLI", strings.Repeat("a", 64), nil, nil, time.Now(), "read_write")
	require.NoError(t, err)
	usedAt := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)

	assert.True(t, token.ShouldRecordUsage(usedAt, time.Hour))
	token.RecordUsage(usedAt)

	require.NotNil(t, token.LastUsedAt)
	assert.Equal(t, usedAt, *token.LastUsedAt)
	assert.False(t, token.ShouldRecordUsage(usedAt.Add(59*time.Minute), time.Hour))
	assert.True(t, token.ShouldRecordUsage(usedAt.Add(time.Hour), time.Hour))
}

func TestRestoreAPIToken_LegacyTokenAllowsMissingPrefix(t *testing.T) {
	token, err := RestoreAPIToken(1, 10, "既存のトークン", strings.Repeat("a", 64), nil, nil, time.Now(), "read_write")

	require.NoError(t, err)
	assert.Nil(t, token.TokenPrefix)
}

func TestNewAPIToken_RejectsInvalidPrefix(t *testing.T) {
	_, err := NewAPIToken(10, "CLI", strings.Repeat("a", 64), "abcd", "read_write")

	assert.ErrorIs(t, err, ErrAPITokenPrefixInvalid)
}

func TestNewAPIToken_RejectsInvalidPermission(t *testing.T) {
	_, err := NewAPIToken(10, "CLI", strings.Repeat("a", 64), "abcde", "write")

	assert.ErrorIs(t, err, ErrAPITokenPermissionInvalid)
}

func TestNewAPIToken_RejectsInvalidUserIDAndHash(t *testing.T) {
	tests := []struct {
		name string
		// Given: 前提条件
		userID int
		hash   string
		// Then: 期待する結果
		expected error
	}{
		{
			name:     "userIDが0の場合はエラー",
			userID:   0,
			hash:     strings.Repeat("a", 64),
			expected: ErrAPITokenUserIDInvalid,
		},
		{
			name:     "hashが63文字の場合はエラー",
			userID:   10,
			hash:     strings.Repeat("a", 63),
			expected: ErrAPITokenHashInvalid,
		},
		{
			name:     "hashが65文字の場合はエラー",
			userID:   10,
			hash:     strings.Repeat("a", 65),
			expected: ErrAPITokenHashInvalid,
		},
		{
			// 奇数長はhex.DecodeStringでも弾かれるため、長さ検査の削除を検知できる偶数長を1件だけ追加する。
			name:     "hashが62文字の場合はエラー",
			userID:   10,
			hash:     strings.Repeat("a", 62),
			expected: ErrAPITokenHashInvalid,
		},
		{
			name:     "hashに非16進文字を含む場合はエラー",
			userID:   10,
			hash:     strings.Repeat("g", 64),
			expected: ErrAPITokenHashInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			token, err := NewAPIToken(tt.userID, "CLI", tt.hash, "abcde", "read_write")

			// Then
			assert.ErrorIs(t, err, tt.expected)
			assert.Nil(t, token)
		})
	}
}

func TestRestoreAPIToken_RejectsInvalidIDAndPrefix(t *testing.T) {
	// Given: 復元時の固定日時
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	validPrefix := "abcde"
	invalidPrefix := "abcdef"
	tests := []struct {
		name string
		// Given: 前提条件
		id     uint64
		prefix *string
		// Then: 期待する結果
		expected error
	}{
		{
			name:     "idが0の場合はエラー",
			id:       0,
			prefix:   &validPrefix,
			expected: ErrAPITokenIDInvalid,
		},
		{
			name:     "prefixが6文字の場合はエラー",
			id:       1,
			prefix:   &invalidPrefix,
			expected: ErrAPITokenPrefixInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			token, err := RestoreAPIToken(tt.id, 10, "CLI", strings.Repeat("a", 64), tt.prefix, nil, createdAt, "read_write")

			// Then
			assert.ErrorIs(t, err, tt.expected)
			assert.Nil(t, token)
		})
	}
}
