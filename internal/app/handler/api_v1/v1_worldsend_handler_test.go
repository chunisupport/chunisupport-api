package api_v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestV1WorldsendHandler_GetWorldsendSongRejectsInvalidDisplayID(t *testing.T) {
	e := echo.New()
	called := false
	handler := NewV1WorldsendHandler(&testutil.MockWorldsendUsecase{
		GetWorldsendSongByDisplayIDFunc: func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.WorldsendSongWithChart, error) {
			called = true
			return nil, nil
		},
	}, &masterdata.Cache{})

	req := httptest.NewRequest(http.MethodGet, "/v1/worldsend-songs/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "displayid", Value: "invalid"}})

	err := handler.GetWorldsendSong(c)

	var apiErr *apierror.APIError
	if assert.ErrorAs(t, err, &apiErr) {
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	}
	assert.False(t, called)
}
