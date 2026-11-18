package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubInternalScoreHistoryUsecase struct {
	getStandard  func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error)
	getWorldsend func(context.Context, string, *entity.User, string) ([]usecase.ScoreHistoryEntry, error)
	getCourse    func(context.Context, string, *entity.User, string) ([]usecase.CourseScoreHistoryEntry, error)
}

func (s *stubInternalScoreHistoryUsecase) GetCourse(ctx context.Context, username string, requester *entity.User, displayID string) ([]usecase.CourseScoreHistoryEntry, error) {
	return s.getCourse(ctx, username, requester, displayID)
}

func (s *stubInternalScoreHistoryUsecase) GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
	return s.getStandard(ctx, username, requester, displayID, difficulty)
}

func (s *stubInternalScoreHistoryUsecase) GetWorldsend(ctx context.Context, username string, requester *entity.User, displayID string) ([]usecase.ScoreHistoryEntry, error) {
	return s.getWorldsend(ctx, username, requester, displayID)
}

func TestInternalScoreHistoryHandler_GetStandard(t *testing.T) {
	t.Run("Firebase認証ユーザーを閲覧者として渡す", func(t *testing.T) {
		// Given: スコア・各ランプ・日時が互いに異なる2件の履歴。項目の取り違えが分かるよう値をずらす。
		clearLampFirst := "CLEAR"
		comboLampFirst := "ALL JUSTICE"
		fullChainFirst := "FULL CHAIN GOLD"
		clearLampSecond := "ABSOLUTE"
		comboLampSecond := "FULL COMBO"
		requester := &entity.User{ID: 1}
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{
			getStandard: func(_ context.Context, username string, gotRequester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
				assert.Equal(t, "testuser", username)
				assert.Same(t, requester, gotRequester)
				assert.Equal(t, "0123456789abcdef", displayID)
				assert.Equal(t, "MASTER", difficulty)
				return []usecase.ScoreHistoryEntry{
					{
						Score:     1010000,
						ClearLamp: &clearLampFirst,
						ComboLamp: &comboLampFirst,
						FullChain: &fullChainFirst,
						UpdatedAt: time.Date(2026, 4, 27, 12, 34, 56, 0, time.UTC),
					},
					{
						Score:     1009000,
						ClearLamp: &clearLampSecond,
						ComboLamp: &comboLampSecond,
						FullChain: nil,
						UpdatedAt: time.Date(2026, 4, 26, 11, 22, 33, 0, time.UTC),
					},
				}, nil
			},
		})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/song/master/history", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/internal/users/:username/record/songs/:id/:difficulty/history")
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "id", Value: "0123456789abcdef"}, {Name: "difficulty", Value: "master"}})
		c.Set("userEntity", requester)

		// When: 通常譜面のスコア履歴を取得する。
		err := h.GetStandard(c)

		// Then: 履歴2件がそのままの順序と値で返る（全体比較なので余分な項目があれば失敗する）。
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{
			"entries": [
				{
					"score": 1010000,
					"clear_lamp": "CLEAR",
					"combo_lamp": "ALL JUSTICE",
					"full_chain": "FULL CHAIN GOLD",
					"updated_at": "2026-04-27T12:34:56Z"
				},
				{
					"score": 1009000,
					"clear_lamp": "ABSOLUTE",
					"combo_lamp": "FULL COMBO",
					"full_chain": null,
					"updated_at": "2026-04-26T11:22:33Z"
				}
			]
		}`, rec.Body.String())
	})

	t.Run("不正なユーザー名は400を返す", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/users/-/record/songs/song/master/history", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "-"}, {Name: "id", Value: "0123456789abcdef"}, {Name: "difficulty", Value: "master"}})

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("不正な難易度は400を返す", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/song/invalid/history", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "id", Value: "0123456789abcdef"}, {Name: "difficulty", Value: "invalid"}})

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidDifficulty, apiErr.Code)
	})
}

func TestInternalScoreHistoryHandler_GetWorldsend(t *testing.T) {
	// Given: スコア・各ランプ・日時が互いに異なる2件の履歴。項目の取り違えが分かるよう値をずらす。
	clearLampFirst := "HARD"
	comboLampFirst := "FULL COMBO"
	fullChainFirst := "FULL CHAIN PLATINUM"
	clearLampSecond := "FAILED"
	requester := &entity.User{ID: 1}
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{
		getWorldsend: func(_ context.Context, username string, gotRequester *entity.User, displayID string) ([]usecase.ScoreHistoryEntry, error) {
			assert.Equal(t, "testuser", username)
			assert.Same(t, requester, gotRequester)
			assert.Equal(t, "0123456789abcdef", displayID)
			return []usecase.ScoreHistoryEntry{
				{
					Score:     995000,
					ClearLamp: &clearLampFirst,
					ComboLamp: &comboLampFirst,
					FullChain: &fullChainFirst,
					UpdatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
				},
				{
					Score:     880000,
					ClearLamp: &clearLampSecond,
					ComboLamp: nil,
					FullChain: nil,
					UpdatedAt: time.Date(2026, 4, 30, 8, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/worldsend-songs/song/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/users/:username/record/worldsend-songs/:id/history")
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "id", Value: "0123456789abcdef"}})
	c.Set("userEntity", requester)

	// When: WORLD'S END譜面のスコア履歴を取得する。
	err := h.GetWorldsend(c)

	// Then: 履歴2件がそのままの順序と値で返る（全体比較なので余分な項目があれば失敗する）。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{
		"entries": [
			{
				"score": 995000,
				"clear_lamp": "HARD",
				"combo_lamp": "FULL COMBO",
				"full_chain": "FULL CHAIN PLATINUM",
				"updated_at": "2026-05-01T09:00:00Z"
			},
			{
				"score": 880000,
				"clear_lamp": "FAILED",
				"combo_lamp": null,
				"full_chain": null,
				"updated_at": "2026-04-30T08:00:00Z"
			}
		]
	}`, rec.Body.String())
}

