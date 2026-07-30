package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

const playerLatestUpdateSchemaVersion = info.PlayerLatestUpdateSchemaVersion

// buildPlayerLatestUpdate は登録結果からスキップ詳細を除外し、永続化用の圧縮データを生成します。
func buildPlayerLatestUpdate(result *playerdataresult.Result, sourceUpdatedAt time.Time, bodyHash string) (*entity.PlayerLatestUpdate, error) {
	if result == nil {
		return nil, fmt.Errorf("player data result is nil")
	}

	raw, err := json.Marshal(playerLatestUpdatePayload(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal player latest update: %w", err)
	}

	compressed, err := gzipBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to compress player latest update: %w", err)
	}

	update, err := entity.NewPlayerLatestUpdate(
		result.PlayerID,
		playerLatestUpdateSchemaVersion,
		compressed,
		sourceUpdatedAt,
		result.ImportedAt,
		bodyHash,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create player latest update: %w", err)
	}
	return update, nil
}

// GetLatestUpdate はプレイヤーの最新データ登録結果を展開して返します。
func (us *playerDataUsecase) GetLatestUpdate(ctx context.Context, user *entity.User) (json.RawMessage, error) {
	if user == nil || !user.HasLinkedPlayer() {
		return nil, ErrPlayerNotLinked
	}

	update, err := us.playerDataRepo.FindLatestUpdateByPlayerID(ctx, *user.PlayerID)
	if err != nil {
		if errors.Is(err, repository.ErrPlayerLatestUpdateNotFound) {
			return nil, ErrPlayerLatestUpdateNotFound
		}
		return nil, err
	}

	raw, err := gunzipBytes(update.ResultGzip(), info.PlayerLatestUpdateMaxPayloadBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decompress player latest update: %v", ErrInternalError, err)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("%w: failed to decode player latest update: %v", ErrInternalError, err)
	}
	var schemaVersion int
	if err := json.Unmarshal(envelope["schema_version"], &schemaVersion); err != nil ||
		schemaVersion != update.SchemaVersion() ||
		schemaVersion < info.PlayerLatestUpdateMinSupportedSchemaVersion ||
		schemaVersion > playerLatestUpdateSchemaVersion {
		return nil, fmt.Errorf("%w: unsupported player latest update schema", ErrInternalError)
	}
	requiredFields := [...]string{"player_id", "app_ver", "imported_at", "profile", "summary", "statistics", "counts", "changes"}
	for _, field := range requiredFields {
		if len(envelope[field]) == 0 {
			return nil, fmt.Errorf("%w: player latest update field is missing: %s", ErrInternalError, field)
		}
	}
	if schemaVersion >= info.PlayerLatestUpdateMetricDiffSchemaVersion && len(envelope["metric_diffs"]) == 0 {
		return nil, fmt.Errorf("%w: player latest update field is missing: metric_diffs", ErrInternalError)
	}
	var playerID int
	if err := json.Unmarshal(envelope["player_id"], &playerID); err != nil || playerID != update.PlayerID() || playerID != *user.PlayerID {
		return nil, fmt.Errorf("%w: player latest update player_id is invalid", ErrInternalError)
	}

	return json.RawMessage(append([]byte(nil), raw...)), nil
}

func playerLatestUpdatePayload(result *playerdataresult.Result) map[string]any {
	changes := make([]map[string]any, 0, len(result.Changes))
	for _, change := range result.Changes {
		changes = append(changes, playerLatestUpdateChangePayload(change))
	}

	return map[string]any{
		"schema_version": playerLatestUpdateSchemaVersion,
		"player_id":      result.PlayerID,
		"app_ver":        result.AppVersion,
		"imported_at":    result.ImportedAt,
		"profile": map[string]any{
			"player_id":            result.Profile.PlayerID,
			"name":                 result.Profile.Name,
			"level":                result.Profile.Level,
			"rating":               result.Profile.Rating,
			"class_emblem_id":      result.Profile.ClassEmblemID,
			"class_emblem_base_id": result.Profile.ClassEmblemBaseID,
			"last_played_at":       result.Profile.LastPlayedAt,
			"overpower_value":      result.Profile.OverpowerValue,
			"overpower_percent":    result.Profile.OverpowerPercent,
		},
		"summary": map[string]any{
			"name":                 result.Summary.Name,
			"level":                result.Summary.Level,
			"rating":               result.Summary.Rating,
			"last_played_at":       result.Summary.LastPlayedAt,
			"overpower_value":      result.Summary.OverpowerValue,
			"overpower_percentage": result.Summary.OverpowerPercent,
		},
		"metric_diffs": map[string]any{
			"rating":          playerLatestUpdateFloat64DiffPayload(result.MetricDiffs.Rating),
			"overpower_value": playerLatestUpdateFloat64DiffPayload(result.MetricDiffs.OverpowerValue),
		},
		"statistics": playerLatestUpdateStatisticsPayload(result.Statistics),
		"counts":     playerLatestUpdateCountsPayload(result.Counts),
		"changes":    changes,
	}
}

