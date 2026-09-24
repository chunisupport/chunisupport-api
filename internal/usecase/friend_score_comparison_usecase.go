package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

const (
	FriendScoreComparisonSelfWin   = "SELF_WIN"
	FriendScoreComparisonDraw      = "DRAW"
	FriendScoreComparisonFriendWin = "FRIEND_WIN"
)

// FriendScoreComparisonUser は比較レスポンスに出す公開識別子です。
type FriendScoreComparisonUser struct {
	Username   string
	PlayerName string
}

// FriendScoreComparisonSong は比較対象楽曲の公開概要です。
type FriendScoreComparisonSong struct {
	ID     string
	Title  string
	Artist string
}

// FriendScoreComparisonChart は比較対象譜面の公開概要です。
type FriendScoreComparisonChart struct {
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
	LevelStar      *int
	Attribute      *string
}

// FriendScoreComparisonRecord は未プレイを正規化した1人分の現在レコードです。
type FriendScoreComparisonRecord struct {
	IsPlayed  bool
	Score     uint32
	ClearLamp *string
	ComboLamp *string
	FullChain *string
	UpdatedAt *time.Time
}

// FriendScoreComparisonItem は1譜面の比較結果です。
type FriendScoreComparisonItem struct {
	Song            FriendScoreComparisonSong
	Chart           FriendScoreComparisonChart
	Self            FriendScoreComparisonRecord
	Friend          FriendScoreComparisonRecord
	ScoreDifference int
	Result          string
}

// FriendScoreComparisonSummary は指定難易度の勝敗とプレイ状態の集計です。
type FriendScoreComparisonSummary struct {
	TotalCharts      int
	SelfWins         int
	Draws            int
	FriendWins       int
	SelfPlayed       int
	FriendPlayed     int
	BothPlayed       int
	SelfOnlyPlayed   int
	FriendOnlyPlayed int
	BothUnplayed     int
}

// FriendScoreComparisonResult はフレンドスコア比較の取得結果です。
type FriendScoreComparisonResult struct {
	Difficulty string
	Self       FriendScoreComparisonUser
	Friend     FriendScoreComparisonUser
	Summary    FriendScoreComparisonSummary
	Items      []FriendScoreComparisonItem
}

// FriendScoreComparisonUsecase は承認済みフレンドとの譜面スコア比較を提供します。
type FriendScoreComparisonUsecase interface {
	Get(ctx context.Context, selfUserID int, friendUsername string, difficulty string) (*FriendScoreComparisonResult, error)
	GetWorldsend(ctx context.Context, selfUserID int, friendUsername string) (*FriendScoreComparisonResult, error)
}

type friendScoreComparisonUsecase struct {
	query repository.FriendScoreComparisonQueryService
}

func NewFriendScoreComparisonUsecase(query repository.FriendScoreComparisonQueryService) FriendScoreComparisonUsecase {
	return &friendScoreComparisonUsecase{query: query}
}

func (u *friendScoreComparisonUsecase) Get(ctx context.Context, selfUserID int, friendUsername string, difficulty string) (*FriendScoreComparisonResult, error) {
	if !service.IsExactStandardDifficulty(difficulty) {
		return nil, ErrInvalidDifficulty
	}

	pair, err := u.query.FindAcceptedFriendPair(ctx, selfUserID, friendUsername)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, ErrFriendNotFound
	}
	if pair.Self.PlayerID == nil || pair.Friend.PlayerID == nil {
		return nil, ErrFriendScoreComparisonUnavailable
	}

	rows, err := u.query.ListChartRecords(ctx, *pair.Self.PlayerID, *pair.Friend.PlayerID, difficulty)
	if err != nil {
		return nil, err
	}

	return buildFriendScoreComparisonResult(difficulty, pair, rows), nil
}

