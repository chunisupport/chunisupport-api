package repository

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// ChartRankingChart は譜面単位ランキング対象の譜面情報のうち、管理用・フレンド用で共通の項目です（通常 / WORLD'S END）。
type ChartRankingChart struct {
	SongDisplayID  string
	SongTitle      string
	SongArtist     string
	Difficulty     string
	Const          *chartconstant.ChartConstant
	IsConstUnknown bool
	LevelStar      *int
	Attribute      *string
	IsWorldsend    bool
}

// ChartRankingRecord は譜面単位ランキングの1件のうち、管理用・フレンド用で共通の項目です。
type ChartRankingRecord struct {
	Username   string
	PlayerName string
	Score      uint32
	ClearLamp  string
	ComboLamp  string
	FullChain  string
	UpdatedAt  time.Time
}
