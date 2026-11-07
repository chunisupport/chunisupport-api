package consolidation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/difficulty"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/importer"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/songchart"
)

// errMainframeOfficialMismatch は mainframe の譜面定数が公式表示レベルと矛盾することを表します。
// 矛盾は個別譜面の欠落ではなくデータソース全体が古い可能性を示すため、呼び出し側はバッチを失敗させます。
var errMainframeOfficialMismatch = errors.New("mainframe data is inconsistent with official data")

type mainframeChartMatch struct {
	SongID        int
	Title         string
	OfficialConst float64
}

type mainframeConstConflict struct {
	songTitle      string
	difficulty     string
	officialLevel  string
	mainframeConst float64
}

type mainframeConstUpdate struct {
	songID     int
	diffID     int
	constValue float64
}

// MainframeConsolidator は mainframe 由来の定数を補完します。
type MainframeConsolidator struct {
	workspace *songchart.SongChartWorkspace
	data      *importer.MainframeData
}

// NewMainframeConsolidator は MainframeConsolidator の新しいインスタンスを作成します。
func NewMainframeConsolidator(workspace *songchart.SongChartWorkspace, data *importer.MainframeData) *MainframeConsolidator {
	return &MainframeConsolidator{
		workspace: workspace,
		data:      data,
	}
}

// Consolidate は mainframe の譜面定数をワークスペースへ反映します。
// 公式表示レベルに収まらない定数が1件でもあれば、一部の譜面だけ更新せずエラーを返します。
func (c *MainframeConsolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || len(*c.data) == 0 {
		slog.Warn("Mainframe data is empty; skipping consolidation")
		return nil
	}

	chartMap, err := c.buildChartMap(ctx)
	if err != nil {
		return err
	}

	updates := make([]mainframeConstUpdate, 0, len(*c.data))
	var conflicts []mainframeConstConflict
	for _, chart := range *c.data {
		title := strings.TrimSpace(chart.Title)
		genre := strings.TrimSpace(chart.Genre)
		if title == "" || chart.Const <= 0 {
			continue
		}

		diffID := difficulty.ParseName(chart.Diff).Int()
		if diffID == 0 {
			continue
		}

		matchKey := fmt.Sprintf("%s|%s|%d", normalizer(title), normalizer(genre), diffID)
		matched, exists := chartMap[matchKey]
		if !exists {
			continue
		}

		if !songbatch.DisplayLevelBandContains(matched.OfficialConst, chart.Const) {
			conflicts = append(conflicts, mainframeConstConflict{
				songTitle:      matched.Title,
				difficulty:     difficultyLabel(diffID),
				officialLevel:  songbatch.FormatOfficialDisplayLevel(matched.OfficialConst),
				mainframeConst: chart.Const,
			})
			continue
		}

		updates = append(updates, mainframeConstUpdate{
			songID:     matched.SongID,
			diffID:     diffID,
			constValue: chart.Const,
		})
	}

	// 1件でも公式レベルと矛盾する定数があれば、mainframe全体を信用できない。
	// 範囲内の譜面だけ先に書くと不整合なワークスペースがMySQLへ同期されるため、UPDATEの前に中断する。
	if len(conflicts) > 0 {
		return newMainframeOfficialMismatchError(conflicts)
	}

	var updated int
	for _, update := range updates {
		result, err := c.workspace.DB().ExecContext(ctx, `
UPDATE charts
SET const = ?, is_const_unknown = 0
WHERE song_id = ? AND difficulty_id = ?
`, update.constValue, update.songID, update.diffID)
		if err != nil {
			return fmt.Errorf("failed to update mainframe chart song_id=%d diff=%d: %w", update.songID, update.diffID, err)
		}

		affected, _ := result.RowsAffected()
		if affected > 0 {
			updated++
		}
	}

	slog.Info("Mainframe chart constants updated", "count", updated)
	return nil
}

func (c *MainframeConsolidator) buildChartMap(ctx context.Context) (map[string]mainframeChartMatch, error) {
	var rows []struct {
		SongID       int     `db:"song_id"`
		DifficultyID int     `db:"difficulty_id"`
		Title        string  `db:"title"`
		GenreName    string  `db:"genre_name"`
		Const        float64 `db:"const"`
	}
	query := `
		SELECT c.song_id, c.difficulty_id, s.title, COALESCE(g.name, '') as genre_name, c.const
		FROM charts c
		INNER JOIN songs s ON c.song_id = s.id
		LEFT JOIN genres g ON s.genre_id = g.id
		WHERE s.is_deleted = 0
	`
	if err := c.workspace.DB().SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to build workspace chart map: %w", err)
	}

	result := make(map[string]mainframeChartMatch, len(rows))
	for _, row := range rows {
		key := fmt.Sprintf("%s|%s|%d", normalizer(row.Title), normalizer(row.GenreName), row.DifficultyID)
		result[key] = mainframeChartMatch{
			SongID:        row.SongID,
			Title:         row.Title,
			OfficialConst: row.Const,
		}
	}
	return result, nil
}

func newMainframeOfficialMismatchError(conflicts []mainframeConstConflict) error {
	limit := min(len(conflicts), info.SongBatchMainframeConflictReportLimit)
	var b strings.Builder
	for _, conflict := range conflicts[:limit] {
		fmt.Fprintf(&b, "\n- %s [%s]: official=%s, mainframe=%.1f",
			conflict.songTitle, conflict.difficulty, conflict.officialLevel, conflict.mainframeConst)
	}
	if remaining := len(conflicts) - limit; remaining > 0 {
		fmt.Fprintf(&b, "\n- ... and %d more", remaining)
	}
	return fmt.Errorf("%w:%s", errMainframeOfficialMismatch, b.String())
}

func difficultyLabel(id int) string {
	switch difficulty.ID(id) {
	case difficulty.Basic:
		return "BASIC"
	case difficulty.Advanced:
		return "ADVANCED"
	case difficulty.Expert:
		return "EXPERT"
	case difficulty.Master:
		return "MASTER"
	case difficulty.Ultima:
		return "ULTIMA"
	default:
		return strconv.Itoa(id)
	}
}

// normalizer はタイトル文字列を正規化します。
// NFKC正規化を行い、特殊文字を統一し、空白・制御文字・フォーマット文字を削除し、小文字に変換します。
func normalizer(title string) string {
	title = strings.ReplaceAll(title, `\"`, `"`)
	title = strings.ReplaceAll(title, `\'`, `'`)

	normalized := norm.NFKC.String(title)

	// NFKC正規化では引用符や波線が統一されないため、正規化後に置換します。
	normalized = strings.ReplaceAll(normalized, "\u201C", `"`)
	normalized = strings.ReplaceAll(normalized, "\u201D", `"`)
	normalized = strings.ReplaceAll(normalized, "\u2018", `'`)
	normalized = strings.ReplaceAll(normalized, "\u2019", `'`)
	normalized = strings.ReplaceAll(normalized, "\u301C", `~`)
	normalized = strings.ReplaceAll(normalized, "\uFF5E", `~`)

	var builder strings.Builder
	for _, r := range normalized {
		if !unicode.IsSpace(r) && !unicode.IsControl(r) && !unicode.Is(unicode.Cf, r) {
			builder.WriteRune(r)
		}
	}

	return strings.ToLower(builder.String())
}
