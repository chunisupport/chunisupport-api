package entity

import "testing"

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
				t.Errorf("IsDeleted: got %v, want %v", song.IsDeleted, tt.expectedDeleted)
			}
			if song.IsActive() != tt.expectedActive {
				t.Errorf("IsActive(): got %v, want %v", song.IsActive(), tt.expectedActive)
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
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSongStartsWithEmptyCharts(t *testing.T) {
	song := NewSong()

	if song.Charts == nil {
		t.Fatal("Charts must be initialized")
	}
	if len(song.Charts) != 0 {
		t.Fatalf("Charts length: got %d, want 0", len(song.Charts))
	}
}
