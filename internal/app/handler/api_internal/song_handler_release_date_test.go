package api_internal

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

func TestUpdateSongs_ReleaseDateFormat(t *testing.T) {
	masterCache := &masterdata.Cache{}
	staticMasterCache := &masterdata.StaticCache{}

	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	newPutSongsContext := func(body string) *echo.Context {
		req := httptest.NewRequest(http.MethodPut, "/internal/songs", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	t.Run("YYYY-MM-DDは受理される", func(t *testing.T) {
		var captured []*api_internal.UpdateSongRequest
		mockUsecase := &testutil.MockSongUsecase{
			UpdateSongsFunc: func(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
				captured = requests
				return nil
			},
		}
		handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)

		c := newPutSongsContext(`[{"id":"1234567890123456","title":"test","artist":"artist","released_at":"2024-01-01"}]`)
		response, _ := echo.UnwrapResponse(c.Response())
		rec := response.ResponseWriter.(*httptest.ResponseRecorder)

		err := handler.UpdateSongs(c)
		if err != nil {
			require.Failf(t, "前提条件失敗", "UpdateSongs returned error: %v", err)
		}

		if rec.Code != http.StatusNoContent {
			require.Failf(t, "前提条件失敗", "Status code = %d, want %d", rec.Code, http.StatusNoContent)
		}

		if len(captured) != 1 {
			require.Failf(t, "前提条件失敗", "captured len = %d, want 1", len(captured))
		}
		if captured[0] == nil || captured[0].ReleasedAt == nil {
			require.Fail(t, "released_at should be parsed")
		}
		if got := captured[0].ReleasedAt.Format("2006-01-02"); got != "2024-01-01" {
			require.Failf(t, "前提条件失敗", "ReleasedAt = %s, want 2024-01-01", got)
		}
	})

	t.Run("時刻付きはbad_requestになる", func(t *testing.T) {
		called := false
		mockUsecase := &testutil.MockSongUsecase{
			UpdateSongsFunc: func(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
				called = true
				return nil
			},
		}
		handler := NewSongHandler(mockUsecase, &testutil.MockChartStatsUsecase{}, masterCache, staticMasterCache)

		c := newPutSongsContext(`[{"id":"1234567890123456","title":"test","artist":"artist","released_at":"2024-01-01T00:00:00Z"}]`)

		err := handler.UpdateSongs(c)
		if err == nil {
			require.Fail(t, "UpdateSongs should return error")
		}

		apiErr, ok := err.(*apierror.APIError)
		if !ok {
			require.Failf(t, "前提条件失敗", "error type = %T, want *apierror.APIError", err)
		}
		if apiErr.Code != apierror.CodeBadRequest {
			require.Failf(t, "前提条件失敗", "api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
		}

		if called {
			require.Fail(t, "usecase should not be called")
		}
	})
}
