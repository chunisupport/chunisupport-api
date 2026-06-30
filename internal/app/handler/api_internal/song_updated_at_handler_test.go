package api_internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSongHandler_GetSongsUpdatedAt(t *testing.T) {
	e := echo.New()
	now := time.Date(2026, 4, 9, 12, 34, 56, 0, time.UTC)
	mockUsecase := &testutil.MockSongUsecase{
		GetSongsUpdatedAtFunc: func(ctx context.Context) (*time.Time, error) {
			return &now, nil
		},
	}
	handler := NewSongHandler(mockUsecase, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/songs/updated-at", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetSongsUpdatedAt(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body dto_internal.SongUpdatedAtDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotNil(t, body.UpdatedAt)
	assert.True(t, now.Equal(*body.UpdatedAt))
}
