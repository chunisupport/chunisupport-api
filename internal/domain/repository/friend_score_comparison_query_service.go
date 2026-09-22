package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// FriendScoreComparisonUser はスコア比較対象ユーザーの読み取りモデルです。
// PlayerID が nil の場合はプレイヤーデータ未連携です。
type FriendScoreComparisonUser struct {
	Username   string
	PlayerName string
	PlayerID   *int
}

// FriendScoreComparisonUsers は自分と承認済みフレンドの組です。
type FriendScoreComparisonUsers struct {
	Self   FriendScoreComparisonUser
	Friend FriendScoreComparisonUser
}

// FriendScoreComparisonPlay は1譜面の現在レコードです。
// ポインタの nil は未プレイを表し、スコア0の実レコードとは区別します。
type FriendScoreComparisonPlay struct {
	Score     uint32
	ClearLamp string
	ComboLamp string
	FullChain string
	UpdatedAt time.Time
}

// FriendScoreComparisonChartRecord は指定難易度の1譜面と比較用レコードです。
type FriendScoreComparisonChartRecord struct {
	SongDisplayID  string
	SongTitle      string
	SongArtist     string
	ChartConst     chartconstant.ChartConstant
	IsConstUnknown bool
	Self           *FriendScoreComparisonPlay
	Friend         *FriendScoreComparisonPlay
}

// FriendScoreComparisonQueryService はフレンドスコア比較の読み取りを扱います。
type FriendScoreComparisonQueryService interface {
	FindAcceptedFriendPair(
		ctx context.Context,
		selfUserID int,
		friendUsername string,
	) (*FriendScoreComparisonUsers, error)

	ListChartRecords(
		ctx context.Context,
		selfPlayerID int,
		friendPlayerID int,
		difficulty string,
	) ([]*FriendScoreComparisonChartRecord, error)
}
