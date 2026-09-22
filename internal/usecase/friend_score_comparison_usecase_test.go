package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFriendScoreComparisonUsecase_Get_勝敗と未プレイを正規化する(t *testing.T) {
	updatedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	friendUpdatedAt := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	selfPlayerID := 101
	friendPlayerID := 102
	tests := []struct {
		name             string
		self             *repository.FriendScoreComparisonPlay
		friend           *repository.FriendScoreComparisonPlay
		wantResult       string
		wantDifference   int
		wantSelfPlayed   bool
		wantFriendPlayed bool
	}{
		{
			name:             "自分のスコアが高い場合はSELF_WIN",
			self:             &repository.FriendScoreComparisonPlay{Score: 1009000, ClearLamp: "CLEAR", UpdatedAt: updatedAt},
			friend:           &repository.FriendScoreComparisonPlay{Score: 1007500, ClearLamp: "CLEAR", UpdatedAt: friendUpdatedAt},
			wantResult:       FriendScoreComparisonSelfWin,
			wantDifference:   1500,
			wantSelfPlayed:   true,
			wantFriendPlayed: true,
		},
		{
			name:             "フレンドのスコアが高い場合はFRIEND_WIN",
			self:             &repository.FriendScoreComparisonPlay{Score: 1000000, ClearLamp: "CLEAR", UpdatedAt: updatedAt},
			friend:           &repository.FriendScoreComparisonPlay{Score: 1008000, ClearLamp: "CLEAR", UpdatedAt: friendUpdatedAt},
			wantResult:       FriendScoreComparisonFriendWin,
			wantDifference:   -8000,
			wantSelfPlayed:   true,
			wantFriendPlayed: true,
		},
		{
			name:             "同点はDRAW",
			self:             &repository.FriendScoreComparisonPlay{Score: 1007500, ClearLamp: "CLEAR", UpdatedAt: updatedAt},
			friend:           &repository.FriendScoreComparisonPlay{Score: 1007500, ClearLamp: "HARD", UpdatedAt: friendUpdatedAt},
			wantResult:       FriendScoreComparisonDraw,
			wantDifference:   0,
			wantSelfPlayed:   true,
			wantFriendPlayed: true,
		},
		{
			name:             "ランプ差があっても同スコアならDRAW",
			self:             &repository.FriendScoreComparisonPlay{Score: 1009000, ClearLamp: "CLEAR", ComboLamp: "FULL COMBO", UpdatedAt: updatedAt},
			friend:           &repository.FriendScoreComparisonPlay{Score: 1009000, ClearLamp: "FAILED", ComboLamp: "NONE", UpdatedAt: friendUpdatedAt},
			wantResult:       FriendScoreComparisonDraw,
			wantDifference:   0,
			wantSelfPlayed:   true,
			wantFriendPlayed: true,
		},
		{
			name:             "自分だけプレイ済みでスコアが0より大きい場合はSELF_WIN",
			self:             &repository.FriendScoreComparisonPlay{Score: 900000, ClearLamp: "CLEAR", UpdatedAt: updatedAt},
			wantResult:       FriendScoreComparisonSelfWin,
			wantDifference:   900000,
			wantSelfPlayed:   true,
			wantFriendPlayed: false,
		},
		{
			name:             "フレンドだけプレイ済みでスコアが0より大きい場合はFRIEND_WIN",
			friend:           &repository.FriendScoreComparisonPlay{Score: 900000, ClearLamp: "CLEAR", UpdatedAt: friendUpdatedAt},
			wantResult:       FriendScoreComparisonFriendWin,
			wantDifference:   -900000,
			wantSelfPlayed:   false,
			wantFriendPlayed: true,
		},
		{
			name:             "片方だけプレイ済みでもスコア0ならDRAW",
			self:             &repository.FriendScoreComparisonPlay{Score: 0, ClearLamp: "NONE", ComboLamp: "NONE", FullChain: "NONE", UpdatedAt: updatedAt},
			wantResult:       FriendScoreComparisonDraw,
			wantDifference:   0,
			wantSelfPlayed:   true,
			wantFriendPlayed: false,
		},
		{
			name:             "両者未プレイの場合はDRAW",
			wantResult:       FriendScoreComparisonDraw,
			wantDifference:   0,
			wantSelfPlayed:   false,
			wantFriendPlayed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			query := &friendScoreComparisonQueryMock{
				pair: acceptedComparisonPair(selfPlayerID, friendPlayerID),
				records: []*repository.FriendScoreComparisonChartRecord{{
					SongDisplayID: "0000000000000001",
					SongTitle:     "楽曲名",
					SongArtist:    "アーティスト名",
					ChartConst:    mustComparisonChartConst(t, 14.5),
					Self:          tt.self,
					Friend:        tt.friend,
				}},
			}
			u := NewFriendScoreComparisonUsecase(query)

			// When
			result, err := u.Get(context.Background(), 1, "frienduser", "MASTER")

			// Then
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			item := result.Items[0]
			assert.Equal(t, tt.wantResult, item.Result)
			assert.Equal(t, tt.wantDifference, item.ScoreDifference)
			assert.Equal(t, tt.wantSelfPlayed, item.Self.IsPlayed)
			assert.Equal(t, tt.wantFriendPlayed, item.Friend.IsPlayed)
			assert.Equal(t, "0000000000000001", item.Song.ID)
			if !tt.wantSelfPlayed {
				assert.Equal(t, uint32(0), item.Self.Score)
				assert.Nil(t, item.Self.ClearLamp)
				assert.Nil(t, item.Self.UpdatedAt)
			}
			if !tt.wantFriendPlayed {
				assert.Equal(t, uint32(0), item.Friend.Score)
				assert.Nil(t, item.Friend.ClearLamp)
				assert.Nil(t, item.Friend.UpdatedAt)
			}
		})
	}
}

