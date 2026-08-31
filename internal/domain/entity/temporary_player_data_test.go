package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemporaryPlayerData(t *testing.T) {
	createdAt := time.Date(2026, 8, 29, 1, 2, 3, 0, time.UTC)
	expiresAt := createdAt.Add(time.Hour)
	payload := []byte("payload")

	data, err := NewTemporaryPlayerData("token", "192.0.2.1", payload, "hash", createdAt, expiresAt)

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, "token", data.Token)
	assert.Equal(t, "192.0.2.1", data.IPAddress)
	assert.Equal(t, []byte("payload"), data.Payload)
	assert.Equal(t, "hash", data.BodyHash)
	assert.Equal(t, createdAt, data.CreatedAt)
	assert.Equal(t, expiresAt, data.ExpiresAt)
	payload[0] = 'X'
	assert.Equal(t, []byte("payload"), data.Payload)
}

func TestNewTemporaryPlayerData_必須項目の検証(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		token     string
		ipAddress string
		payload   []byte
		expiresAt time.Time
		expected  error
	}{
		{name: "トークンが空", ipAddress: "192.0.2.1", payload: []byte{1}, expiresAt: now.Add(time.Second), expected: ErrTemporaryPlayerDataTokenRequired},
		{name: "IPアドレスが空", token: "token", payload: []byte{1}, expiresAt: now.Add(time.Second), expected: ErrTemporaryPlayerDataIPAddressRequired},
		{name: "ペイロードが空", token: "token", ipAddress: "192.0.2.1", expiresAt: now.Add(time.Second), expected: ErrTemporaryPlayerDataPayloadRequired},
		{name: "有効期限が作成日時と同じ", token: "token", ipAddress: "192.0.2.1", payload: []byte{1}, expiresAt: now, expected: ErrTemporaryPlayerDataExpiresAtInvalid},
		{name: "有効期限が作成日時より前", token: "token", ipAddress: "192.0.2.1", payload: []byte{1}, expiresAt: now.Add(-time.Second), expected: ErrTemporaryPlayerDataExpiresAtInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := NewTemporaryPlayerData(tt.token, tt.ipAddress, tt.payload, "", now, tt.expiresAt)

			assert.Nil(t, data)
			assert.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestTemporaryPlayerData_IsExpired(t *testing.T) {
	expiresAt := time.Date(2026, 8, 29, 2, 0, 0, 0, time.UTC)
	data := &TemporaryPlayerData{ExpiresAt: expiresAt}

	assert.False(t, data.IsExpired(expiresAt.Add(-time.Nanosecond)))
	assert.True(t, data.IsExpired(expiresAt))
	assert.True(t, data.IsExpired(expiresAt.Add(time.Nanosecond)))
	assert.True(t, (*TemporaryPlayerData)(nil).IsExpired(expiresAt))
}
