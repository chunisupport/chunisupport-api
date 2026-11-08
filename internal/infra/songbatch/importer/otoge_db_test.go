package importer

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOtogeDbImporter_DiscardsMistypedFields(t *testing.T) {
	tests := []struct {
		name string
		// Given: otoge-db の JSON
		content string
		// Then: 期待する楽曲データと WARN ログの内容
		expected OtogeDbData
		wantLog  string
	}{
		{
			name: "文字列でない譜面製作者はその楽曲の値だけ捨て、他のフィールドと他の楽曲は取り込む",
			content: `[
				{"id":"8299","title":"Song A","bpm":"180","lev_we_notes":"1234","lev_we_designer":{"exp":"A","mst":"B","ult":""}},
				{"id":"8300","title":"Song B","lev_we_designer":"Designer"}
			]`,
			expected: OtogeDbData{
				{ID: "8299", Title: "Song A", BPM: "180", LevWENotes: "1234"},
				{ID: "8300", Title: "Song B", LevWEDesigner: "Designer"},
			},
			wantLog: "count=1 fields=8299:lev_we_designer\n",
		},
		{
			name:    "1つの楽曲で複数のフィールドが文字列でない場合はすべて捨てる",
			content: `[{"id":"8299","title":"Song A","bpm":180,"lev_we_designer":["A"]}]`,
			expected: OtogeDbData{
				{ID: "8299", Title: "Song A"},
			},
			wantLog: "count=2 fields=8299:bpm,8299:lev_we_designer\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var logs bytes.Buffer
			original := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
			t.Cleanup(func() { slog.SetDefault(original) })

			// When
			result, err := NewOtogeDbImporter().Import(writeImportFixture(t, tt.content))

			// Then
			require.NoError(t, err)
			data, ok := result.Data.(*OtogeDbData)
			require.True(t, ok)
			assert.Equal(t, tt.expected, *data)
			assert.Contains(t, logs.String(), `level=WARN msg="Discarded otoge-db fields with unexpected types" `+tt.wantLog)
		})
	}
}
