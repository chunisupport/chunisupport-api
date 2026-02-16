package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func TestUpdateSongs_ReleaseDateFormat(t *testing.T) {
	masterCache := &masterdata.Cache{}
	staticMasterCache := &masterdata.StaticCache{}

	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	newPutSongsContext := func(body string) echo.Context {
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
		rec := c.Response().Writer.(*httptest.ResponseRecorder)

		err := handler.UpdateSongs(c)
		if err != nil {
			t.Fatalf("UpdateSongs returned error: %v", err)
		}

		if rec.Code != http.StatusNoContent {
			t.Fatalf("Status code = %d, want %d", rec.Code, http.StatusNoContent)
		}

		if len(captured) != 1 {
			t.Fatalf("captured len = %d, want 1", len(captured))
		}
		if captured[0] == nil || captured[0].ReleasedAt == nil {
			t.Fatal("released_at should be parsed")
		}
		if got := captured[0].ReleasedAt.Format("2006-01-02"); got != "2024-01-01" {
			t.Fatalf("ReleasedAt = %s, want 2024-01-01", got)
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
			t.Fatal("UpdateSongs should return error")
		}

		apiErr, ok := err.(*apierror.APIError)
		if !ok {
			t.Fatalf("error type = %T, want *apierror.APIError", err)
		}
		if apiErr.Code != apierror.CodeBadRequest {
			t.Fatalf("api error code = %s, want %s", apiErr.Code, apierror.CodeBadRequest)
		}

		if called {
			t.Fatal("usecase should not be called")
		}
	})
}
