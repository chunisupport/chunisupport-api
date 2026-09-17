package staticdataexport

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// ChartStatsExportResult は監視ログで通常譜面とWORLD'S ENDの欠落を判別できるよう件数を分けて保持します。
type ChartStatsExportResult struct {
	ChartCount          int
	WorldsendChartCount int
}

// ChartStatsExporter は難易度別の公開統計JSONを生成します。
type ChartStatsExporter struct {
	source      domainrepo.ChartStatsExportQueryService
	writer      JSONWriter
	cachePurger CachePurger
	now         func() time.Time
}

// NewChartStatsExporter は公開統計エクスポーターを生成します。
func NewChartStatsExporter(source domainrepo.ChartStatsExportQueryService, writer JSONWriter, cachePurger CachePurger, location *time.Location) *ChartStatsExporter {
	return newChartStatsExporter(source, writer, cachePurger, func() time.Time {
		return time.Now().In(location)
	})
}

func newChartStatsExporter(source domainrepo.ChartStatsExportQueryService, writer JSONWriter, cachePurger CachePurger, now func() time.Time) *ChartStatsExporter {
	return &ChartStatsExporter{source: source, writer: writer, cachePurger: cachePurger, now: now}
}

type chartStatsRankJSON struct {
	Max  int `json:"max"`
	SSSP int `json:"sssp"`
	SSS  int `json:"sss"`
	SSP  int `json:"ssp"`
	SS   int `json:"ss"`
	SP   int `json:"sp"`
	S    int `json:"s"`
	AAAL int `json:"aaal"`
}

type chartStatsComboJSON struct {
	None int `json:"none"`
	FC   int `json:"fc"`
	AJ   int `json:"aj"`
	AJC  int `json:"ajc"`
}

type chartStatsClearJSON struct {
	Failed      int `json:"failed"`
	Clear       int `json:"clear"`
	Hard        int `json:"hard"`
	Brave       int `json:"brave"`
	Absolute    int `json:"absolute"`
	Catastrophy int `json:"catastrophy"`
}

type chartStatsScoreJSON struct {
	RatingBand   string   `json:"rating_band"`
	AverageScore *float64 `json:"average_score"`
	MedianScore  *float64 `json:"median_score"`
}

type chartStatsJSON struct {
	SongID         string              `json:"song_id"`
	Title          string              `json:"title"`
	Const          float64             `json:"const"`
	IsConstUnknown bool                `json:"is_const_unknown"`
	PlayerCount    int                 `json:"player_count"`
	Rank           chartStatsRankJSON  `json:"rank"`
	Clear          chartStatsClearJSON `json:"clear"`
	Combo          chartStatsComboJSON `json:"combo"`
}

type worldsendChartStatsJSON struct {
	SongID      string              `json:"song_id"`
	Title       string              `json:"title"`
	LevelStar   *int                `json:"level_star"`
	Attribute   *string             `json:"attribute"`
	PlayerCount int                 `json:"player_count"`
	Rank        chartStatsRankJSON  `json:"rank"`
	Clear       chartStatsClearJSON `json:"clear"`
	Combo       chartStatsComboJSON `json:"combo"`
}

type chartScoresJSON struct {
	SongID string                `json:"song_id"`
	Scores []chartStatsScoreJSON `json:"scores"`
}

type worldsendChartScoresJSON struct {
	SongID    string                `json:"song_id"`
	LevelStar *int                  `json:"level_star"`
	Attribute *string               `json:"attribute"`
	Scores    []chartStatsScoreJSON `json:"scores"`
}

type chartStatsPayload[T any] struct {
	GeneratedAt string `json:"generated_at"`
	Difficulty  string `json:"difficulty"`
	RatingBand  string `json:"rating_band"`
	Charts      []T    `json:"charts"`
}

type chartScoresPayload[T any] struct {
	GeneratedAt string `json:"generated_at"`
	Difficulty  string `json:"difficulty"`
	Charts      []T    `json:"charts"`
}

