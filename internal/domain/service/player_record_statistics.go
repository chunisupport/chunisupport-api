package service

import (
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// RecordAchievementStatistics は通常譜面の達成件数を表します。
type RecordAchievementStatistics struct {
	AJ      int
	FC      int
	CLR     int
	FCH     int
	MAX     int
	SSSPlus int
	SSS     int
	SSPlus  int
	SS      int
}

// PlayerRecordStatistics は通常譜面のスコア合計と達成件数を表します。
type PlayerRecordStatistics struct {
	TotalHighScore int64
	Achievements   RecordAchievementStatistics
}

// PlayerRecordStatisticsSnapshot は全難易度および難易度別の集計結果です。
type PlayerRecordStatisticsSnapshot struct {
	Overall      PlayerRecordStatistics
	ByDifficulty map[string]PlayerRecordStatistics
}

// PlayerRecordDifficultyNames は統計で返す固定難易度名を返します。
func PlayerRecordDifficultyNames() []string {
	return append([]string(nil), playerRecordDifficultyNames[:]...)
}

// CalculatePlayerRecordStatistics は通常譜面レコードを全体・難易度別に集計します。
// マスタ不整合を統計へ混入させないため、関連情報の欠損や未知の難易度はエラーにします。
func CalculatePlayerRecordStatistics(records []*entity.PlayerRecord) (PlayerRecordStatisticsSnapshot, error) {
	snapshot := PlayerRecordStatisticsSnapshot{ByDifficulty: make(map[string]PlayerRecordStatistics, len(playerRecordDifficultyNames))}
	for _, difficulty := range playerRecordDifficultyNames {
		snapshot.ByDifficulty[difficulty] = PlayerRecordStatistics{}
	}

	for _, record := range records {
		if record == nil || record.ChartDifficulty == nil || record.ClearLamp == nil || record.ComboLamp == nil || record.FullChain == nil {
			return PlayerRecordStatisticsSnapshot{}, fmt.Errorf("player record has missing statistics relation")
		}
		difficulty := record.ChartDifficulty.Name
		group, ok := snapshot.ByDifficulty[difficulty]
		if !ok {
			return PlayerRecordStatisticsSnapshot{}, fmt.Errorf("unknown chart difficulty: %s", difficulty)
		}

		addPlayerRecordStatistics(&snapshot.Overall, record)
		addPlayerRecordStatistics(&group, record)
		snapshot.ByDifficulty[difficulty] = group
	}

	return snapshot, nil
}

func addPlayerRecordStatistics(statistics *PlayerRecordStatistics, record *entity.PlayerRecord) {
	score := int(record.Score)
	statistics.TotalHighScore += int64(score)
	achievements := &statistics.Achievements

	if record.ComboLamp.Name == "ALL JUSTICE" {
		achievements.AJ++
	}
	if record.ComboLamp.Name == "FULL COMBO" || record.ComboLamp.Name == "ALL JUSTICE" {
		achievements.FC++
	}
	if record.ClearLamp.Name != "FAILED" {
		achievements.CLR++
	}
	if record.FullChain.Name == "FULL CHAIN GOLD" || record.FullChain.Name == "FULL CHAIN PLATINUM" {
		achievements.FCH++
	}
	if score == playerRecordScoreMax {
		achievements.MAX++
	}
	if score >= playerRecordScoreSSSPlus {
		achievements.SSSPlus++
	}
	if score >= playerRecordScoreSSS {
		achievements.SSS++
	}
	if score >= playerRecordScoreSSPlus {
		achievements.SSPlus++
	}
	if score >= playerRecordScoreSS {
		achievements.SS++
	}
}