func (u *friendScoreComparisonUsecase) GetWorldsend(ctx context.Context, selfUserID int, friendUsername string) (*FriendScoreComparisonResult, error) {
	pair, err := u.query.FindAcceptedFriendPair(ctx, selfUserID, friendUsername)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, ErrFriendNotFound
	}
	if pair.Self.PlayerID == nil || pair.Friend.PlayerID == nil {
		return nil, ErrFriendScoreComparisonUnavailable
	}

	rows, err := u.query.ListWorldsendChartRecords(ctx, *pair.Self.PlayerID, *pair.Friend.PlayerID)
	if err != nil {
		return nil, err
	}
	return buildFriendScoreComparisonResult(info.StatsDifficultyWorldsend, pair, rows), nil
}

func buildFriendScoreComparisonResult(difficulty string, pair *repository.FriendScoreComparisonUsers, rows []*repository.FriendScoreComparisonChartRecord) *FriendScoreComparisonResult {
	result := &FriendScoreComparisonResult{
		Difficulty: difficulty,
		Self: FriendScoreComparisonUser{
			Username:   pair.Self.Username,
			PlayerName: pair.Self.PlayerName,
		},
		Friend: FriendScoreComparisonUser{
			Username:   pair.Friend.Username,
			PlayerName: pair.Friend.PlayerName,
		},
		Items: make([]FriendScoreComparisonItem, 0, len(rows)),
	}

	for _, row := range rows {
		selfRecord := normalizeComparisonPlay(row.Self)
		friendRecord := normalizeComparisonPlay(row.Friend)
		difference, comparisonResult := compareFriendScores(selfRecord.Score, friendRecord.Score)
		accumulateComparisonSummary(&result.Summary, selfRecord.IsPlayed, friendRecord.IsPlayed, comparisonResult)
		result.Items = append(result.Items, FriendScoreComparisonItem{
			Song: FriendScoreComparisonSong{
				ID:     row.SongDisplayID,
				Title:  row.SongTitle,
				Artist: row.SongArtist,
			},
			Chart: FriendScoreComparisonChart{
				Const:          row.ChartConst,
				IsConstUnknown: row.IsConstUnknown,
				LevelStar:      row.LevelStar,
				Attribute:      row.Attribute,
			},
			Self:            selfRecord,
			Friend:          friendRecord,
			ScoreDifference: difference,
			Result:          comparisonResult,
		})
	}
	return result
}

func normalizeComparisonPlay(play *repository.FriendScoreComparisonPlay) FriendScoreComparisonRecord {
	if play == nil {
		return FriendScoreComparisonRecord{}
	}
	updatedAt := play.UpdatedAt
	return FriendScoreComparisonRecord{
		IsPlayed:  true,
		Score:     play.Score,
		ClearLamp: rankingLampNamePtr(play.ClearLamp),
		ComboLamp: rankingLampNamePtr(play.ComboLamp),
		FullChain: rankingLampNamePtr(play.FullChain),
		UpdatedAt: &updatedAt,
	}
}

func compareFriendScores(selfScore uint32, friendScore uint32) (int, string) {
	difference := int(selfScore) - int(friendScore)
	switch {
	case difference > 0:
		return difference, FriendScoreComparisonSelfWin
	case difference < 0:
		return difference, FriendScoreComparisonFriendWin
	default:
		return 0, FriendScoreComparisonDraw
	}
}

func accumulateComparisonSummary(summary *FriendScoreComparisonSummary, selfPlayed bool, friendPlayed bool, result string) {
	summary.TotalCharts++
	switch {
	case selfPlayed && friendPlayed:
		summary.BothPlayed++
	case selfPlayed:
		summary.SelfOnlyPlayed++
	case friendPlayed:
		summary.FriendOnlyPlayed++
	default:
		summary.BothUnplayed++
	}
	if selfPlayed {
		summary.SelfPlayed++
	}
	if friendPlayed {
		summary.FriendPlayed++
	}
	switch result {
	case FriendScoreComparisonSelfWin:
		summary.SelfWins++
	case FriendScoreComparisonFriendWin:
		summary.FriendWins++
	default:
		summary.Draws++
	}
}
