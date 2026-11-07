package consolidation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/importer"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/songchart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainframeConsolidate_AppliesConstantsInsideOfficialLevel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws := newMainframeTestWorkspace(t)
	insertWorkspaceChart(t, ws, 1, "範囲内A", 4, 14.0, true)
	insertWorkspaceChart(t, ws, 2, "範囲内B", 3, 13.5, true)
	insertWorkspaceChart(t, ws, 3, "未対象", 4, 12.0, true)
	insertWorkspaceChart(t, ws, 4, "低レベル", 1, 7.0, false)
	insertWorkspaceChart(t, ws, 5, "途中定数", 4, 14.2, false)

	data := importer.MainframeData{
		{Title: "範囲内A", Diff: "MASTER", Genre: "POPS & ANIME", Const: 14.4},
		{Title: "範囲内B", Diff: "EXP", Genre: "POPS & ANIME", Const: 13.5},
		{Title: "低レベル", Diff: "BAS", Genre: "POPS & ANIME", Const: 7.0},
		{Title: "途中定数", Diff: "MAS", Genre: "POPS & ANIME", Const: 14.0},
		{Title: "存在しない曲", Diff: "MASTER", Genre: "POPS & ANIME", Const: 13.9},
		{Title: "範囲内A", Diff: "MASTER", Genre: "ORIGINAL", Const: 1.0},
	}
	err := NewMainframeConsolidator(ws, &data).Consolidate(ctx)
	require.NoError(t, err)

	assertChartConst(t, ws, 1, 4, 14.4, 0)
	assertChartConst(t, ws, 2, 3, 13.5, 0)
	assertChartConst(t, ws, 3, 4, 12.0, 1)
	assertChartConst(t, ws, 4, 1, 7.0, 0)
	assertChartConst(t, ws, 5, 4, 14.0, 0)
}

func TestMainframeConsolidate_AbortsWhenAnyConstantConflictsWithOfficialLevel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws := newMainframeTestWorkspace(t)
	insertWorkspaceChart(t, ws, 1, "Song A", 4, 14.0, true)
	insertWorkspaceChart(t, ws, 2, "Song B", 3, 13.5, true)
	insertWorkspaceChart(t, ws, 3, "Song C", 5, 14.5, true)
	insertWorkspaceChart(t, ws, 4, "Song D", 4, 14.2, false)

	data := importer.MainframeData{
		{Title: "Song C", Diff: "ULT", Genre: "POPS & ANIME", Const: 14.7},
		{Title: "Song A", Diff: "MAS", Genre: "POPS & ANIME", Const: 13.9},
		{Title: "Song B", Diff: "EXP", Genre: "POPS & ANIME", Const: 13.4},
		{Title: "Song D", Diff: "MASTER", Genre: "POPS & ANIME", Const: 14.5},
	}
	err := NewMainframeConsolidator(ws, &data).Consolidate(ctx)

	require.ErrorIs(t, err, errMainframeOfficialMismatch)
	assert.Equal(t, `mainframe data is inconsistent with official data:
- Song A [MASTER]: official=14, mainframe=13.9
- Song B [EXPERT]: official=13+, mainframe=13.4
- Song D [MASTER]: official=14, mainframe=14.5`, err.Error())

	assertChartConst(t, ws, 1, 4, 14.0, 1)
	assertChartConst(t, ws, 2, 3, 13.5, 1)
	assertChartConst(t, ws, 3, 5, 14.5, 1)
	assertChartConst(t, ws, 4, 4, 14.2, 0)
}

