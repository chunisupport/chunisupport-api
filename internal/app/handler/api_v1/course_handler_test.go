package api_v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubV1CourseUsecase struct {
	usecase.CourseUsecase
	get func(context.Context, string, bool) (*usecase.CourseOutput, error)
}

func (s *stubV1CourseUsecase) Get(ctx context.Context, displayID string, includeDeleted bool) (*usecase.CourseOutput, error) {
	return s.get(ctx, displayID, includeDeleted)
}

func TestV1CourseHandler_GetUsesIDPathParameter(t *testing.T) {
	// Given: usecaseが全項目を持つ出力を返す（内部の数値IDや削除状態は公開しない）
	const id = "0123456789abcdef"
	var actualID string
	internalUpdatedAt := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	handler := NewV1CourseHandler(&stubV1CourseUsecase{
		get: func(ctx context.Context, displayID string, includeDeleted bool) (*usecase.CourseOutput, error) {
			actualID = displayID
			return &usecase.CourseOutput{
				ID:        999,
				DisplayID: displayID,
				Idx:       "50020",
				Name:      "CLASS I COURSE",
				Class:     "1",
				IsDeleted: true,
				UpdatedAt: &internalUpdatedAt,
			}, nil
		},
	})
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/v1/courses/"+id, nil), rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})

	// When: ハンドラーを実行する
	err := handler.Get(c)

	// Then: 成功ステータスとusecase出力に対応するJSON全体を返す
	require.NoError(t, err)
	assert.Equal(t, id, actualID)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"id":"0123456789abcdef","idx":"50020","name":"CLASS I COURSE","class":"1"}`, rec.Body.String())
}
