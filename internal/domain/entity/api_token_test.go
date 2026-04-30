package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAPIToken_必須項目と作成日時が設定される(t *testing.T) {
	before := time.Now().UTC()

	token := NewAPIToken(123, "メイン", "hashed-token")

	after := time.Now().UTC()
	assert.Equal(t, 123, token.UserID)
	assert.Equal(t, "メイン", token.Name)
	assert.Equal(t, "hashed-token", token.HashedToken)
	assert.False(t, token.CreatedAt.IsZero())
	assert.False(t, token.CreatedAt.Before(before))
	assert.False(t, token.CreatedAt.After(after))
	assert.Equal(t, time.UTC, token.CreatedAt.Location())
}
