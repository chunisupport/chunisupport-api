package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/scorepercentile"
)

// loadScorePercentiles は上位%スナップショットを取得します。
// Provider が未設定、または一時的な取得失敗時は nil を返し、レコード取得自体は継続させます。
func loadScorePercentiles(ctx context.Context, provider repository.ScorePercentileProvider) *scorepercentile.Snapshot {
	if provider == nil {
		return nil
	}
	snapshot, err := provider.Snapshot(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to load score percentiles snapshot", "error", err)
		}
		return nil
	}
	return snapshot
}

func applyPlayerRecordTopPercents(outputs []*PlayerRecordOutput, snapshot *scorepercentile.Snapshot) {
	if snapshot == nil {
		return
	}
	for _, out := range outputs {
		if out == nil || !out.IsPlayed || out.PlayerRecord == nil {
			continue
		}
		chartID := playerRecordChartID(out.PlayerRecord)
		if chartID == 0 {
			continue
		}
		out.TopPercent = snapshot.LookupChart(chartID, uint32(out.PlayerRecord.Score))
	}
}

func applyWorldsendRecordTopPercents(outputs []*WorldsendRecordOutput, snapshot *scorepercentile.Snapshot) {
	if snapshot == nil {
		return
	}
	for _, out := range outputs {
		if out == nil || !out.IsPlayed || out.PlayerWorldsendRecord == nil {
			continue
		}
		chartID := worldsendRecordChartID(out.PlayerWorldsendRecord)
		if chartID == 0 {
			continue
		}
		out.TopPercent = snapshot.LookupWorldsendChart(chartID, uint32(out.PlayerWorldsendRecord.Score))
	}
}

func worldsendRecordChartID(record *entity.PlayerWorldsendRecord) int {
	if record == nil {
		return 0
	}
	if record.WorldsendChartID != 0 {
		return record.WorldsendChartID
	}
	if record.WorldsendChart != nil {
		return record.WorldsendChart.ID
	}
	return 0
}

// toCourseRecordOutputWithTopPercent はコース記録を出力へ変換し、プレイ済みなら上位%を付けます。
func toCourseRecordOutputWithTopPercent(record *entity.PlayerCourseRecord, snapshot *scorepercentile.Snapshot) *CourseRecordOutput {
	item := toCourseRecordOutput(record)
	if item == nil || !item.IsPlayed || snapshot == nil {
		return item
	}
	if courseID := courseRecordCourseID(record); courseID != 0 {
		item.TopPercent = snapshot.LookupCourse(courseID, item.Score)
	}
	return item
}

func courseRecordCourseID(record *entity.PlayerCourseRecord) int {
	if record == nil {
		return 0
	}
	if record.CourseID != 0 {
		return record.CourseID
	}
	if record.Course != nil {
		return record.Course.ID
	}
	return 0
}
