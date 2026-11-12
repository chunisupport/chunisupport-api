package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubFriendChartRankingUsecase は呼び出し引数を記録するスタブです。
// context 以外の引数は具体値で記録し、mock.Anything は使いません。
type stubFriendChartRankingUsecase struct {
	standardResult  *usecase.FriendChartRankingResult
	worldsendResult *usecase.FriendChartRankingResult
	standardErr     error
	worldsendErr    error
	standardCalls   int
	worldsendCalls  int
	gotUserID       int
	gotDisplayID    string
	gotDifficulty   string
}

func (s *stubFriendChartRankingUsecase) GetStandard(_ context.Context, userID int, displayID string, difficulty string) (*usecase.FriendChartRankingResult, error) {
	s.standardCalls++
	s.gotUserID = userID
	s.gotDisplayID = displayID
	s.gotDifficulty = difficulty
	return s.standardResult, s.standardErr
}

func (s *stubFriendChartRankingUsecase) GetWorldsend(_ context.Context, userID int, displayID string) (*usecase.FriendChartRankingResult, error) {
	s.worldsendCalls++
	s.gotUserID = userID
	s.gotDisplayID = displayID
	return s.worldsendResult, s.worldsendErr
}

func TestFriendChartRankingHandler_不正なDisplayIDはHTTP422を返す(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	handler := NewFriendChartRankingHandler(&stubFriendChartRankingUsecase{})
	e.GET("/internal/friend-rankings/songs/:id/charts/:difficulty", func(c *echo.Context) error {
		c.Set("userEntity", &entity.User{ID: 1})
		return handler.GetStandard(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/internal/friend-rankings/songs/invalid/charts/MASTER", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.JSONEq(t, `{"error":{"status":422,"code":"validation_failed","message":"入力値の形式を確認してください。"}}`, rec.Body.String())
}

func TestFriendChartRankingHandler_通常譜面のランキングを返す(t *testing.T) {
	// Given: 認証ユーザーの ID は 0 や 1 と紛れない固定値にする。
	const userID = 42
	const displayID = "abcdef0123456789"
	chartConst, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	isConstUnknown := false
	clearLamp := "CLEAR"
	comboLamp := "ALL JUSTICE"
	updatedAt := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	myRank := 2
	stub := &stubFriendChartRankingUsecase{standardResult: &usecase.FriendChartRankingResult{
		Song:  usecase.ChartRankingSong{ID: displayID, Title: "楽曲名", Artist: "アーティスト名"},
		Chart: usecase.ChartRankingChart{Difficulty: "MASTER", Const: &chartConst, IsConstUnknown: isConstUnknown},
		Ranking: []usecase.FriendChartRankingEntry{
			{
				ChartRankingEntry: usecase.ChartRankingEntry{
					Rank:             1,
					Username:         "frienduser",
					PlayerName:       "FRIEND",
					Score:            1010000,
					Rating:           16.9,
					Overpower:        89.1,
					OverpowerPercent: 100,
					ClearLamp:        &clearLamp,
					ComboLamp:        &comboLamp,
					FullChain:        nil,
					UpdatedAt:        updatedAt,
				},
				UserID: 4242,
				IsSelf: false,
			},
			{
				ChartRankingEntry: usecase.ChartRankingEntry{
					Rank:             2,
					Username:         "myuser",
					PlayerName:       "PLAYER",
					Score:            1009500,
					Rating:           16.65,
					Overpower:        87.123,
					OverpowerPercent: 98.7654,
					ClearLamp:        &clearLamp,
					ComboLamp:        &comboLamp,
					FullChain:        nil,
					UpdatedAt:        updatedAt,
				},
				UserID: 42,
				IsSelf: true,
			},
		},
		MyRank: &myRank,
		Total:  2,
	}}
	e := echo.New()
	handler := NewFriendChartRankingHandler(stub)
	c, rec := newFriendChartRankingContext(e, displayID, "master")
	c.Set("userEntity", &entity.User{ID: userID})

	// When: 通常譜面のランキングを取得する。
	err = handler.GetStandard(c)

	// Then: 認証コンテキストのユーザー ID とパス値がそのまま渡る。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, stub.standardCalls)
	assert.Zero(t, stub.worldsendCalls)
	assert.Equal(t, userID, stub.gotUserID)
	assert.Equal(t, displayID, stub.gotDisplayID)
	assert.Equal(t, "MASTER", stub.gotDifficulty)
	assert.JSONEq(t, `{
		"song": {"id": "abcdef0123456789", "title": "楽曲名", "artist": "アーティスト名"},
		"chart": {"difficulty": "MASTER", "const": 14.5, "is_const_unknown": false, "is_worldsend": false},
		"ranking": [
			{
				"rank": 1,
				"username": "frienduser",
				"player_name": "FRIEND",
				"score": 1010000,
				"rating": 16.9,
				"overpower": 89.1,
				"overpower_percent": 100.0,
				"clear_lamp": "CLEAR",
				"combo_lamp": "ALL JUSTICE",
				"full_chain": null,
				"updated_at": "2026-07-09T12:00:00Z",
				"is_self": false
			},
			{
				"rank": 2,
				"username": "myuser",
				"player_name": "PLAYER",
				"score": 1009500,
				"rating": 16.65,
				"overpower": 87.123,
				"overpower_percent": 98.7654,
				"clear_lamp": "CLEAR",
				"combo_lamp": "ALL JUSTICE",
				"full_chain": null,
				"updated_at": "2026-07-09T12:00:00Z",
				"is_self": true
			}
		],
		"my_rank": 2,
		"total": 2
	}`, rec.Body.String())
	// 内部ユーザー ID を公開しないことを確かめる。
	assert.NotContains(t, rec.Body.String(), "user_id")
}

func TestFriendChartRankingHandler_WORLD送信のランキングを返す(t *testing.T) {
	// Given: WORLD'S END 譜面のランキング取得結果。ID は通常譜面と紛れない固定値にする。
	const userID = 42
	const displayID = "0123456789abcdef"
	levelStar := 5
	attribute := "狂"
	clearLamp := "CLEAR"
	comboLamp := "FULL COMBO"
	updatedAt := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	stub := &stubFriendChartRankingUsecase{worldsendResult: &usecase.FriendChartRankingResult{
		Song:  usecase.ChartRankingSong{ID: displayID, Title: "WORLD'S END楽曲名", Artist: "アーティスト名"},
		Chart: usecase.ChartRankingChart{Difficulty: "WORLD'S END", LevelStar: &levelStar, Attribute: &attribute, IsWorldsend: true},
		Ranking: []usecase.FriendChartRankingEntry{
			{
				ChartRankingEntry: usecase.ChartRankingEntry{
					Rank:       1,
					Username:   "frienduser",
					PlayerName: "FRIEND",
					Score:      1009500,
					ClearLamp:  &clearLamp,
					ComboLamp:  &comboLamp,
					FullChain:  nil,
					UpdatedAt:  updatedAt,
				},
				UserID: 4242,
				IsSelf: false,
			},
		},
		MyRank: nil,
		Total:  1,
	}}
	e := echo.New()
	handler := NewFriendChartRankingHandler(stub)
	c, rec := newFriendChartRankingContext(e, displayID, "")
	c.Set("userEntity", &entity.User{ID: userID})

	// When: WORLD'S END 譜面のランキングを取得する。
	err := handler.GetWorldsend(c)

	// Then: 認証コンテキストのユーザー ID とパス値がそのまま渡る。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, stub.worldsendCalls)
	assert.Zero(t, stub.standardCalls)
	assert.Equal(t, userID, stub.gotUserID)
	assert.Equal(t, displayID, stub.gotDisplayID)
	assert.JSONEq(t, `{
		"song": {"id": "0123456789abcdef", "title": "WORLD'S END楽曲名", "artist": "アーティスト名"},
		"chart": {"difficulty": "WORLD'S END", "level_star": 5, "attribute": "狂", "is_worldsend": true},
		"ranking": [{
			"rank": 1,
			"username": "frienduser",
			"player_name": "FRIEND",
			"score": 1009500,
			"clear_lamp": "CLEAR",
			"combo_lamp": "FULL COMBO",
			"full_chain": null,
			"updated_at": "2026-07-09T12:00:00Z",
			"is_self": false
		}],
		"my_rank": null,
		"total": 1
	}`, rec.Body.String())
	// WORLD'S END ではレート項目を返さず、内部ユーザー ID を公開しないことを確かめる。
	assert.NotContains(t, rec.Body.String(), `"const"`)
	assert.NotContains(t, rec.Body.String(), `"rating"`)
	assert.NotContains(t, rec.Body.String(), `"overpower"`)
	assert.NotContains(t, rec.Body.String(), "user_id")
}

func TestFriendChartRankingHandler_未認証は401を返す(t *testing.T) {
	tests := []struct {
		name string
		// Given: 呼び出すハンドラーメソッド。
		worldsend bool
	}{
		{name: "通常譜面", worldsend: false},
		{name: "WORLD'S END", worldsend: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 認証コンテキストにユーザーを設定しない。
			const displayID = "abcdef0123456789"
			stub := &stubFriendChartRankingUsecase{}
			e := echo.New()
			handler := NewFriendChartRankingHandler(stub)
			var c *echo.Context
			if tt.worldsend {
				c, _ = newFriendChartRankingContext(e, displayID, "")
			} else {
				c, _ = newFriendChartRankingContext(e, displayID, "master")
			}

			// When
			var err error
			if tt.worldsend {
				err = handler.GetWorldsend(c)
			} else {
				err = handler.GetStandard(c)
			}

			// Then: usecase を呼ばず 401 を返す。
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, apierror.CodeUnauthorized, apiErr.Code)
			assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
			assert.Zero(t, stub.standardCalls)
			assert.Zero(t, stub.worldsendCalls)
		})
	}
}

func TestFriendChartRankingHandler_usecaseエラーをAPIエラーへ変換する(t *testing.T) {
	tests := []struct {
		name string
		// Given: 呼び出すハンドラーメソッドと usecase の返すエラー。
		worldsend  bool
		usecaseErr error
		// Then: 期待する API エラーのコードとステータス。
		wantCode   string
		wantStatus int
	}{
		{
			name:       "通常譜面が存在しない場合は404",
			worldsend:  false,
			usecaseErr: usecase.ErrChartNotFound,
			wantCode:   apierror.CodeChartNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "WORLD'S ENDで内部エラーの場合は500",
			worldsend:  true,
			usecaseErr: usecase.ErrInternalError,
			wantCode:   apierror.CodeInternalError,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			const userID = 42
			const displayID = "abcdef0123456789"
			stub := &stubFriendChartRankingUsecase{standardErr: tt.usecaseErr, worldsendErr: tt.usecaseErr}
			e := echo.New()
			handler := NewFriendChartRankingHandler(stub)
			var c *echo.Context
			if tt.worldsend {
				c, _ = newFriendChartRankingContext(e, displayID, "")
			} else {
				c, _ = newFriendChartRankingContext(e, displayID, "master")
			}
			c.Set("userEntity", &entity.User{ID: userID})

			// When
			var err error
			if tt.worldsend {
				err = handler.GetWorldsend(c)
			} else {
				err = handler.GetStandard(c)
			}

			// Then: usecase のエラーが対応する API エラーへ変換される。
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
		})
	}
}

func newFriendChartRankingContext(e *echo.Echo, id string, difficulty string) (*echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if difficulty == "" {
		c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})
	} else {
		c.SetPathValues(echo.PathValues{{Name: "id", Value: id}, {Name: "difficulty", Value: difficulty}})
	}
	return c, rec
}
