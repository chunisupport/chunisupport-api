package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAdminChartRankingUsecase は呼び出し引数を記録するスタブです。
// context 以外の引数は具体値で記録し、mock.Anything は使いません。
type stubAdminChartRankingUsecase struct {
	standardResult  *usecase.AdminChartRankingResult
	worldsendResult *usecase.AdminChartRankingResult
	standardErr     error
	worldsendErr    error
	standardCalls   int
	worldsendCalls  int
	gotDisplayID    string
	gotDifficulty   string
}

func (s *stubAdminChartRankingUsecase) GetStandard(_ context.Context, displayID string, difficulty string) (*usecase.AdminChartRankingResult, error) {
	s.standardCalls++
	s.gotDisplayID = displayID
	s.gotDifficulty = difficulty
	return s.standardResult, s.standardErr
}

func (s *stubAdminChartRankingUsecase) GetWorldsend(_ context.Context, displayID string) (*usecase.AdminChartRankingResult, error) {
	s.worldsendCalls++
	s.gotDisplayID = displayID
	return s.worldsendResult, s.worldsendErr
}

func TestAdminChartRankingHandler_不正なパスパラメータはHTTP422を返す(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantCode   string
	}{
		{name: "display IDが不正", path: "/internal/admin/chart-rankings/songs/invalid/charts/MASTER", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "難易度が不正", path: "/internal/admin/chart-rankings/songs/0000000000000001/charts/WORLD'S%20END", wantStatus: http.StatusBadRequest, wantCode: "invalid_difficulty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			handler := NewAdminChartRankingHandler(&stubAdminChartRankingUsecase{})
			e.GET("/internal/admin/chart-rankings/songs/:id/charts/:difficulty", handler.GetStandard)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), `"code":"`+tt.wantCode+`"`)
		})
	}
}

