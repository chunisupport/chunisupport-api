package api_internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bestSlotRankingUsecaseStub struct{}

func (bestSlotRankingUsecaseStub) Get(_ context.Context, ratingBand string, _ *repository.BestSlotRankingCursor, limit int) (*usecase.BestSlotRankingResult, error) {
	return &usecase.BestSlotRankingResult{
		RatingBand: ratingBand,
		Ranking: []usecase.BestSlotRankingEntry{{
			Rank: 1, SongID: "0000000000000001", Title: "楽曲名", Difficulty: "MASTER",
			ChartConst: chartconstant.ChartConstant(14.8), BestPlayerCount: 10, BestPlayerPercentage: 25,
		}},
	}, nil
}

type chartStatsUsecaseBestSlotStub struct{}

type bestSlotMasterProviderStub struct{}

func (bestSlotMasterProviderStub) RatingBands() []*ratingband.RatingBand { return nil }

func (chartStatsUsecaseBestSlotStub) GetSongStatsByDisplayID(context.Context, string, *int) (*entity.SongChartStats, error) {
	return nil, nil
}
func (chartStatsUsecaseBestSlotStub) GetChartStatsByDisplayIDAndDifficulty(context.Context, string, string, *int) (*entity.SingleChartStats, error) {
	return nil, nil
}
func (chartStatsUsecaseBestSlotStub) GetChartBestSlotStatsByDisplayIDAndDifficulty(context.Context, string, string, *int) (*entity.SingleChartBestSlotStats, error) {
	return nil, nil
}

