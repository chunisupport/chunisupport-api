package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIToken_Rename(t *testing.T) {
	token, err := NewAPIToken(10, "CLI", strings.Repeat("a", 64), "abcde")
	require.NoError(t, err)

	err = token.Rename("  Discord Bot  ")

	require.NoError(t, err)
	assert.Equal(t, "Discord Bot", token.Name.String())
}

func TestAPIToken_RecordUsage(t *testing.T) {
	token, err := RestoreAPIToken(1, 10, "CLI", strings.Repeat("a", 64), nil, nil, time.Now())
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
	token, err := RestoreAPIToken(1, 10, "既存のトークン", strings.Repeat("a", 64), nil, nil, time.Now())

	require.NoError(t, err)
	assert.Nil(t, token.TokenPrefix)
}

func TestNewAPIToken_RejectsInvalidPrefix(t *testing.T) {
	_, err := NewAPIToken(10, "CLI", strings.Repeat("a", 64), "abcd")

	assert.ErrorIs(t, err, ErrAPITokenPrefixInvalid)
}
