package entity

import "time"

// Goal はユーザーが設定する目標を表します。
type Goal struct {
	ID                uint32
	UserID            int
	GroupID           *uint32
	Title             string
	AchievementTypeID int
	AchievementType   string
	AchievementParams []byte
	Attributes        []byte
	InvertValue       bool
	InvertPercentage  bool
	// GaugeStart は進捗ゲージを0%として描く値です。単位は目標種別の現在値と同じで、0は従来どおり0から描きます。
	GaugeStart float64
	// ApplyGaugeStartToPercentage は達成率表示もゲージと同じくスタート位置から計算するかです。
	ApplyGaugeStartToPercentage bool
	SortOrder                   uint16
	CreatedAt                   time.Time
}
