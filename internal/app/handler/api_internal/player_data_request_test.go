package api_internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnknownPlayerDataFields(t *testing.T) {
	// Given: playerDataRequest の全フィールドを含むJSON
	knownJSON, err := json.Marshal(playerDataRequest{})
	require.NoError(t, err)

	tests := []struct {
		name string
		// Given
		body []byte
		// Then
		expected []string
	}{
		{
			name:     "入力構造体の全フィールドは未知フィールドとして扱わない",
			body:     knownJSON,
			expected: nil,
		},
		{
			name:     "トップレベルの未知フィールドを名前順で返す",
			body:     []byte(`{"name":"テスト","zeta":1,"alpha":true}`),
			expected: []string{"alpha", "zeta"},
		},
		{
			name:     "ネストしたオブジェクト内の未知フィールドは対象外",
			body:     []byte(`{"team":{"name":"チーム","unknown":1}}`),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			result, err := unknownPlayerDataFields(tt.body)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnknownPlayerDataFields_JSONオブジェクトでなければエラー(t *testing.T) {
	// When
	_, err := unknownPlayerDataFields([]byte(`[1,2,3]`))

	// Then
	assert.Error(t, err)
}

func TestPlayerDataRequest_ToUsecase_チーム情報をチームコードとともに渡す(t *testing.T) {
	// Given
	var request playerDataRequest
	require.NoError(t, json.Unmarshal([]byte(`{"team":{"name":"チーム名","color":"green","code":"AB12CD"}}`), &request))

	// When
	payload := request.toUsecase()

	// Then
	assert.Equal(t, "チーム名", payload.Team.Name)
	assert.Equal(t, "green", payload.Team.Color)
	assert.Equal(t, "AB12CD", payload.Team.Code)
}
