package api_v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubScoreHistoryUsecase struct {
	getStandard  func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error)
	getWorldsend func(context.Context, string, *entity.User, string) ([]usecase.ScoreHistoryEntry, error)
}

func (s *stubScoreHistoryUsecase) GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
	return s.getStandard(ctx, username, requester, displayID, difficulty)
}

func (s *stubScoreHistoryUsecase) GetWorldsend(ctx context.Context, username string, requester *entity.User, displayID string) ([]usecase.ScoreHistoryEntry, error) {
	if s.getWorldsend != nil {
		return s.getWorldsend(ctx, username, requester, displayID)
	}
	return nil, nil
}

func TestScoreHistoryHandler_GetStandard(t *testing.T) {
	t.Run("username未指定は400のvalidation_failed", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{})
		c := newScoreHistoryContext("/v1/songs/0123456789abcdef/score-history/master", "0123456789abcdef", "master", "")

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})

	t.Run("非対応難易度は400の専用コード", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{
			getStandard: func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error) {
				return nil, usecase.ErrScoreHistoryUnsupportedDifficulty
			},
		})
		c := newScoreHistoryContext("/v1/songs/0123456789abcdef/score-history/basic?username=testuser", "0123456789abcdef", "basic", "testuser")

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeScoreHistoryUnsupportedDifficulty, apiErr.Code)
	})
}

func TestScoreHistoryHandler_GetStandard_不正な楽曲IDはUsecaseへ渡さず400系エラーを返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{})
	c := newScoreHistoryContext("/v1/songs/invalid/score-history/master?username=testuser", "invalid", "master", "testuser")

	err := h.GetStandard(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
}

func TestScoreHistoryHandler_GetWorldsend_不正な楽曲IDはUsecaseへ渡さず400系エラーを返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{})
	c := newScoreHistoryContext("/v1/worldsend-songs/invalid/score-history?username=testuser", "invalid", "", "testuser")

	err := h.GetWorldsend(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
}

func newScoreHistoryContext(target, displayID, difficulty, username string) *echo.Context {
	c, _ := newScoreHistoryContextWithRecorder(target, displayID, difficulty, username)
	return c
}

// newScoreHistoryContextWithRecorder は成功レスポンスの本文検証用にレコーダーも返します。
func newScoreHistoryContextWithRecorder(target, displayID, difficulty, username string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: displayID}, {Name: "difficulty", Value: difficulty}})
	if username != "" {
		query := req.URL.Query()
		query.Set("username", username)
		req.URL.RawQuery = query.Encode()
	}
	return c, rec
}

func TestScoreHistoryHandler_GetStandard_正常系は履歴全体をそのまま返す(t *testing.T) {
	// Given: ランプ・スコア・日時が異なる非空履歴（通常側の配列変換をJSON全体で固定する）。
	firstAt := time.Date(2026, time.April, 27, 12, 34, 56, 0, time.UTC)
	secondAt := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	var gotUsername, gotDisplayID, gotDifficulty string
	h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{
		getStandard: func(_ context.Context, username string, _ *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
			gotUsername = username
			gotDisplayID = displayID
			gotDifficulty = difficulty
			return []usecase.ScoreHistoryEntry{
				{Score: 1009000, ClearLamp: new("ABSOLUTE"), ComboLamp: new("ALL JUSTICE"), FullChain: new("FULL CHAIN GOLD"), UpdatedAt: firstAt},
				{Score: 990000, ClearLamp: new("CLEAR"), ComboLamp: new("FULL COMBO"), FullChain: nil, UpdatedAt: secondAt},
			}, nil
		},
	})
	c, rec := newScoreHistoryContextWithRecorder("/v1/songs/0123456789abcdef/score-history/MASTER?username=testuser", "0123456789abcdef", "MASTER", "testuser")

	// When
	err := h.GetStandard(c)

	// Then: 経路の引数がそのまま渡り、ラッパー込みのJSON全体が仕様通りになる。
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "testuser", gotUsername)
	assert.Equal(t, "0123456789abcdef", gotDisplayID)
	assert.Equal(t, "MASTER", gotDifficulty)
	assert.JSONEq(t, `{"entries":[{"score":1009000,"clear_lamp":"ABSOLUTE","combo_lamp":"ALL JUSTICE","full_chain":"FULL CHAIN GOLD","updated_at":"2026-04-27T12:34:56Z"},{"score":990000,"clear_lamp":"CLEAR","combo_lamp":"FULL COMBO","full_chain":null,"updated_at":"2026-04-20T10:00:00Z"}]}`, rec.Body.String())
}

func TestScoreHistoryHandler_GetStandard_空履歴は空配列のラッパーを返す(t *testing.T) {
	// Given: 空入力でも entries キーを欠落・null にせず空配列で固定する。
	h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{
		getStandard: func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error) {
			return []usecase.ScoreHistoryEntry{}, nil
		},
	})
	c, rec := newScoreHistoryContextWithRecorder("/v1/songs/0123456789abcdef/score-history/MASTER?username=testuser", "0123456789abcdef", "MASTER", "testuser")

	// When
	err := h.GetStandard(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"entries":[]}`, rec.Body.String())
}

func TestScoreHistoryHandler_GetWorldsend_正常系は代表1件の契約を返す(t *testing.T) {
	// Given: WE側は代表1件で経路・引数・JSON契約だけを固定する（変換詳細の二重検証はしない）。
	updatedAt := time.Date(2026, time.April, 27, 12, 34, 56, 0, time.UTC)
	var gotUsername, gotDisplayID string
	h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{
		getWorldsend: func(_ context.Context, username string, _ *entity.User, displayID string) ([]usecase.ScoreHistoryEntry, error) {
			gotUsername = username
			gotDisplayID = displayID
			return []usecase.ScoreHistoryEntry{
				{Score: 950000, ClearLamp: new("CLEAR"), ComboLamp: nil, FullChain: nil, UpdatedAt: updatedAt},
			}, nil
		},
	})
	c, rec := newScoreHistoryContextWithRecorder("/v1/worldsend-songs/0123456789abcdef/score-history?username=testuser", "0123456789abcdef", "", "testuser")

	// When
	err := h.GetWorldsend(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "testuser", gotUsername)
	assert.Equal(t, "0123456789abcdef", gotDisplayID)
	assert.JSONEq(t, `{"entries":[{"score":950000,"clear_lamp":"CLEAR","combo_lamp":null,"full_chain":null,"updated_at":"2026-04-27T12:34:56Z"}]}`, rec.Body.String())
}