func TestMainframeConsolidate_RejectsNonexistentConstantBelowLevel10(t *testing.T) {
	t.Parallel()

	ws := newMainframeTestWorkspace(t)
	insertWorkspaceChart(t, ws, 1, "低レベル", 1, 7.0, false)
	insertWorkspaceChart(t, ws, 2, "範囲内", 4, 14.0, true)

	data := importer.MainframeData{
		{Title: "範囲内", Diff: "MASTER", Genre: "POPS & ANIME", Const: 14.2},
		{Title: "低レベル", Diff: "BASIC", Genre: "POPS & ANIME", Const: 7.4},
	}
	err := NewMainframeConsolidator(ws, &data).Consolidate(context.Background())

	require.ErrorIs(t, err, errMainframeOfficialMismatch)
	assert.Equal(t, `mainframe data is inconsistent with official data:
- 低レベル [BASIC]: official=7, mainframe=7.4`, err.Error())
	assertChartConst(t, ws, 1, 1, 7.0, 0)
	assertChartConst(t, ws, 2, 4, 14.0, 1)
}

func TestMainframeConsolidate_AggregatesConflictsUpToReportLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws := newMainframeTestWorkspace(t)
	limit := info.SongBatchMainframeConflictReportLimit
	data := make(importer.MainframeData, 0, limit+2)

	insertWorkspaceChart(t, ws, 1, "範囲内", 4, 14.0, true)
	data = append(data, importer.MainframeChartData{
		Title: "範囲内", Diff: "MASTER", Genre: "POPS & ANIME", Const: 14.1,
	})

	for i := 1; i <= limit+1; i++ {
		title := fmt.Sprintf("矛盾曲%02d", i)
		songID := i + 1
		insertWorkspaceChart(t, ws, songID, title, 4, 14.0, true)
		data = append(data, importer.MainframeChartData{
			Title: title, Diff: "MAS", Genre: "POPS & ANIME", Const: 13.9,
		})
	}

	err := NewMainframeConsolidator(ws, &data).Consolidate(ctx)
	require.ErrorIs(t, err, errMainframeOfficialMismatch)
	assert.Contains(t, err.Error(), "- 矛盾曲01 [MASTER]: official=14, mainframe=13.9")
	assert.Contains(t, err.Error(), fmt.Sprintf("- 矛盾曲%02d [MASTER]: official=14, mainframe=13.9", limit))
	assert.NotContains(t, err.Error(), fmt.Sprintf("矛盾曲%02d", limit+1))
	assert.Contains(t, err.Error(), "- ... and 1 more")
	assert.Equal(t, limit, strings.Count(err.Error(), "mainframe=13.9"))
	assert.NotContains(t, err.Error(), "範囲内")

	assertChartConst(t, ws, 1, 4, 14.0, 1)
	assertChartConst(t, ws, 2, 4, 14.0, 1)
	assertChartConst(t, ws, limit+2, 4, 14.0, 1)
}

func newMainframeTestWorkspace(t *testing.T) *songchart.SongChartWorkspace {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)"
	ws, err := songchart.NewSongChartWorkspace(context.Background(), songchart.Config{DSN: dsn})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ws.Close())
	})
	return ws
}

func insertWorkspaceChart(t *testing.T, ws *songchart.SongChartWorkspace, songID int, title string, difficultyID int, chartConst float64, unknown bool) {
	t.Helper()
	ctx := context.Background()
	_, err := ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx)
		VALUES (?, ?, ?, 'artist', 1, ?)
	`, songID, fmt.Sprintf("disp-%d", songID), title, fmt.Sprintf("idx-%d", songID))
	require.NoError(t, err)

	unknownInt := 0
	if unknown {
		unknownInt = 1
	}
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown)
		VALUES (?, ?, ?, ?)
	`, songID, difficultyID, chartConst, unknownInt)
	require.NoError(t, err)
}

func assertChartConst(t *testing.T, ws *songchart.SongChartWorkspace, songID, difficultyID int, wantConst float64, wantUnknown int) {
	t.Helper()
	var row struct {
		Const          float64 `db:"const"`
		IsConstUnknown int     `db:"is_const_unknown"`
	}
	err := ws.DB().GetContext(context.Background(), &row, `
		SELECT const, is_const_unknown FROM charts WHERE song_id = ? AND difficulty_id = ?
	`, songID, difficultyID)
	require.NoError(t, err)
	assert.InDelta(t, wantConst, row.Const, 1e-9)
	assert.Equal(t, wantUnknown, row.IsConstUnknown)
}
