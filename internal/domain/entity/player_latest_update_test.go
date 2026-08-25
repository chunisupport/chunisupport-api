package entity

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayerLatestUpdate(t *testing.T) {
	// Given
	sourceUpdatedAt := time.Date(2026, 7, 16, 1, 2, 3, 0, time.UTC)
	importedAt := sourceUpdatedAt.Add(time.Minute)
	payload := []byte("gzip-payload")

	// When
	update, err := NewPlayerLatestUpdate(10, 1, payload, sourceUpdatedAt, importedAt, "hash")

	// Then
	require.NoError(t, err)
	assert.Equal(t, 10, update.PlayerID())
	assert.Equal(t, 1, update.SchemaVersion())
	assert.Equal(t, payload, update.ResultGzip())
	assert.Equal(t, sourceUpdatedAt, update.SourceUpdatedAt())
	assert.Equal(t, importedAt, update.ImportedAt())
	assert.Equal(t, "hash", update.BodyHash())

	payload[0] = 'X'
	assert.Equal(t, []byte("gzip-payload"), update.ResultGzip())
}

func TestPlayerLatestUpdate_同一取得日時の本文競合を判定する(t *testing.T) {
	// Given
	sourceUpdatedAt := time.Date(2026, 8, 25, 1, 2, 3, 0, time.UTC)
	update, err := NewPlayerLatestUpdate(10, 1, []byte("gzip-payload"), sourceUpdatedAt, sourceUpdatedAt.Add(time.Minute), "same-hash")
	require.NoError(t, err)

	tests := []struct {
		name     string
		sourceAt time.Time
		bodyHash string
		wantErr  error
	}{
		{name: "同一取得日時かつ同一本文は冪等", sourceAt: sourceUpdatedAt, bodyHash: "same-hash"},
		{name: "同一取得日時かつ異なる本文は競合", sourceAt: sourceUpdatedAt, bodyHash: "different-hash", wantErr: ErrConflictingPlayerDataBody},
		{name: "異なる取得日時は本文が異なっても競合しない", sourceAt: sourceUpdatedAt.Add(time.Minute), bodyHash: "different-hash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			err := update.ValidateInputIdentity(tt.sourceAt, tt.bodyHash)

			// Then
			assert.True(t, errors.Is(err, tt.wantErr))
		})
	}
}

func TestNewPlayerLatestUpdate_不正な値を拒否する(t *testing.T) {
	tests := []struct {
		name          string
		playerID      int
		schemaVersion int
		payload       []byte
		sourceAt      time.Time
		importedAt    time.Time
		bodyHash      string
	}{
		{name: "プレイヤーIDが不正", playerID: 0, schemaVersion: 1, payload: []byte("x"), sourceAt: time.Now(), importedAt: time.Now(), bodyHash: "hash"},
		{name: "スキーマバージョンが不正", playerID: 1, schemaVersion: 0, payload: []byte("x"), sourceAt: time.Now(), importedAt: time.Now(), bodyHash: "hash"},
		{name: "保存内容が空", playerID: 1, schemaVersion: 1, sourceAt: time.Now(), importedAt: time.Now(), bodyHash: "hash"},
		{name: "収集日時が空", playerID: 1, schemaVersion: 1, payload: []byte("x"), importedAt: time.Now(), bodyHash: "hash"},
		{name: "登録日時が空", playerID: 1, schemaVersion: 1, payload: []byte("x"), sourceAt: time.Now(), bodyHash: "hash"},
		{name: "本文ハッシュが空", playerID: 1, schemaVersion: 1, payload: []byte("x"), sourceAt: time.Now(), importedAt: time.Now()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := NewPlayerLatestUpdate(tt.playerID, tt.schemaVersion, tt.payload, tt.sourceAt, tt.importedAt, tt.bodyHash)

			// Then
			assert.Error(t, err)
		})
	}
}
