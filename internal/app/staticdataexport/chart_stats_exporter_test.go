package staticdataexport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubChartStatsSnapshotSource struct {
	snapshot *domainrepo.ChartStatsExportSnapshot
	err      error
}

func (s *stubChartStatsSnapshotSource) Get(context.Context) (*domainrepo.ChartStatsExportSnapshot, error) {
	return s.snapshot, s.err
}

func TestChartStatsExporterExport_難易度ごとのJSONをアップロードする(t *testing.T) {
	// Given
	chartConst, err := chartconstant.NewChartConstant(12.7)
	require.NoError(t, err)
	levelStar := 4
	attribute := "狂"
	averageScore := 1007000.5
	medianScore := 1008000.0
	source := &stubChartStatsSnapshotSource{snapshot: &domainrepo.ChartStatsExportSnapshot{
		Charts: []domainrepo.ChartStatsExportItem{{
			SongDisplayID: "0123456789abcdef", SongTitle: "通常楽曲", Difficulty: "MASTER",
			ChartConst: chartConst, IsConstUnknown: true, PlayerCount: 10,
			Scores: []domainrepo.ChartStatsExportScore{{RatingBand: "ALL", AverageScore: &averageScore, MedianScore: &medianScore}, {RatingBand: "15.0"}},
			Rank:   domainrepo.ChartStatsExportRank{Max: 1, SSSP: 2},
			Clear:  domainrepo.ChartStatsExportClear{Failed: 4, Clear: 3, Hard: 2, Brave: 1},
			Combo:  domainrepo.ChartStatsExportCombo{FC: 3, AJ: 2, AJC: 1},
		}},
		WorldsendCharts: []domainrepo.WorldsendChartStatsExportItem{{
			SongDisplayID: "fedcba9876543210", SongTitle: "WE楽曲", LevelStar: &levelStar,
			Attribute: &attribute, PlayerCount: 5,
			Scores: []domainrepo.ChartStatsExportScore{{RatingBand: "ALL", AverageScore: &averageScore, MedianScore: &medianScore}},
			Clear:  domainrepo.ChartStatsExportClear{Absolute: 2, Catastrophy: 1},
		}},
	}}
	writer := &recordingWriter{}
	purger := &recordingCachePurger{}
	exporter := newChartStatsExporter(source, writer, purger, func() time.Time {
		return time.Date(2026, 9, 2, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	})

	// When
	result, err := exporter.Export(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, ChartStatsExportResult{ChartCount: 1, WorldsendChartCount: 1}, result)
	assert.Len(t, writer.objects, 12)
	assert.ElementsMatch(t, append(info.ChartStatsSnapshotObjectKeys(), info.ChartScoresSnapshotObjectKeys()...), purger.objectKeys)

	var masterPayload struct {
		GeneratedAt string `json:"generated_at"`
		Difficulty  string `json:"difficulty"`
		RatingBand  string `json:"rating_band"`
		Charts      []struct {
			SongID         string  `json:"song_id"`
			Title          string  `json:"title"`
			Const          float64 `json:"const"`
			IsConstUnknown bool    `json:"is_const_unknown"`
			PlayerCount    int     `json:"player_count"`
			Rank           struct {
				Max  int `json:"max"`
				SSSP int `json:"sssp"`
			} `json:"rank"`
			Clear struct {
				Failed int `json:"failed"`
				Hard   int `json:"hard"`
				Brave  int `json:"brave"`
			} `json:"clear"`
			Combo struct {
				FC  int `json:"fc"`
				AJ  int `json:"aj"`
				AJC int `json:"ajc"`
			} `json:"combo"`
		} `json:"charts"`
	}
	require.NoError(t, json.Unmarshal(writer.objects[info.MasterChartStatsSnapshotObjectKey], &masterPayload))
	assert.Equal(t, "2026-09-02T12:00:00+09:00", masterPayload.GeneratedAt)
	assert.Equal(t, "MASTER", masterPayload.Difficulty)
	assert.Equal(t, "ALL", masterPayload.RatingBand)
	require.Len(t, masterPayload.Charts, 1)
	assert.Equal(t, "0123456789abcdef", masterPayload.Charts[0].SongID)
	assert.Equal(t, 12.7, masterPayload.Charts[0].Const)
	assert.True(t, masterPayload.Charts[0].IsConstUnknown)
	assert.Equal(t, 1, masterPayload.Charts[0].Rank.Max)
	assert.Equal(t, 2, masterPayload.Charts[0].Rank.SSSP)
	assert.Equal(t, 4, masterPayload.Charts[0].Clear.Failed)
	assert.Equal(t, 2, masterPayload.Charts[0].Clear.Hard)
	assert.Equal(t, 1, masterPayload.Charts[0].Clear.Brave)
	assert.Equal(t, 3, masterPayload.Charts[0].Combo.FC)

	var masterRaw map[string]any
	require.NoError(t, json.Unmarshal(writer.objects[info.MasterChartStatsSnapshotObjectKey], &masterRaw))
	assert.ElementsMatch(t, []string{"generated_at", "difficulty", "rating_band", "charts"}, mapKeys(masterRaw))
	masterChartRaw := masterRaw["charts"].([]any)[0].(map[string]any)
	assert.ElementsMatch(t, []string{"song_id", "title", "const", "is_const_unknown", "player_count", "rank", "clear", "combo"}, mapKeys(masterChartRaw))
	assert.ElementsMatch(t, []string{"max", "sssp", "sss", "ssp", "ss", "sp", "s", "aaal"}, mapKeys(masterChartRaw["rank"].(map[string]any)))
	assert.ElementsMatch(t, []string{"failed", "clear", "hard", "brave", "absolute", "catastrophy"}, mapKeys(masterChartRaw["clear"].(map[string]any)))
	assert.ElementsMatch(t, []string{"none", "fc", "aj", "ajc"}, mapKeys(masterChartRaw["combo"].(map[string]any)))

	var basicRaw map[string]any
	require.NoError(t, json.Unmarshal(writer.objects[info.BasicChartStatsSnapshotObjectKey], &basicRaw))
	assert.Empty(t, basicRaw["charts"].([]any))

	var worldsendPayload struct {
		Difficulty string `json:"difficulty"`
		Charts     []struct {
			LevelStar *int    `json:"level_star"`
			Attribute *string `json:"attribute"`
			Clear     struct {
				Absolute    int `json:"absolute"`
				Catastrophy int `json:"catastrophy"`
			} `json:"clear"`
		} `json:"charts"`
	}
	require.NoError(t, json.Unmarshal(writer.objects[info.WorldsendChartStatsSnapshotObjectKey], &worldsendPayload))
	assert.Equal(t, "WORLD'S END", worldsendPayload.Difficulty)
	require.Len(t, worldsendPayload.Charts, 1)
	assert.Equal(t, 4, *worldsendPayload.Charts[0].LevelStar)
	assert.Equal(t, "狂", *worldsendPayload.Charts[0].Attribute)
	assert.Equal(t, 2, worldsendPayload.Charts[0].Clear.Absolute)
	assert.Equal(t, 1, worldsendPayload.Charts[0].Clear.Catastrophy)

	var worldsendRaw map[string]any
	require.NoError(t, json.Unmarshal(writer.objects[info.WorldsendChartStatsSnapshotObjectKey], &worldsendRaw))
	worldsendChartRaw := worldsendRaw["charts"].([]any)[0].(map[string]any)
	assert.ElementsMatch(t, []string{"song_id", "title", "level_star", "attribute", "player_count", "rank", "clear", "combo"}, mapKeys(worldsendChartRaw))

	var scorePayload struct {
		GeneratedAt string `json:"generated_at"`
		Difficulty  string `json:"difficulty"`
		Charts      []struct {
			SongID string `json:"song_id"`
			Scores []struct {
				RatingBand   string   `json:"rating_band"`
				AverageScore *float64 `json:"average_score"`
				MedianScore  *float64 `json:"median_score"`
			} `json:"scores"`
		} `json:"charts"`
	}
	require.NoError(t, json.Unmarshal(writer.objects[info.MasterChartScoresSnapshotObjectKey], &scorePayload))
	assert.Equal(t, "2026-09-02T12:00:00+09:00", scorePayload.GeneratedAt)
	assert.Equal(t, "MASTER", scorePayload.Difficulty)
	require.Len(t, scorePayload.Charts, 1)
	assert.Equal(t, "0123456789abcdef", scorePayload.Charts[0].SongID)
	require.Len(t, scorePayload.Charts[0].Scores, 2)
	assert.Equal(t, "ALL", scorePayload.Charts[0].Scores[0].RatingBand)
	assert.Equal(t, averageScore, *scorePayload.Charts[0].Scores[0].AverageScore)
	assert.Equal(t, medianScore, *scorePayload.Charts[0].Scores[0].MedianScore)
	assert.Equal(t, "15.0", scorePayload.Charts[0].Scores[1].RatingBand)
	assert.Nil(t, scorePayload.Charts[0].Scores[1].AverageScore)
	assert.Nil(t, scorePayload.Charts[0].Scores[1].MedianScore)
	var scoreRaw map[string]any
	require.NoError(t, json.Unmarshal(writer.objects[info.MasterChartScoresSnapshotObjectKey], &scoreRaw))
	assert.ElementsMatch(t, []string{"generated_at", "difficulty", "charts"}, mapKeys(scoreRaw))
	assert.ElementsMatch(t, []string{"song_id", "scores"}, mapKeys(scoreRaw["charts"].([]any)[0].(map[string]any)))
	assert.ElementsMatch(t, []string{"rating_band", "average_score", "median_score"}, mapKeys(scoreRaw["charts"].([]any)[0].(map[string]any)["scores"].([]any)[0].(map[string]any)))

	var worldsendScoreRaw map[string]any
	require.NoError(t, json.Unmarshal(writer.objects[info.WorldsendChartScoresSnapshotObjectKey], &worldsendScoreRaw))
	assert.ElementsMatch(t, []string{"generated_at", "difficulty", "charts"}, mapKeys(worldsendScoreRaw))
	assert.Equal(t, "WORLD'S END", worldsendScoreRaw["difficulty"])
	assert.ElementsMatch(t, []string{"song_id", "level_star", "attribute", "scores"}, mapKeys(worldsendScoreRaw["charts"].([]any)[0].(map[string]any)))
}

func TestChartStatsExporterExport_取得失敗または空譜面をアップロードしない(t *testing.T) {
	tests := []struct {
		name   string
		source *stubChartStatsSnapshotSource
	}{
		{name: "取得失敗", source: &stubChartStatsSnapshotSource{err: errors.New("query failed")}},
		{name: "通常譜面が空", source: &stubChartStatsSnapshotSource{snapshot: &domainrepo.ChartStatsExportSnapshot{
			WorldsendCharts: []domainrepo.WorldsendChartStatsExportItem{{SongDisplayID: "1"}},
		}}},
		{name: "WORLD'S END譜面が空", source: &stubChartStatsSnapshotSource{snapshot: &domainrepo.ChartStatsExportSnapshot{
			Charts: []domainrepo.ChartStatsExportItem{{SongDisplayID: "1", Difficulty: "MASTER"}},
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			writer := &recordingWriter{}
			exporter := NewChartStatsExporter(tt.source, writer, &recordingCachePurger{}, time.UTC)

			// When
			_, err := exporter.Export(context.Background())

			// Then
			assert.Error(t, err)
			assert.Empty(t, writer.objects)
		})
	}
}

func TestChartStatsExporterExport_アップロード失敗時はパージしない(t *testing.T) {
	// Given
	source := &stubChartStatsSnapshotSource{snapshot: &domainrepo.ChartStatsExportSnapshot{
		Charts: []domainrepo.ChartStatsExportItem{
			{SongDisplayID: "1", Difficulty: "BASIC"},
			{SongDisplayID: "2", Difficulty: "ADVANCED"},
			{SongDisplayID: "3", Difficulty: "EXPERT"},
			{SongDisplayID: "4", Difficulty: "MASTER"},
			{SongDisplayID: "5", Difficulty: "ULTIMA"},
		},
		WorldsendCharts: []domainrepo.WorldsendChartStatsExportItem{{SongDisplayID: "6"}},
	}}
	writer := &recordingWriter{failOnKey: info.ExpertChartScoresSnapshotObjectKey}
	purger := &recordingCachePurger{}
	exporter := NewChartStatsExporter(source, writer, purger, time.UTC)

	// When
	_, err := exporter.Export(context.Background())

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), info.ExpertChartScoresSnapshotObjectKey)
	assert.Empty(t, purger.objectKeys)
}