func TestBestSlotStatsHandler_GetRanking_不要な楽曲情報と内部IDを返さない(t *testing.T) {
	// Given
	e := echo.New()
	h := NewBestSlotStatsHandler(bestSlotRankingUsecaseStub{}, chartStatsUsecaseBestSlotStub{}, bestSlotMasterProviderStub{})
	e.GET("/internal/best-slot-rankings", h.GetRanking)
	req := httptest.NewRequest(http.MethodGet, "/internal/best-slot-rankings?rating_band=17.0", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"rating_band":"17.0"`)
	assert.Contains(t, rec.Body.String(), `"title":"楽曲名"`)
	assert.NotContains(t, rec.Body.String(), "rating_band_id")
	assert.NotContains(t, rec.Body.String(), "artist")
	assert.NotContains(t, rec.Body.String(), "jacket")
}

func TestBestSlotRankingCursor_公開値だけで往復する(t *testing.T) {
	// Given
	cursor := &repository.BestSlotRankingCursor{
		RatingBand: "17.0", BestPlayerPercentage: 25, BestPlayerCount: 10,
		SongDisplayID: "0000000000000001", Difficulty: "MASTER",
	}

	// When
	encoded, err := encodeBestSlotRankingCursor(cursor)
	require.NoError(t, err)
	decoded, apiErr := decodeBestSlotRankingCursor(encoded)

	// Then
	require.Nil(t, apiErr)
	assert.Equal(t, cursor, decoded)
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "ChartID")
	assert.NotContains(t, string(payload), "RatingBandID")
}

func TestDecodeBestSlotRankingCursor_破損した値はエラー(t *testing.T) {
	// When
	decoded, apiErr := decodeBestSlotRankingCursor("not-base64!")

	// Then
	assert.Nil(t, decoded)
	require.NotNil(t, apiErr)
}

func TestDecodeBestSlotRankingCursor_難易度を大文字へ正規化する(t *testing.T) {
	// Given
	cursor := &repository.BestSlotRankingCursor{
		RatingBand: "17.0", BestPlayerPercentage: 25, BestPlayerCount: 10,
		SongDisplayID: "0000000000000001", Difficulty: "master",
	}
	encoded, err := encodeBestSlotRankingCursor(cursor)
	require.NoError(t, err)

	// When
	decoded, apiErr := decodeBestSlotRankingCursor(encoded)

	// Then
	require.Nil(t, apiErr)
	require.NotNil(t, decoded)
	assert.Equal(t, "MASTER", decoded.Difficulty)
}

// recordingBestSlotRankingUsecaseStub はハンドラーから渡された引数を記録します。
// limit とカーソルが検証後に usecase へ正しく渡ることを具体値で照合するために使います。
type recordingBestSlotRankingUsecaseStub struct {
	called     bool
	ratingBand string
	cursor     *repository.BestSlotRankingCursor
	limit      int
}

func (s *recordingBestSlotRankingUsecaseStub) Get(_ context.Context, ratingBand string, cursor *repository.BestSlotRankingCursor, limit int) (*usecase.BestSlotRankingResult, error) {
	s.called = true
	s.ratingBand = ratingBand
	s.cursor = cursor
	s.limit = limit
	return &usecase.BestSlotRankingResult{RatingBand: ratingBand}, nil
}

// newBestSlotRankingRecordingServer は引数記録スタブを使ったランキング取得のテストサーバーを組み立てます。
func newBestSlotRankingRecordingServer(stub usecase.BestSlotRankingUsecase) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	h := NewBestSlotStatsHandler(stub, chartStatsUsecaseBestSlotStub{}, bestSlotMasterProviderStub{})
	e.GET("/internal/best-slot-rankings", h.GetRanking)
	return e
}

func TestBestSlotStatsHandler_GetRanking_limitの検証(t *testing.T) {
	tests := []struct {
		name string
		// Given: リクエストの limit クエリ（空文字はクエリ自体を省略）
		limitParam string
		// Then: 期待する応答と usecase へ渡る値（拒否時は wantLimit を見ない）
		wantStatus int
		wantLimit  int
	}{
		{
			name:       "未指定は既定値50が渡る",
			limitParam: "",
			wantStatus: http.StatusOK,
			wantLimit:  50,
		},
		{
			name:       "下限ちょうど1はその値が渡る",
			limitParam: "1",
			wantStatus: http.StatusOK,
			wantLimit:  1,
		},
		{
			name:       "上限ちょうど100はその値が渡る",
			limitParam: "100",
			wantStatus: http.StatusOK,
			wantLimit:  100,
		},
		{
			name:       "下限-1の0は拒否",
			limitParam: "0",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "上限+1の101は拒否",
			limitParam: "101",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "数値でない値は拒否",
			limitParam: "abc",
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &recordingBestSlotRankingUsecaseStub{}
			e := newBestSlotRankingRecordingServer(stub)
			target := "/internal/best-slot-rankings?rating_band=17.0"
			if tt.limitParam != "" {
				target += "&limit=" + url.QueryEscape(tt.limitParam)
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus != http.StatusOK {
				assert.False(t, stub.called)
				assert.Contains(t, rec.Body.String(), `"code":"validation_failed"`)
				return
			}
			require.True(t, stub.called)
			assert.Equal(t, tt.wantLimit, stub.limit)
		})
	}
}

// newValidBestSlotRankingCursor はカーソル検証を通過する固定値を返します。
// 境界テストではこの値から検証したい1項目だけを変えます。
func newValidBestSlotRankingCursor() repository.BestSlotRankingCursor {
	return repository.BestSlotRankingCursor{
		RatingBand:           "17.0",
		BestPlayerPercentage: 25,
		BestPlayerCount:      10,
		SongDisplayID:        "0000000000000001",
		Difficulty:           "MASTER",
	}
}

// encodeBestSlotRankingRawCursorParam はカーソル検証の拒否テスト用に、JSONを直接組み立ててbase64化します。
func encodeBestSlotRankingRawCursorParam(t *testing.T, mutate func(map[string]any)) string {
	t.Helper()
	payload := map[string]any{
		"RatingBand":           "17.0",
		"BestPlayerPercentage": 25.0,
		"BestPlayerCount":      10,
		"SongDisplayID":        "0000000000000001",
		"Difficulty":           "MASTER",
	}
	mutate(payload)
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func TestBestSlotStatsHandler_GetRanking_カーソル境界値は受理してそのまま渡る(t *testing.T) {
	tests := []struct {
		name string
		// Given: 有効なカーソルから変える1項目
		mutate func(*repository.BestSlotRankingCursor)
		// Then: usecaseへ渡ることを期待するカーソルの具体値
		want repository.BestSlotRankingCursor
	}{
		{
			name:   "割合の下限0は受理",
			mutate: func(c *repository.BestSlotRankingCursor) { c.BestPlayerPercentage = 0 },
			want:   repository.BestSlotRankingCursor{RatingBand: "17.0", BestPlayerPercentage: 0, BestPlayerCount: 10, SongDisplayID: "0000000000000001", Difficulty: "MASTER"},
		},
		{
			name:   "割合の上限100は受理",
			mutate: func(c *repository.BestSlotRankingCursor) { c.BestPlayerPercentage = 100 },
			want:   repository.BestSlotRankingCursor{RatingBand: "17.0", BestPlayerPercentage: 100, BestPlayerCount: 10, SongDisplayID: "0000000000000001", Difficulty: "MASTER"},
		},
		{
			name:   "件数の下限0は受理",
			mutate: func(c *repository.BestSlotRankingCursor) { c.BestPlayerCount = 0 },
			want:   repository.BestSlotRankingCursor{RatingBand: "17.0", BestPlayerPercentage: 25, BestPlayerCount: 0, SongDisplayID: "0000000000000001", Difficulty: "MASTER"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 有効なカーソルを本番の符号化関数で組み立てる
			cursor := newValidBestSlotRankingCursor()
			tt.mutate(&cursor)
			encoded, err := encodeBestSlotRankingCursor(&cursor)
			require.NoError(t, err)
			stub := &recordingBestSlotRankingUsecaseStub{}
			e := newBestSlotRankingRecordingServer(stub)
			req := httptest.NewRequest(http.MethodGet, "/internal/best-slot-rankings?rating_band=17.0&cursor="+url.QueryEscape(encoded), nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, http.StatusOK, rec.Code)
			require.True(t, stub.called)
			require.NotNil(t, stub.cursor)
			assert.Equal(t, tt.want, *stub.cursor)
		})
	}
}

func TestBestSlotStatsHandler_GetRanking_不正なカーソルは拒否する(t *testing.T) {
	tests := []struct {
		name string
		// Given: base64パラメーターの組み立て（有効なJSONから1項目だけ変えるか、JSON自体を壊す）
		cursorParam func(t *testing.T) string
	}{
		{
			name: "割合が下限のすぐ外は拒否",
			cursorParam: func(t *testing.T) string {
				return encodeBestSlotRankingRawCursorParam(t, func(p map[string]any) { p["BestPlayerPercentage"] = -0.1 })
			},
		},
		{
			name: "割合が上限のすぐ外は拒否",
			cursorParam: func(t *testing.T) string {
				return encodeBestSlotRankingRawCursorParam(t, func(p map[string]any) { p["BestPlayerPercentage"] = 100.1 })
			},
		},
		{
			name: "件数が負は拒否",
			cursorParam: func(t *testing.T) string {
				return encodeBestSlotRankingRawCursorParam(t, func(p map[string]any) { p["BestPlayerCount"] = -1 })
			},
		},
		{
			name: "レート帯が空は拒否",
			cursorParam: func(t *testing.T) string {
				return encodeBestSlotRankingRawCursorParam(t, func(p map[string]any) { p["RatingBand"] = "" })
			},
		},
		{
			name: "IDが不正は拒否",
			cursorParam: func(t *testing.T) string {
				return encodeBestSlotRankingRawCursorParam(t, func(p map[string]any) { p["SongDisplayID"] = "not-a-display-id" })
			},
		},
		{
			name: "難易度が不正は拒否",
			cursorParam: func(t *testing.T) string {
				return encodeBestSlotRankingRawCursorParam(t, func(p map[string]any) { p["Difficulty"] = "UNKNOWN" })
			},
		},
		{
			name: "WORLD'S ENDは拒否",
			cursorParam: func(t *testing.T) string {
				return encodeBestSlotRankingRawCursorParam(t, func(p map[string]any) { p["Difficulty"] = "WORLDSEND" })
			},
		},
		{
			name: "base64は正しいがJSON不正は拒否",
			cursorParam: func(t *testing.T) string {
				t.Helper()
				return base64.RawURLEncoding.EncodeToString([]byte("{invalid-json"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &recordingBestSlotRankingUsecaseStub{}
			e := newBestSlotRankingRecordingServer(stub)
			req := httptest.NewRequest(http.MethodGet, "/internal/best-slot-rankings?rating_band=17.0&cursor="+url.QueryEscape(tt.cursorParam(t)), nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			assert.False(t, stub.called)
			assert.Contains(t, rec.Body.String(), `"code":"validation_failed"`)
		})
	}
}