func TestFriendScoreComparisonUsecase_Get_summary不変条件が一致する(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	selfPlayerID := 101
	friendPlayerID := 102
	query := &friendScoreComparisonQueryMock{
		pair: acceptedComparisonPair(selfPlayerID, friendPlayerID),
		records: []*repository.FriendScoreComparisonChartRecord{
			{Self: playedComparison(1009000, updatedAt), Friend: playedComparison(1007500, updatedAt)},
			{},
			{Friend: playedComparison(1008000, updatedAt)},
			{Self: playedComparison(0, updatedAt)},
			{Self: playedComparison(1001000, updatedAt)},
			{Self: playedComparison(1005000, updatedAt), Friend: &repository.FriendScoreComparisonPlay{Score: 1005000, ClearLamp: "CLEAR", ComboLamp: "FULL COMBO", UpdatedAt: updatedAt}},
		},
	}
	u := NewFriendScoreComparisonUsecase(query)

	// When
	result, err := u.Get(context.Background(), 1, "frienduser", "ULTIMA")

	// Then
	require.NoError(t, err)
	summary := result.Summary
	assert.Equal(t, 6, summary.TotalCharts)
	assert.Equal(t, 2, summary.SelfWins)
	assert.Equal(t, 3, summary.Draws)
	assert.Equal(t, 1, summary.FriendWins)
	assert.Equal(t, 4, summary.SelfPlayed)
	assert.Equal(t, 3, summary.FriendPlayed)
	assert.Equal(t, 2, summary.BothPlayed)
	assert.Equal(t, 2, summary.SelfOnlyPlayed)
	assert.Equal(t, 1, summary.FriendOnlyPlayed)
	assert.Equal(t, 1, summary.BothUnplayed)
	assert.Equal(t, summary.TotalCharts, summary.SelfWins+summary.Draws+summary.FriendWins)
	assert.Equal(t, summary.TotalCharts, summary.BothPlayed+summary.SelfOnlyPlayed+summary.FriendOnlyPlayed+summary.BothUnplayed)
	assert.Equal(t, summary.SelfPlayed, summary.BothPlayed+summary.SelfOnlyPlayed)
	assert.Equal(t, summary.FriendPlayed, summary.BothPlayed+summary.FriendOnlyPlayed)
	require.NotNil(t, result.Items[5].Friend.ComboLamp)
	assert.Equal(t, "FULL COMBO", *result.Items[5].Friend.ComboLamp)
	assert.Nil(t, result.Items[3].Self.ComboLamp)
	assert.Nil(t, result.Items[3].Self.FullChain)
	require.NotNil(t, result.Items[3].Self.ClearLamp)
	assert.Equal(t, "CLEAR", *result.Items[3].Self.ClearLamp)
}

func TestFriendScoreComparisonUsecase_Get_許可されていない難易度はRepositoryへ渡さない(t *testing.T) {
	tests := []struct {
		name       string
		difficulty string
	}{
		{name: "小文字", difficulty: "master"},
		{name: "混在", difficulty: "Master"},
		{name: "短縮形", difficulty: "MAS"},
		{name: "空文字", difficulty: ""},
		{name: "WORLD'S END", difficulty: "WORLD'S END"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			query := &friendScoreComparisonQueryMock{}
			u := NewFriendScoreComparisonUsecase(query)

			// When
			result, err := u.Get(context.Background(), 1, "frienduser", tt.difficulty)

			// Then
			assert.Nil(t, result)
			assert.ErrorIs(t, err, ErrInvalidDifficulty)
			assert.Zero(t, query.findCalls)
			assert.Zero(t, query.listCalls)
		})
	}
}