func playerLatestUpdateFloat64DiffPayload(diff playerdataresult.Float64Diff) map[string]*float64 {
	return map[string]*float64{"before": diff.Before, "after": diff.After, "delta": diff.Delta}
}

func playerLatestUpdateStatisticsPayload(statistics playerdataresult.Statistics) map[string]any {
	byDifficulty := make(map[string]any, len(statistics.ByDifficulty))
	for difficulty, group := range statistics.ByDifficulty {
		byDifficulty[difficulty] = playerLatestUpdateStatisticsGroupPayload(group)
	}
	return map[string]any{
		"overall":       playerLatestUpdateStatisticsGroupPayload(statistics.Overall),
		"by_difficulty": byDifficulty,
	}
}

func playerLatestUpdateStatisticsGroupPayload(group playerdataresult.StatisticsGroup) map[string]any {
	stats := group.RecordStatistics
	return map[string]any{
		"total_high_score": playerLatestUpdateInt64DiffPayload(group.TotalHighScore),
		"record_statistics": map[string]any{
			"aj":       playerLatestUpdateIntDiffPayload(stats.AJ),
			"fc":       playerLatestUpdateIntDiffPayload(stats.FC),
			"clr":      playerLatestUpdateIntDiffPayload(stats.CLR),
			"fch":      playerLatestUpdateIntDiffPayload(stats.FCH),
			"max":      playerLatestUpdateIntDiffPayload(stats.MAX),
			"sss_plus": playerLatestUpdateIntDiffPayload(stats.SSSPlus),
			"sss":      playerLatestUpdateIntDiffPayload(stats.SSS),
			"ss_plus":  playerLatestUpdateIntDiffPayload(stats.SSPlus),
			"ss":       playerLatestUpdateIntDiffPayload(stats.SS),
			"s_plus":   playerLatestUpdateIntDiffPayload(stats.SPlus),
			"s":        playerLatestUpdateIntDiffPayload(stats.S),
		},
	}
}

func playerLatestUpdateInt64DiffPayload(diff playerdataresult.Int64Diff) map[string]int64 {
	return map[string]int64{"before": diff.Before, "after": diff.After, "delta": diff.Delta}
}

func playerLatestUpdateIntDiffPayload(diff playerdataresult.IntDiff) map[string]int {
	return map[string]int{"before": diff.Before, "after": diff.After, "delta": diff.Delta}
}

func playerLatestUpdateCountsPayload(counts playerdataresult.Counts) map[string]int {
	return map[string]int{
		"standard_records_upserted":          counts.FullRecordsUpserted,
		"worldsend_records_upserted":         counts.WorldsendRecordsUpserted,
		"standard_records_skipped":           counts.FullRecordsSkipped,
		"worldsend_records_skipped":          counts.WorldsendRecordsSkipped,
		"honors_skipped":                     counts.HonorsSkipped,
		"standard_records_actually_changed":  counts.FullRecordsActuallyChanged,
		"worldsend_records_actually_changed": counts.WorldsendRecordsActuallyChanged,
		"course_records_upserted":            counts.CourseRecordsUpserted,
		"course_records_skipped":             counts.CourseRecordsSkipped,
		"course_records_actually_changed":    counts.CourseRecordsActuallyChanged,
	}
}

func playerLatestUpdateChangePayload(change playerdataresult.RecordChange) map[string]any {
	payload := map[string]any{
		"record_type": change.RecordType,
		"change_type": change.ChangeType,
		"idx":         change.Idx,
		"before":      nil,
		"after":       playerLatestUpdateRecordStatePayload(change.After),
	}
	if change.Diff != "" {
		payload["diff"] = change.Diff
	}
	if change.CourseClass != "" {
		payload["course_class"] = change.CourseClass
	}
	if change.Before != nil {
		payload["before"] = playerLatestUpdateRecordStatePayload(*change.Before)
	}
	return payload
}

func playerLatestUpdateRecordStatePayload(state playerdataresult.RecordState) map[string]any {
	payload := map[string]any{
		"score":      state.Score,
		"clear_lamp": state.ClearLamp,
		"combo_lamp": state.ComboLamp,
		"full_chain": state.FullChain,
	}
	if state.IsClear != nil {
		payload["is_clear"] = state.IsClear
	}
	return payload
}
