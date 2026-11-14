package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

type calculatedOverpowerSummary struct {
	Value             *float64
	Percent           *float64
	MaxOverpowerTotal float64
}

func (us *playerDataUsecase) listLockedSongsForOverpower(ctx context.Context, tx repository.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	if us.lockedRepo == nil {
		return nil, nil
	}
	return us.lockedRepo.ListByPlayerID(ctx, tx, playerID)
}

func calculateOverpowerSummaryFromPlayerRecords(ctx context.Context, records []*entity.PlayerRecord, lockedSongs []*entity.PlayerLockedSong, maxOverpowerTotal float64) (calculatedOverpowerSummary, error) {
	lockedSet := service.NewPlayerLockedSongSet(lockedSongs)
	overpowerRecords, skippedRecords, err := playerRecordsToOverpowerRecordsWithSkipped(records, false, func(record *entity.PlayerRecord) (bool, string) {
		if lockedSet.IsEmpty() {
			return true, ""
		}
		if record.ChartDifficulty == nil {
			return false, "chart_difficulty_nil"
		}
		isUltima := record.ChartDifficulty.Name == info.DifficultyNameUltima
		if lockedSet.IsLocked(record.Song.ID, isUltima) {
			return false, "locked_song"
		}
		return true, ""
	})
	if err != nil {
		return calculatedOverpowerSummary{}, err
	}
	unexpectedSkippedRecords := make([]skippedOverpowerRecord, 0, len(skippedRecords))
	for _, skippedRecord := range skippedRecords {
		if skippedRecord.Reason != "locked_song" {
			unexpectedSkippedRecords = append(unexpectedSkippedRecords, skippedRecord)
		}
	}
	if len(unexpectedSkippedRecords) > 0 {
		slog.WarnContext(ctx, "skipped player records during overpower recalculation", "total_records", len(records), "aggregated_records", len(overpowerRecords), "skipped_records", unexpectedSkippedRecords)
	}
	value, percent := service.CalcOverpowerSummary(overpowerRecords, maxOverpowerTotal)
	return calculatedOverpowerSummary{Value: &value, Percent: &percent, MaxOverpowerTotal: maxOverpowerTotal}, nil
}

// calculateAndUpdateRatings はプレイヤーのレーティングを再計算してDBに保存します。
// ベスト枠30曲 + 新曲枠20曲から計算したレーティングを保存します。
func (us *playerDataUsecase) calculateAndUpdateRatings(ctx context.Context, tx repository.Executor, playerID int) (service.RatingStats, error) {
	player, err := us.playerRepo.FindByIDForUpdate(ctx, tx, playerID)
	if err != nil {
		return service.RatingStats{}, err
	}

	records, err := us.playerRecRepo.FindByPlayerIDForRating(ctx, tx, playerID)
	if err != nil {
		return service.RatingStats{}, fmt.Errorf("failed to fetch player records: %w", err)
	}

	bestRecords := make([]service.RatingSlotRecord, 0, 30)
	newRecords := make([]service.RatingSlotRecord, 0, 20)
	for _, rec := range records {
		if rec.Chart == nil || rec.Slot == nil {
			return service.RatingStats{}, fmt.Errorf("rating record relation is missing: chart_id=%d", rec.ChartID)
		}
		ratingRecord := service.RatingSlotRecord{ChartID: rec.ChartID, Score: uint32(rec.Score), ChartConst: rec.Chart.Const.Float64()} // #nosec G115
		switch rec.Slot.Name {
		case "best":
			bestRecords = append(bestRecords, ratingRecord)
		case "new":
			newRecords = append(newRecords, ratingRecord)
		}
	}

	stats := service.AggregateOfficialRating(bestRecords, newRecords)

	player.ChangeCalculatedRatings(stats.PlayerRating, stats.BestAverage, stats.NewAverage)
	if err := us.playerRepo.Save(ctx, tx, player); err != nil {
		return service.RatingStats{}, err
	}

	return stats, nil
}