// Export は全JSONを生成してからPUTし、すべて成功した場合だけキャッシュをパージします。
func (e *ChartStatsExporter) Export(ctx context.Context) (ChartStatsExportResult, error) {
	snapshot, err := e.source.Get(ctx)
	if err != nil {
		return ChartStatsExportResult{}, fmt.Errorf("failed to get chart stats export snapshot: %w", err)
	}
	if snapshot == nil || len(snapshot.Charts) == 0 {
		return ChartStatsExportResult{}, fmt.Errorf("refusing to export an empty chart stats snapshot")
	}
	if len(snapshot.WorldsendCharts) == 0 {
		return ChartStatsExportResult{}, fmt.Errorf("refusing to export an empty worldsend chart stats snapshot")
	}

	generatedAt := e.now().Format(time.RFC3339)
	difficulties := info.ChartStatsDifficulties()
	chartsByDifficulty := make(map[string][]chartStatsJSON, len(difficulties))
	scoresByDifficulty := make(map[string][]chartScoresJSON, len(difficulties))
	for _, difficulty := range difficulties {
		chartsByDifficulty[difficulty] = []chartStatsJSON{}
		scoresByDifficulty[difficulty] = []chartScoresJSON{}
	}
	for _, chart := range snapshot.Charts {
		charts, ok := chartsByDifficulty[chart.Difficulty]
		if !ok {
			return ChartStatsExportResult{}, fmt.Errorf("unknown chart difficulty for export: %s", chart.Difficulty)
		}
		chartsByDifficulty[chart.Difficulty] = append(charts, chartStatsJSON{
			SongID: chart.SongDisplayID, Title: chart.SongTitle, Const: chart.ChartConst.Float64(),
			IsConstUnknown: chart.IsConstUnknown, PlayerCount: chart.PlayerCount,
			Rank: rankJSON(chart.Rank), Clear: clearJSON(chart.Clear), Combo: comboJSON(chart.Combo),
		})
		scoresByDifficulty[chart.Difficulty] = append(scoresByDifficulty[chart.Difficulty], chartScoresJSON{
			SongID: chart.SongDisplayID, Scores: scoresJSON(chart.Scores),
		})
	}

	type object struct {
		key  string
		body []byte
	}
	objects := make([]object, 0, 2*(len(difficulties)+1))
	for _, difficulty := range difficulties {
		body, err := json.Marshal(chartStatsPayload[chartStatsJSON]{
			GeneratedAt: generatedAt,
			Difficulty:  difficulty,
			RatingBand:  info.AllRatingBandLabel,
			Charts:      chartsByDifficulty[difficulty],
		})
		if err != nil {
			return ChartStatsExportResult{}, fmt.Errorf("failed to marshal %s chart stats snapshot: %w", difficulty, err)
		}
		objects = append(objects, object{key: info.ChartStatsSnapshotObjectKey(difficulty), body: body})
		scoreBody, err := json.Marshal(chartScoresPayload[chartScoresJSON]{
			GeneratedAt: generatedAt,
			Difficulty:  difficulty,
			Charts:      scoresByDifficulty[difficulty],
		})
		if err != nil {
			return ChartStatsExportResult{}, fmt.Errorf("failed to marshal %s chart scores snapshot: %w", difficulty, err)
		}
		objects = append(objects, object{key: info.ChartScoresSnapshotObjectKey(difficulty), body: scoreBody})
	}

	worldsendCharts := make([]worldsendChartStatsJSON, 0, len(snapshot.WorldsendCharts))
	worldsendScores := make([]worldsendChartScoresJSON, 0, len(snapshot.WorldsendCharts))
	for _, chart := range snapshot.WorldsendCharts {
		worldsendCharts = append(worldsendCharts, worldsendChartStatsJSON{
			SongID: chart.SongDisplayID, Title: chart.SongTitle, LevelStar: chart.LevelStar,
			Attribute: chart.Attribute, PlayerCount: chart.PlayerCount,
			Rank: rankJSON(chart.Rank), Clear: clearJSON(chart.Clear), Combo: comboJSON(chart.Combo),
		})
		worldsendScores = append(worldsendScores, worldsendChartScoresJSON{
			SongID: chart.SongDisplayID, LevelStar: chart.LevelStar,
			Attribute: chart.Attribute, Scores: scoresJSON(chart.Scores),
		})
	}
	worldsendBody, err := json.Marshal(chartStatsPayload[worldsendChartStatsJSON]{
		GeneratedAt: generatedAt,
		Difficulty:  info.StatsDifficultyWorldsend,
		RatingBand:  info.AllRatingBandLabel,
		Charts:      worldsendCharts,
	})
	if err != nil {
		return ChartStatsExportResult{}, fmt.Errorf("failed to marshal worldsend chart stats snapshot: %w", err)
	}
	objects = append(objects, object{key: info.WorldsendChartStatsSnapshotObjectKey, body: worldsendBody})
	worldsendScoreBody, err := json.Marshal(chartScoresPayload[worldsendChartScoresJSON]{
		GeneratedAt: generatedAt,
		Difficulty:  info.StatsDifficultyWorldsend,
		Charts:      worldsendScores,
	})
	if err != nil {
		return ChartStatsExportResult{}, fmt.Errorf("failed to marshal worldsend chart scores snapshot: %w", err)
	}
	objects = append(objects, object{key: info.WorldsendChartScoresSnapshotObjectKey, body: worldsendScoreBody})

	objectKeys := make([]string, 0, len(objects))
	for _, item := range objects {
		objectKeys = append(objectKeys, item.key)
		if err := e.writer.PutJSON(ctx, item.key, item.body); err != nil {
			return ChartStatsExportResult{}, fmt.Errorf("failed to upload %s: %w", item.key, err)
		}
	}
	if err := e.cachePurger.Purge(ctx, objectKeys); err != nil {
		return ChartStatsExportResult{}, fmt.Errorf("failed to purge chart stats cache: %w", err)
	}

	return ChartStatsExportResult{ChartCount: len(snapshot.Charts), WorldsendChartCount: len(snapshot.WorldsendCharts)}, nil
}

func rankJSON(rank domainrepo.ChartStatsExportRank) chartStatsRankJSON {
	return chartStatsRankJSON{Max: rank.Max, SSSP: rank.SSSP, SSS: rank.SSS, SSP: rank.SSP, SS: rank.SS, SP: rank.SP, S: rank.S, AAAL: rank.AAAL}
}

func comboJSON(combo domainrepo.ChartStatsExportCombo) chartStatsComboJSON {
	return chartStatsComboJSON{None: combo.None, FC: combo.FC, AJ: combo.AJ, AJC: combo.AJC}
}

func clearJSON(clear domainrepo.ChartStatsExportClear) chartStatsClearJSON {
	return chartStatsClearJSON{
		Failed: clear.Failed, Clear: clear.Clear, Hard: clear.Hard,
		Brave: clear.Brave, Absolute: clear.Absolute, Catastrophy: clear.Catastrophy,
	}
}

func scoresJSON(scores []domainrepo.ChartStatsExportScore) []chartStatsScoreJSON {
	result := make([]chartStatsScoreJSON, 0, len(scores))
	for _, score := range scores {
		result = append(result, chartStatsScoreJSON{
			RatingBand: score.RatingBand, AverageScore: score.AverageScore, MedianScore: score.MedianScore,
		})
	}
	return result
}