func TestInternalScoreHistoryHandler_GetStandard_不正な楽曲IDはUsecaseへ渡さず400系エラーを返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/invalid/master/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "id", Value: "invalid"}, {Name: "difficulty", Value: "master"}})

	err := h.GetStandard(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
}

func TestInternalScoreHistoryHandler_GetWorldsend_不正な楽曲IDはUsecaseへ渡さず400系エラーを返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/worldsend-songs/invalid/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "id", Value: "invalid"}})

	err := h.GetWorldsend(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
}

func TestInternalScoreHistoryHandler_GetWorldsend_不正なユーザー名は400を返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/-/record/worldsend-songs/song/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "-"}, {Name: "id", Value: "0123456789abcdef"}})

	err := h.GetWorldsend(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
}

func TestInternalScoreHistoryHandler_GetStandard_空履歴は空配列で返す(t *testing.T) {
	// Given: 履歴が空の場合。
	requester := &entity.User{ID: 1}
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{
		getStandard: func(_ context.Context, username string, gotRequester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
			return []usecase.ScoreHistoryEntry{}, nil
		},
	})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/song/master/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/users/:username/record/songs/:id/:difficulty/history")
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "id", Value: "0123456789abcdef"}, {Name: "difficulty", Value: "master"}})
	c.Set("userEntity", requester)

	// When: 通常譜面のスコア履歴を取得する。
	err := h.GetStandard(c)

	// Then: entries が null ではなく空配列で返る（変換処理は通常譜面と WORLD'S END で共通）。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"entries": []}`, rec.Body.String())
}

func newInternalCourseHistoryContext(username, displayID string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/"+username+"/record/courses/"+displayID+"/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/users/:username/record/courses/:id/history")
	c.SetPathValues(echo.PathValues{{Name: "username", Value: username}, {Name: "id", Value: displayID}})
	return c, rec
}

func TestInternalScoreHistoryHandler_GetCourse_コース用の履歴形式で返す(t *testing.T) {
	// Given
	requester := &entity.User{ID: 1}
	updatedAt := time.Date(2026, time.November, 1, 12, 34, 56, 0, time.UTC)
	var gotUsername, gotDisplayID string
	var gotRequester *entity.User
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{
		getCourse: func(_ context.Context, username string, requester *entity.User, displayID string) ([]usecase.CourseScoreHistoryEntry, error) {
			gotUsername, gotRequester, gotDisplayID = username, requester, displayID
			return []usecase.CourseScoreHistoryEntry{
				{Score: 3010000, IsClear: true, ComboLamp: new("ALL JUSTICE"), UpdatedAt: updatedAt},
				{Score: 2900000, IsClear: false, ComboLamp: nil, UpdatedAt: updatedAt.Add(-time.Hour)},
			}, nil
		},
	})
	c, rec := newInternalCourseHistoryContext("testuser", "0123456789abcdef")
	c.Set("userEntity", requester)

	// When
	err := h.GetCourse(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "testuser", gotUsername)
	assert.Same(t, requester, gotRequester)
	assert.Equal(t, "0123456789abcdef", gotDisplayID)
	assert.JSONEq(t, `{"entries":[
		{"score":3010000,"is_clear":true,"combo_lamp":"ALL JUSTICE","updated_at":"2026-11-01T12:34:56Z"},
		{"score":2900000,"is_clear":false,"combo_lamp":null,"updated_at":"2026-11-01T11:34:56Z"}
	]}`, rec.Body.String())
}

func TestInternalScoreHistoryHandler_GetCourse_エラー(t *testing.T) {
	tests := []struct {
		name string
		// Given
		username   string
		displayID  string
		usecaseErr error
		// Then
		wantStatus int
		wantCode   string
	}{
		{name: "不正なコースIDはUsecaseへ渡さない", username: "testuser", displayID: "invalid", wantStatus: http.StatusUnprocessableEntity, wantCode: apierror.CodeValidationFailed},
		{name: "不正なユーザー名はUsecaseへ渡さない", username: "-", displayID: "0123456789abcdef", wantStatus: http.StatusBadRequest},
		{name: "非公開ユーザーは404", username: "testuser", displayID: "0123456789abcdef", usecaseErr: usecase.ErrUserPrivate, wantStatus: http.StatusNotFound},
		{name: "コースなしは404", username: "testuser", displayID: "0123456789abcdef", usecaseErr: repository.ErrCourseNotFound, wantStatus: http.StatusNotFound},
		{name: "履歴なしは404", username: "testuser", displayID: "0123456789abcdef", usecaseErr: usecase.ErrScoreHistoryNotFound, wantStatus: http.StatusNotFound, wantCode: apierror.CodeScoreHistoryNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: usecaseErr がないケースでUsecaseが呼ばれるとnil関数呼び出しで失敗する。
			stub := &stubInternalScoreHistoryUsecase{}
			if tt.usecaseErr != nil {
				stub.getCourse = func(context.Context, string, *entity.User, string) ([]usecase.CourseScoreHistoryEntry, error) {
					return nil, tt.usecaseErr
				}
			}
			c, _ := newInternalCourseHistoryContext(tt.username, tt.displayID)

			// When
			err := NewScoreHistoryHandler(stub).GetCourse(c)

			// Then
			apiErr, ok := err.(*apierror.APIError)
			require.True(t, ok)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, apiErr.Code)
			}
		})
	}
}