func TestChartStatsExporterExport_未知の難易度ではアップロードしない(t *testing.T) {
	// Given
	writer := &recordingWriter{}
	exporter := NewChartStatsExporter(&stubChartStatsSnapshotSource{snapshot: &domainrepo.ChartStatsExportSnapshot{
		Charts:          []domainrepo.ChartStatsExportItem{{SongDisplayID: "1", Difficulty: "UNKNOWN"}},
		WorldsendCharts: []domainrepo.WorldsendChartStatsExportItem{{SongDisplayID: "2"}},
	}}, writer, &recordingCachePurger{}, time.UTC)

	// When
	_, err := exporter.Export(context.Background())

	// Then
	require.Error(t, err)
	assert.Empty(t, writer.objects)
}

func TestChartStatsExporterExport_パージ失敗を返す(t *testing.T) {
	// Given
	purger := &recordingCachePurger{err: errors.New("purge failed")}
	exporter := NewChartStatsExporter(&stubChartStatsSnapshotSource{snapshot: &domainrepo.ChartStatsExportSnapshot{
		Charts:          []domainrepo.ChartStatsExportItem{{SongDisplayID: "1", Difficulty: "MASTER"}},
		WorldsendCharts: []domainrepo.WorldsendChartStatsExportItem{{SongDisplayID: "2"}},
	}}, &recordingWriter{}, purger, time.UTC)

	// When
	_, err := exporter.Export(context.Background())

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to purge chart stats cache")
	assert.ElementsMatch(t, append(info.ChartStatsSnapshotObjectKeys(), info.ChartScoresSnapshotObjectKeys()...), purger.objectKeys)
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
