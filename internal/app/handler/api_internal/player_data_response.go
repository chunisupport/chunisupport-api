package api_internal

import (
	dto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

// toPlayerDataResponse はUsecaseの結果を公開APIのレスポンスへ変換します。
func toPlayerDataResponse(result *playerdataresult.Result) *dto.PlayerDataResult {
	changes := make([]dto.PlayerDataRecordChange, len(result.Changes))
	for i, change := range result.Changes {
		changes[i] = dto.PlayerDataRecordChange{
			RecordType: change.RecordType, ChangeType: change.ChangeType, Idx: change.Idx,
			Diff: change.Diff, CourseClass: change.CourseClass, Before: toPlayerDataRecordStatePointer(change.Before),
			After: toPlayerDataRecordState(change.After),
		}
	}
	skippedRecords := make([]dto.SkippedRecord, len(result.SkippedRecords))
	for i, skipped := range result.SkippedRecords {
		skippedRecords[i] = dto.SkippedRecord{RecordType: skipped.RecordType, Reason: skipped.Reason, Details: skipped.Details}
	}
	overpowerPercentDiff := toPlayerDataFloat64Diff(result.MetricDiffs.OverpowerPercent)

	return &dto.PlayerDataResult{
		PlayerID: result.PlayerID, AppVersion: result.AppVersion, ImportedAt: result.ImportedAt,
		Profile: dto.PlayerDataProfile{
			PlayerID: result.Profile.PlayerID, Name: result.Profile.Name, Level: result.Profile.Level,
			Rating: result.Profile.Rating, ClassEmblemID: result.Profile.ClassEmblemID,
			ClassEmblemBaseID: result.Profile.ClassEmblemBaseID, LastPlayedAt: result.Profile.LastPlayedAt,
			OverpowerValue: result.Profile.OverpowerValue, OverpowerPercent: result.Profile.OverpowerPercent,
		},
		Summary: dto.PlayerDataSummary{
			Name: result.Summary.Name, Level: result.Summary.Level, Rating: result.Summary.Rating,
			LastPlayedAt: result.Summary.LastPlayedAt, OverpowerValue: result.Summary.OverpowerValue,
			OverpowerPercent: result.Summary.OverpowerPercent,
		},
		MetricDiffs: dto.PlayerDataMetricDiffs{
			Rating:           toPlayerDataFloat64Diff(result.MetricDiffs.Rating),
			OverpowerValue:   toPlayerDataFloat64Diff(result.MetricDiffs.OverpowerValue),
			OverpowerPercent: &overpowerPercentDiff,
		},
		Statistics: toPlayerDataStatistics(result.Statistics), Counts: dto.PlayerDataCounts(result.Counts),
		Changes: changes, SkippedRecords: skippedRecords,
	}
}

// toPlayerDataFloat64Diff はユースケースの小数差分をAPIレスポンスへ変換します。
func toPlayerDataFloat64Diff(diff playerdataresult.Float64Diff) dto.PlayerDataFloat64Diff {
	return dto.PlayerDataFloat64Diff{Before: diff.Before, After: diff.After, Delta: diff.Delta}
}

func toPlayerDataStatistics(statistics playerdataresult.Statistics) dto.PlayerDataStatistics {
	byDifficulty := make(map[string]dto.PlayerDataStatisticsGroup, len(statistics.ByDifficulty))
	for difficulty, group := range statistics.ByDifficulty {
		byDifficulty[difficulty] = toPlayerDataStatisticsGroup(group)
	}
	return dto.PlayerDataStatistics{Overall: toPlayerDataStatisticsGroup(statistics.Overall), ByDifficulty: byDifficulty}
}

func toPlayerDataStatisticsGroup(group playerdataresult.StatisticsGroup) dto.PlayerDataStatisticsGroup {
	return dto.PlayerDataStatisticsGroup{
		TotalHighScore: dto.PlayerDataInt64Diff(group.TotalHighScore),
		RecordStatistics: dto.PlayerDataRecordStatisticsDiff{
			AJ: dto.PlayerDataIntDiff(group.RecordStatistics.AJ), FC: dto.PlayerDataIntDiff(group.RecordStatistics.FC),
			CLR: dto.PlayerDataIntDiff(group.RecordStatistics.CLR), FCH: dto.PlayerDataIntDiff(group.RecordStatistics.FCH),
			MAX: dto.PlayerDataIntDiff(group.RecordStatistics.MAX), SSSPlus: dto.PlayerDataIntDiff(group.RecordStatistics.SSSPlus),
			SSS: dto.PlayerDataIntDiff(group.RecordStatistics.SSS), SSPlus: dto.PlayerDataIntDiff(group.RecordStatistics.SSPlus),
			SS: dto.PlayerDataIntDiff(group.RecordStatistics.SS), SPlus: dto.PlayerDataIntDiff(group.RecordStatistics.SPlus),
			S: dto.PlayerDataIntDiff(group.RecordStatistics.S),
		},
	}
}

func toPlayerDataRecordState(state playerdataresult.RecordState) dto.PlayerDataRecordState {
	return dto.PlayerDataRecordState{Score: state.Score, ClearLamp: state.ClearLamp, ComboLamp: state.ComboLamp, FullChain: state.FullChain, IsClear: state.IsClear}
}

func toPlayerDataRecordStatePointer(state *playerdataresult.RecordState) *dto.PlayerDataRecordState {
	if state == nil {
		return nil
	}
	converted := toPlayerDataRecordState(*state)
	return &converted
}
