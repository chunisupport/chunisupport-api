package chunirec

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunirecHandler_CancellationLogging(t *testing.T) {
	for _, endpoint := range []struct {
		name    string
		handler func(*ChunirecHandler, *echo.Context) error
	}{
		{"レコード", (*ChunirecHandler).GetRecordsShowAll},
		{"プロフィール", (*ChunirecHandler).GetUserShow},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			var output bytes.Buffer
			original := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
			t.Cleanup(func() { slog.SetDefault(original) })
			query := &stubChunirecUserUsecase{err: errors.Join(errors.New("query failed"), context.Canceled)}
			h := NewChunirecHandler(nil, query, nil, time.UTC)
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/?user_name=tester", nil), rec)
			c.Set("userEntity", &entity.User{ID: 1})

			err := endpoint.handler(h, c)

			require.ErrorIs(t, err, context.Canceled)
			require.NoError(t, ChunirecErrorHandlerMiddleware()(func(*echo.Context) error { return err })(c))
			assert.Equal(t, info.StatusClientClosedRequest, rec.Code)
			assert.Empty(t, rec.Body.String())
			assert.Equal(t, 1, strings.Count(output.String(), "level=INFO"))
			assert.NotContains(t, output.String(), "level=WARN")
			assert.NotContains(t, output.String(), "level=ERROR")
		})
	}
}
