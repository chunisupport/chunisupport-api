package api_internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFriendScoreComparisonUsecase struct {
	result     *usecase.FriendScoreComparisonResult
	err        error
	calls      int
	username   string
	difficulty string
}

func (s *stubFriendScoreComparisonUsecase) Get(ctx context.Context, selfUserID int, friendUsername string, difficulty string) (*usecase.FriendScoreComparisonResult, error) {
	s.calls++
	s.username = friendUsername
	s.difficulty = difficulty
	return s.result, s.err
}

func (s *stubFriendScoreComparisonUsecase) GetWorldsend(ctx context.Context, selfUserID int, friendUsername string) (*usecase.FriendScoreComparisonResult, error) {
	s.calls++
	s.username = friendUsername
	return s.result, s.err
}

func TestFriendScoreComparisonHandler_Worldsend専用の譜面情報を返す(t *testing.T) {
	levelStar := 4
	attribute := "蔵"
	stub := &stubFriendScoreComparisonUsecase{result: &usecase.FriendScoreComparisonResult{
		Difficulty: "WORLD'S END",
		Items: []usecase.FriendScoreComparisonItem{{
			Chart: usecase.FriendScoreComparisonChart{LevelStar: &levelStar, Attribute: &attribute},
		}},
	}}
	e := echo.New()
	handler := NewFriendScoreComparisonHandler(stub)
	req := httptest.NewRequest(http.MethodGet, "/internal/friend-comparisons/frienduser/worldsend", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "frienduser"}})
	c.Set("userEntity", &entity.User{ID: 7})

	err := handler.GetWorldsend(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, stub.calls)
	assert.Equal(t, "frienduser", stub.username)
	assert.JSONEq(t, `{"level_star":4,"attribute":"蔵"}`, extractFriendComparisonChartJSON(t, rec.Body.Bytes()))
	assert.Contains(t, rec.Body.String(), `"difficulty":"WORLD'S END"`)
}

func TestFriendScoreComparisonResponse_Worldsend譜面情報未設定はnullを返す(t *testing.T) {
	response := toFriendScoreComparisonResponse(&usecase.FriendScoreComparisonResult{
		Difficulty: "WORLD'S END",
		Items:      []usecase.FriendScoreComparisonItem{{}},
	})
	body, err := json.Marshal(response)
	require.NoError(t, err)
	assert.JSONEq(t, `{"level_star":null,"attribute":null}`, extractFriendComparisonChartJSON(t, body))
}

