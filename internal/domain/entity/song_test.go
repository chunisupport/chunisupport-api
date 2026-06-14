package entity

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSongDeletionLifecycle(t *testing.T) {
	tests := []struct {
		name string
		// Given: 楽曲の初期状態
		initialDeleted bool
		// When: 実行するコマンド
		action string
		// Then: 期待する状態
		expectedDeleted bool
		expectedActive  bool
	}{
		{
			name:            "有効な楽曲を削除すると論理削除される",
			initialDeleted:  false,
			action:          "delete",
			expectedDeleted: true,
			expectedActive:  false,
		},
		{
			name:            "削除済み楽曲を復活させると有効になる",
			initialDeleted:  true,
			action:          "restore",
			expectedDeleted: false,
			expectedActive:  true,
		},
		{
			name:            "削除済み楽曲を再度削除しても削除状態のまま",
			initialDeleted:  true,
			action:          "delete",
			expectedDeleted: true,
			expectedActive:  false,
		},
		{
			name:            "有効な楽曲を復活させても有効状態のまま",
			initialDeleted:  false,
			action:          "restore",
			expectedDeleted: false,
			expectedActive:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			song := &Song{
				ID:        1,
				DisplayID: "TEST0001",
				Title:     "テスト楽曲",
				IsDeleted: tt.initialDeleted,
				Charts:    []*Chart{},
			}

			// When
			switch tt.action {
			case "delete":
				song.Delete()
			case "restore":
				song.Restore()
			}

			// Then
			if song.IsDeleted != tt.expectedDeleted {
				assert.Failf(t, "アサーション失敗", "IsDeleted: got %v, want %v", song.IsDeleted, tt.expectedDeleted)
			}
			if song.IsActive() != tt.expectedActive {
				assert.Failf(t, "アサーション失敗", "IsActive(): got %v, want %v", song.IsActive(), tt.expectedActive)
			}
		})
	}
}

func TestSongActiveStatusByDeletionState(t *testing.T) {
	tests := []struct {
		name string
		// Given
		deleted bool
		// Then
		expected bool
	}{
		{
			name:     "削除されていない楽曲は有効",
			deleted:  false,
			expected: true,
		},
		{
			name:     "削除された楽曲は無効",
			deleted:  true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			song := &Song{IsDeleted: tt.deleted, Charts: []*Chart{}}

			// When
			result := song.IsActive()

			// Then
			if result != tt.expected {
				assert.Failf(t, "アサーション失敗", "got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSongStartsWithEmptyCharts(t *testing.T) {
	song := NewSong()

	if song.Charts == nil {
		require.Fail(t, "Charts must be initialized")
	}
	if len(song.Charts) != 0 {
		require.Failf(t, "前提条件失敗", "Charts length: got %d, want 0", len(song.Charts))
	}
}

func TestSongHasDifficultyChart(t *testing.T) {
	tests := []struct {
		name string
		// Given: 楽曲が持つ譜面
		charts []*Chart
		// When: 判定する難易度ID
		difficultyID int
		// Then: 期待する判定結果
		expected bool
	}{
		{
			name: "ULTIMA譜面を持つ楽曲はtrueになる",
			charts: []*Chart{
				{DifficultyID: 4},
				{DifficultyID: 5},
			},
			difficultyID: 5,
			expected:     true,
		},
		{
			name: "指定した難易度の譜面を持たない楽曲はfalseになる",
			charts: []*Chart{
				{DifficultyID: 4},
			},
			difficultyID: 5,
			expected:     false,
		},
		{
			name:         "譜面がない楽曲はfalseになる",
			charts:       []*Chart{},
			difficultyID: 5,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			song := &Song{Charts: tt.charts}

			// When
			result := song.HasDifficultyChart(tt.difficultyID)

			// Then
			assert.Equal(t, tt.expected, result)
		})
	}
}
