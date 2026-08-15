package service

import (
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// RecordAchievementStatistics は譜面レコードの達成件数を表します。
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
	SPlus   int
	S       int
}

// PlayerRecordStatistics は譜面レコードのスコア合計と達成件数を表します。
type PlayerRecordStatistics struct {
	TotalHighScore int64
	Achievements   RecordAchievementStatistics
}

// PlayerRecordStatisticsSnapshot は全難易度および難易度別の集計結果です。
type PlayerRecordStatisticsSnapshot struct {
	Overall      PlayerRecordStatistics
	ByDifficulty map[string]PlayerRecordStatistics
}

// PlayerRecordStatisticsGroupNames は統計で返す通常難易度とWORLD'S ENDの固定キーを返します。
func PlayerRecordStatisticsGroupNames() []string {
	return append([]string(nil), playerRecordStatisticsGroupNames[:]...)
}

// CalculatePlayerRecordStatistics は通常譜面を全体・難易度別に、WORLD'S ENDをWEへ集計します。
// Overallは既存仕様を維持するため通常譜面だけを対象とします。
// マスタ不整合を統計へ混入させないため、関連情報の欠損や未知の難易度はエラーにします。
func CalculatePlayerRecordStatistics(records []*entity.PlayerRecord, worldsendRecords []*entity.PlayerWorldsendRecord) (PlayerRecordStatisticsSnapshot, error) {
	snapshot := PlayerRecordStatisticsSnapshot{ByDifficulty: make(map[string]PlayerRecordStatistics, len(playerRecordStatisticsGroupNames))}
	for _, groupName := range playerRecordStatisticsGroupNames {
		snapshot.ByDifficulty[groupName] = PlayerRecordStatistics{}
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

		addPlayerRecordStatistics(&snapshot.Overall, int(record.Score), record.ClearLamp.Name, record.ComboLamp.Name, record.FullChain.Name)
		addPlayerRecordStatistics(&group, int(record.Score), record.ClearLamp.Name, record.ComboLamp.Name, record.FullChain.Name)
		snapshot.ByDifficulty[difficulty] = group
	}

	worldsendGroup := snapshot.ByDifficulty[playerRecordStatisticsWorldsendName]
	for _, record := range worldsendRecords {
		if record == nil || record.ClearLamp == nil || record.ComboLamp == nil || record.FullChain == nil {
			return PlayerRecordStatisticsSnapshot{}, fmt.Errorf("player worldsend record has missing statistics relation")
		}
		addPlayerRecordStatistics(&worldsendGroup, int(record.Score), record.ClearLamp.Name, record.ComboLamp.Name, record.FullChain.Name)
	}
	snapshot.ByDifficulty[playerRecordStatisticsWorldsendName] = worldsendGroup

	return snapshot, nil
}

func addPlayerRecordStatistics(statistics *PlayerRecordStatistics, score int, clearLamp string, comboLamp string, fullChain string) {
	statistics.TotalHighScore += int64(score)
	achievements := &statistics.Achievements

	if comboLamp == "ALL JUSTICE" {
		achievements.AJ++
	}
	if comboLamp == "FULL COMBO" || comboLamp == "ALL JUSTICE" {
		achievements.FC++
	}
	if clearLamp != "FAILED" {
		achievements.CLR++
	}
	if fullChain == "FULL CHAIN GOLD" || fullChain == "FULL CHAIN PLATINUM" {
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
	if score >= playerRecordScoreSPlus {
		achievements.SPlus++
	}
	if score >= playerRecordScoreS {
		achievements.S++
	}
}