func extractFriendComparisonChartJSON(t *testing.T, body []byte) string {
	t.Helper()
	var response struct {
		Items []struct {
			Chart json.RawMessage `json:"chart"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &response))
	require.Len(t, response.Items, 1)
	return string(response.Items[0].Chart)
}

func TestFriendScoreComparisonHandler_正常レスポンスのDTO形状(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	clearLamp := "CLEAR"
	comboLamp := "FULL COMBO"
	chartConst, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	stub := &stubFriendScoreComparisonUsecase{result: &usecase.FriendScoreComparisonResult{
		Difficulty: "MASTER",
		Self:       usecase.FriendScoreComparisonUser{Username: "myuser", PlayerName: "MY PLAYER"},
		Friend:     usecase.FriendScoreComparisonUser{Username: "frienduser", PlayerName: "FRIEND"},
		Summary:    usecase.FriendScoreComparisonSummary{TotalCharts: 1, SelfWins: 1, SelfPlayed: 1, FriendPlayed: 1, BothPlayed: 1},
		Items: []usecase.FriendScoreComparisonItem{{
			Song:            usecase.FriendScoreComparisonSong{ID: "0000000000000001", Title: "楽曲名", Artist: "アーティスト名"},
			Chart:           usecase.FriendScoreComparisonChart{Const: chartConst, IsConstUnknown: false},
			Self:            usecase.FriendScoreComparisonRecord{IsPlayed: true, Score: 1009000, ClearLamp: &clearLamp, ComboLamp: &comboLamp, UpdatedAt: &updatedAt},
			Friend:          usecase.FriendScoreComparisonRecord{IsPlayed: false, Score: 0},
			ScoreDifference: 1009000,
			Result:          usecase.FriendScoreComparisonSelfWin,
		}},
	}}
	e := echo.New()
	handler := NewFriendScoreComparisonHandler(stub)
	c, rec := newFriendScoreComparisonContext(e, "frienduser", "MASTER")
	c.Set("userEntity", &entity.User{ID: 7})

	// When
	err = handler.Get(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, stub.calls)
	assert.Equal(t, "frienduser", stub.username)
	assert.Equal(t, "MASTER", stub.difficulty)
	assert.JSONEq(t, `{
		"difficulty": "MASTER",
		"self": {"username": "myuser", "player_name": "MY PLAYER"},
		"friend": {"username": "frienduser", "player_name": "FRIEND"},
		"summary": {
			"total_charts": 1,
			"self_wins": 1,
			"draws": 0,
			"friend_wins": 0,
			"self_played": 1,
			"friend_played": 1,
			"both_played": 1,
			"self_only_played": 0,
			"friend_only_played": 0,
			"both_unplayed": 0
		},
		"items": [{
			"song": {"id": "0000000000000001", "title": "楽曲名", "artist": "アーティスト名"},
			"chart": {"const": 14.5, "is_const_unknown": false},
			"self": {"is_played": true, "score": 1009000, "clear_lamp": "CLEAR", "combo_lamp": "FULL COMBO", "full_chain": null, "updated_at": "2026-07-20T10:00:00Z"},
			"friend": {"is_played": false, "score": 0, "clear_lamp": null, "combo_lamp": null, "full_chain": null, "updated_at": null},
			"score_difference": 1009000,
			"result": "SELF_WIN"
		}]
	}`, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "user_id")
	assert.NotContains(t, rec.Body.String(), "player_id")
}

func TestFriendScoreComparisonHandler_usernameの形式不正(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantCode string
	}{
		{name: "短すぎる", username: "ab", wantCode: apierror.CodeUsernameTooShort},
		{name: "使用できない文字", username: "Invalid", wantCode: apierror.CodeUsernameInvalidChar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &stubFriendScoreComparisonUsecase{}
			e := echo.New()
			handler := NewFriendScoreComparisonHandler(stub)
			c, _ := newFriendScoreComparisonContext(e, tt.username, "MASTER")
			c.Set("userEntity", &entity.User{ID: 1})

			// When
			err := handler.Get(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			assert.Zero(t, stub.calls)
		})
	}
}

func TestFriendScoreComparisonHandler_難易度の不足と不正(t *testing.T) {
	tests := []struct {
		name       string
		difficulty string
	}{
		{name: "不足", difficulty: ""},
		{name: "小文字", difficulty: "master"},
		{name: "混在", difficulty: "Master"},
		{name: "短縮形", difficulty: "MAS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &stubFriendScoreComparisonUsecase{}
			e := echo.New()
			handler := NewFriendScoreComparisonHandler(stub)
			c, _ := newFriendScoreComparisonContext(e, "frienduser", tt.difficulty)
			c.Set("userEntity", &entity.User{ID: 1})

			// When
			err := handler.Get(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, apierror.CodeInvalidDifficulty, apiErr.Code)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			assert.Zero(t, stub.calls)
		})
	}
}

func TestFriendScoreComparisonHandler_未認証(t *testing.T) {
	// Given
	stub := &stubFriendScoreComparisonUsecase{}
	e := echo.New()
	handler := NewFriendScoreComparisonHandler(stub)
	c, _ := newFriendScoreComparisonContext(e, "frienduser", "MASTER")

	// When
	err := handler.Get(c)

	// Then
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierror.CodeUnauthorized, apiErr.Code)
	assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
	assert.Zero(t, stub.calls)
}

func TestFriendScoreComparisonHandler_friend_not_foundはHTTP404(t *testing.T) {
	// Given
	stub := &stubFriendScoreComparisonUsecase{err: usecase.ErrFriendNotFound}
	e := echo.New()
	handler := NewFriendScoreComparisonHandler(stub)
	c, _ := newFriendScoreComparisonContext(e, "frienduser", "MASTER")
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := handler.Get(c)

	// Then
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierror.CodeFriendNotFound, apiErr.Code)
	assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
}

func TestFriendScoreComparisonHandler_比較不能はHTTP409(t *testing.T) {
	// Given
	stub := &stubFriendScoreComparisonUsecase{err: usecase.ErrFriendScoreComparisonUnavailable}
	e := echo.New()
	handler := NewFriendScoreComparisonHandler(stub)
	c, _ := newFriendScoreComparisonContext(e, "frienduser", "MASTER")
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := handler.Get(c)

	// Then
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, apierror.CodeFriendScoreComparisonUnavailable, apiErr.Code)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
}

func TestFriendScoreComparisonResponse_内部IDを公開しない(t *testing.T) {
	// Given
	body, err := json.Marshal(toFriendScoreComparisonResponse(&usecase.FriendScoreComparisonResult{
		Difficulty: "MASTER",
		Items:      []usecase.FriendScoreComparisonItem{},
	}))

	// Then
	require.NoError(t, err)
	assert.NotContains(t, string(body), "user_id")
	assert.NotContains(t, string(body), "song_sort_id")
	assert.Contains(t, string(body), `"items":[]`)
}

func newFriendScoreComparisonContext(e *echo.Echo, username string, difficulty string) (*echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, "/internal/friend-comparisons/"+username+"/charts/"+difficulty, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: username}, {Name: "difficulty", Value: difficulty}})
	return c, rec
}
