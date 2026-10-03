package songchart

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceChartConstantRange(t *testing.T) {
	ctx := context.Background()
	ws, err := NewSongChartWorkspace(ctx, Config{DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ws.Close()) })
	_, err = ws.DB().ExecContext(ctx, `INSERT INTO songs (display_id, title, artist, genre_id, official_idx) VALUES ('song', '曲', '作者', 1, 'official')`)
	require.NoError(t, err)
	for _, value := range []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 16, 16.1} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			_, err := ws.DB().ExecContext(ctx, `INSERT INTO charts (song_id, difficulty_id, const) VALUES (1, 1, ?)`, value)
			if value < 1 || value > 16 {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				_, err = ws.DB().ExecContext(ctx, `DELETE FROM charts WHERE song_id = 1`)
				require.NoError(t, err)
			}
		})
	}
}