func TestFriendScoreComparisonUsecase_Get_フレンドでない場合は比較行を取得しない(t *testing.T) {
	// Given
	query := &friendScoreComparisonQueryMock{}
	u := NewFriendScoreComparisonUsecase(query)

	// When
	result, err := u.Get(context.Background(), 1, "frienduser", "MASTER")

	// Then
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFriendNotFound)
	assert.Equal(t, 1, query.findCalls)
	assert.Zero(t, query.listCalls)
}

func TestFriendScoreComparisonUsecase_Get_プレイヤー未連携は比較行を取得しない(t *testing.T) {
	selfPlayerID := 101
	friendPlayerID := 102
	tests := []struct {
		name string
		pair *repository.FriendScoreComparisonUsers
	}{
		{name: "自分が未連携", pair: acceptedComparisonPair(0, friendPlayerID)},
		{name: "フレンドが未連携", pair: acceptedComparisonPair(selfPlayerID, 0)},
	}
	for i := range tests {
		if tests[i].name == "自分が未連携" {
			tests[i].pair.Self.PlayerID = nil
		} else {
			tests[i].pair.Friend.PlayerID = nil
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			query := &friendScoreComparisonQueryMock{pair: tt.pair}
			u := NewFriendScoreComparisonUsecase(query)

			// When
			result, err := u.Get(context.Background(), 1, "frienduser", "MASTER")

			// Then
			assert.Nil(t, result)
			assert.ErrorIs(t, err, ErrFriendScoreComparisonUnavailable)
			assert.Equal(t, 1, query.findCalls)
			assert.Zero(t, query.listCalls)
		})
	}
}

func TestFriendScoreComparisonUsecase_Get_比較対象譜面0件は空レスポンス(t *testing.T) {
	// Given
	query := &friendScoreComparisonQueryMock{pair: acceptedComparisonPair(101, 102)}
	u := NewFriendScoreComparisonUsecase(query)

	// When
	result, err := u.Get(context.Background(), 1, "frienduser", "BASIC")

	// Then
	require.NoError(t, err)
	assert.Equal(t, "BASIC", result.Difficulty)
	assert.Empty(t, result.Items)
	assert.NotNil(t, result.Items)
	assert.Equal(t, FriendScoreComparisonSummary{}, result.Summary)
	assert.Equal(t, "myuser", result.Self.Username)
	assert.Equal(t, "frienduser", result.Friend.Username)
	assert.Equal(t, 1, query.listCalls)
	assert.Equal(t, "BASIC", query.listDifficulty)
}

type friendScoreComparisonQueryMock struct {
	pair           *repository.FriendScoreComparisonUsers
	pairErr        error
	records        []*repository.FriendScoreComparisonChartRecord
	recordsErr     error
	findCalls      int
	listCalls      int
	listDifficulty string
}

func (m *friendScoreComparisonQueryMock) FindAcceptedFriendPair(ctx context.Context, selfUserID int, friendUsername string) (*repository.FriendScoreComparisonUsers, error) {
	m.findCalls++
	return m.pair, m.pairErr
}

func (m *friendScoreComparisonQueryMock) ListChartRecords(ctx context.Context, selfPlayerID int, friendPlayerID int, difficulty string) ([]*repository.FriendScoreComparisonChartRecord, error) {
	m.listCalls++
	m.listDifficulty = difficulty
	return m.records, m.recordsErr
}

func acceptedComparisonPair(selfPlayerID int, friendPlayerID int) *repository.FriendScoreComparisonUsers {
	selfID := selfPlayerID
	friendID := friendPlayerID
	pair := &repository.FriendScoreComparisonUsers{
		Self:   repository.FriendScoreComparisonUser{Username: "myuser", PlayerName: "MY PLAYER", PlayerID: &selfID},
		Friend: repository.FriendScoreComparisonUser{Username: "frienduser", PlayerName: "FRIEND", PlayerID: &friendID},
	}
	return pair
}

func playedComparison(score uint32, updatedAt time.Time) *repository.FriendScoreComparisonPlay {
	return &repository.FriendScoreComparisonPlay{
		Score:     score,
		ClearLamp: "CLEAR",
		ComboLamp: "NONE",
		FullChain: "none",
		UpdatedAt: updatedAt,
	}
}

func mustComparisonChartConst(t *testing.T, value float64) chartconstant.ChartConstant {
	t.Helper()
	chartConst, err := chartconstant.NewChartConstant(value)
	require.NoError(t, err)
	return chartConst
}
