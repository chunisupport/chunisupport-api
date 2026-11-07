package usecase

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

const (
	rankingComboLampNoneID       = 1
	rankingComboLampFullComboID  = 2
	rankingComboLampAllJusticeID = 3
)

// ChartRankingSong は譜面単位ランキング対象楽曲の概要です。
type ChartRankingSong struct {
	ID     string
	Title  string
	Artist string
}

// ChartRankingChart は譜面単位ランキング対象譜面の概要です。
type ChartRankingChart struct {
	Difficulty     string
	Const          *chartconstant.ChartConstant
	IsConstUnknown bool
	LevelStar      *int
	Attribute      *string
	IsWorldsend    bool
}

// ChartRankingEntry は譜面単位ランキングの1件のうち、管理用・フレンド用で共通の項目です。
type ChartRankingEntry struct {
	Rank             int
	Username         string
	PlayerName       string
	Score            uint32
	Rating           float64
	Overpower        float64
	OverpowerPercent float64
	ClearLamp        *string
	ComboLamp        *string
	FullChain        *string
	UpdatedAt        time.Time
}

// newChartRankingSong はランキング対象楽曲の概要を組み立てます。
func newChartRankingSong(chart *repository.ChartRankingChart) ChartRankingSong {
	return ChartRankingSong{
		ID:     chart.SongDisplayID,
		Title:  chart.SongTitle,
		Artist: chart.SongArtist,
	}
}

// newChartRankingChart はランキング対象譜面の概要を組み立てます。
func newChartRankingChart(chart *repository.ChartRankingChart) ChartRankingChart {
	return ChartRankingChart{
		Difficulty:     chart.Difficulty,
		Const:          chart.Const,
		IsConstUnknown: chart.IsConstUnknown,
		LevelStar:      chart.LevelStar,
		Attribute:      chart.Attribute,
		IsWorldsend:    chart.IsWorldsend,
	}
}

// scoreRanker はスコア降順に並んだレコードへ順位を振ります。
// 同点は同順位とし、次の順位は同点の人数分だけ飛ばします（例: 1, 1, 3）。
type scoreRanker struct {
	count     int
	rank      int
	prevScore uint32
}

// Next は次のレコードのスコアを受け取り、その順位を返します。スコア降順に呼び出してください。
func (r *scoreRanker) Next(score uint32) int {
	r.count++
	// 先頭はスコアにかかわらず1位とするため、件数で判定する。
	if r.count == 1 || score != r.prevScore {
		r.rank = r.count
		r.prevScore = score
	}
	return r.rank
}

// newChartRankingEntry は順位付け済みのレコードから、管理用・フレンド用で共通の項目を組み立てます。
// WORLD'S END と譜面定数が未設定の譜面は単曲レーティングを定義できないため、レーティング・OP・OP割合は0のままにします。
func newChartRankingEntry(chart ChartRankingChart, rank int, record *repository.ChartRankingRecord) ChartRankingEntry {
	entry := ChartRankingEntry{
		Rank:       rank,
		Username:   record.Username,
		PlayerName: record.PlayerName,
		Score:      record.Score,
		ClearLamp:  rankingLampNamePtr(record.ClearLamp),
		ComboLamp:  rankingLampNamePtr(record.ComboLamp),
		FullChain:  rankingLampNamePtr(record.FullChain),
		UpdatedAt:  record.UpdatedAt,
	}
	if !chart.IsWorldsend && chart.Const != nil {
		chartConst := chart.Const.Float64()
		comboLamp := comboLampID(record.ComboLamp)
		entry.Rating = service.CalcSingleRating(record.Score, chartConst)
		entry.Overpower = service.CalcSingleOverpower(record.Score, chartConst, comboLamp)
		entry.OverpowerPercent = service.CalcSingleOverpowerPercent(record.Score, chartConst, comboLamp)
	}
	return entry
}

func rankingLampNamePtr(name string) *string {
	if name == "" || name == "NONE" || name == "none" {
		return nil
	}
	return &name
}

func comboLampID(name string) int {
	switch name {
	case "FULL COMBO":
		return rankingComboLampFullComboID
	case "ALL JUSTICE":
		return rankingComboLampAllJusticeID
	default:
		return rankingComboLampNoneID
	}
}