func TestAdminChartRankingHandler_通常譜面のランキングを返す(t *testing.T) {
	// Given: 通常譜面のランキング取得結果。ID は 0 や 1 と紛れない固定値にする。
	const displayID = "abcdef0123456789"
	chartConst, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	isConstUnknown := false
	clearLamp := "CLEAR"
	comboLamp := "ALL JUSTICE"
	updatedAt := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	stub := &stubAdminChartRankingUsecase{standardResult: &usecase.AdminChartRankingResult{
		Song:  usecase.ChartRankingSong{ID: displayID, Title: "楽曲名", Artist: "アーティスト名"},
		Chart: usecase.ChartRankingChart{Difficulty: "MASTER", Const: &chartConst, IsConstUnknown: isConstUnknown},
		Ranking: []usecase.ChartRankingEntry{{
			Rank:             1,
			Username:         "rankinguser",
			PlayerName:       "PLAYER",
			Score:            1010000,
			Rating:           16.9,
			Overpower:        89.1,
			OverpowerPercent: 100,
			ClearLamp:        &clearLamp,
			ComboLamp:        &comboLamp,
			FullChain:        nil,
			UpdatedAt:        updatedAt,
		}},
		Total: 234,
	}}
	e := echo.New()
	handler := NewAdminChartRankingHandler(stub)
	req := httptest.NewRequest(http.MethodGet, "/internal/admin/chart-rankings/songs/"+displayID+"/charts/master", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// パス難易度は小文字で渡し、大文字に正規化されることを確かめる。
	c.SetPathValues(echo.PathValues{{Name: "id", Value: displayID}, {Name: "difficulty", Value: "master"}})

	// When: 通常譜面のランキングを取得する。
	err = handler.GetStandard(c)

	// Then: パスから取り出した表示用IDと大文字難易度がそのまま渡る。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, stub.standardCalls)
	assert.Zero(t, stub.worldsendCalls)
	assert.Equal(t, displayID, stub.gotDisplayID)
	assert.Equal(t, "MASTER", stub.gotDifficulty)
	assert.JSONEq(t, `{
		"song": {"id": "abcdef0123456789", "title": "楽曲名", "artist": "アーティスト名"},
		"chart": {"difficulty": "MASTER", "const": 14.5, "is_const_unknown": false, "is_worldsend": false},
		"ranking": [{
			"rank": 1,
			"username": "rankinguser",
			"player_name": "PLAYER",
			"score": 1010000,
			"rating": 16.9,
			"overpower": 89.1,
			"overpower_percent": 100.0,
			"clear_lamp": "CLEAR",
			"combo_lamp": "ALL JUSTICE",
			"full_chain": null,
			"updated_at": "2026-08-31T12:00:00Z"
		}],
		"total": 234
	}`, rec.Body.String())
	// 管理者向け項目だけを返し、内部ユーザー ID を公開しないことを確かめる。
	assert.Contains(t, rec.Body.String(), `"username":"rankinguser"`)
	assert.NotContains(t, rec.Body.String(), "user_id")
	assert.NotContains(t, rec.Body.String(), "is_self")
	assert.NotContains(t, rec.Body.String(), "my_rank")
}

func TestAdminChartRankingHandler_WORLD送信のランキングを返す(t *testing.T) {
	// Given: WORLD'S END 譜面のランキング取得結果。ID は通常譜面と紛れない固定値にする。
	const displayID = "0123456789abcdef"
	levelStar := 5
	attribute := "狂"
	clearLamp := "CLEAR"
	comboLamp := "FULL COMBO"
	updatedAt := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	stub := &stubAdminChartRankingUsecase{worldsendResult: &usecase.AdminChartRankingResult{
		Song:  usecase.ChartRankingSong{ID: displayID, Title: "WORLD'S END楽曲名", Artist: "アーティスト名"},
		Chart: usecase.ChartRankingChart{Difficulty: "WORLD'S END", LevelStar: &levelStar, Attribute: &attribute, IsWorldsend: true},
		Ranking: []usecase.ChartRankingEntry{{
			Rank:       1,
			Username:   "rankinguser",
			PlayerName: "PLAYER",
			Score:      1010000,
			ClearLamp:  &clearLamp,
			ComboLamp:  &comboLamp,
			FullChain:  nil,
			UpdatedAt:  updatedAt,
		}},
		Total: 1,
	}}
	e := echo.New()
	handler := NewAdminChartRankingHandler(stub)
	req := httptest.NewRequest(http.MethodGet, "/internal/admin/chart-rankings/worldsend-songs/"+displayID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: displayID}})

	// When: WORLD'S END 譜面のランキングを取得する。
	err := handler.GetWorldsend(c)

	// Then: パスから取り出した表示用IDがそのまま渡る。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, stub.worldsendCalls)
	assert.Zero(t, stub.standardCalls)
	assert.Equal(t, displayID, stub.gotDisplayID)
	assert.JSONEq(t, `{
		"song": {"id": "0123456789abcdef", "title": "WORLD'S END楽曲名", "artist": "アーティスト名"},
		"chart": {"difficulty": "WORLD'S END", "level_star": 5, "attribute": "狂", "is_worldsend": true},
		"ranking": [{
			"rank": 1,
			"username": "rankinguser",
			"player_name": "PLAYER",
			"score": 1010000,
			"clear_lamp": "CLEAR",
			"combo_lamp": "FULL COMBO",
			"full_chain": null,
			"updated_at": "2026-08-31T12:00:00Z"
		}],
		"total": 1
	}`, rec.Body.String())
	// WORLD'S END ではレート項目を返さないことを確かめる。
	assert.NotContains(t, rec.Body.String(), `"const"`)
	assert.NotContains(t, rec.Body.String(), `"rating"`)
	assert.NotContains(t, rec.Body.String(), `"overpower"`)
}

func TestAdminChartRankingHandler_usecaseエラーをAPIエラーへ変換する(t *testing.T) {
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
			const displayID = "abcdef0123456789"
			stub := &stubAdminChartRankingUsecase{standardErr: tt.usecaseErr, worldsendErr: tt.usecaseErr}
			e := echo.New()
			handler := NewAdminChartRankingHandler(stub)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.worldsend {
				c.SetPathValues(echo.PathValues{{Name: "id", Value: displayID}})
			} else {
				c.SetPathValues(echo.PathValues{{Name: "id", Value: displayID}, {Name: "difficulty", Value: "master"}})
			}

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
