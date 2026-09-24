package api_v1

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v1 API では Wiki ページタイトルを扱わないため、更新時は常に既存値を維持し、指定は未知フィールドとして拒否します。
func TestV1SongHandler_UpdateSongs_WikiPageTitle(t *testing.T) {
	tests := []struct {
		name string
		// Given: リクエストボディ
		body string
		// Then: エラーコード（空文字は成功）
		expectedErrCode string
	}{
		{
			name:            "wiki_page_titleを指定した場合はbad_requestを返す",
			body:            `[{"id":"1234567890abcdef","title":"曲","artist":"A","wiki_page_title":"曲(CHUNITHM)"}]`,
			expectedErrCode: apierror.CodeBadRequest,
		},
		{
			name: "wiki_page_titleを指定しない場合は既存値維持としてユースケースへ渡す",
			body: `[{"id":"1234567890abcdef","title":"曲","artist":"A"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var received []*usecase.UpdateSongInput
			handler := NewV1SongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					received = requests
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			req := httptest.NewRequest(http.MethodPut, "/v1/songs", bytes.NewBufferString(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			// When
			err := handler.UpdateSongs(e.NewContext(req, httptest.NewRecorder()))

			// Then
			if tt.expectedErrCode != "" {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.expectedErrCode, apiErr.Code)
				assert.Nil(t, received)
				return
			}
			require.NoError(t, err)
			require.Len(t, received, 1)
			assert.False(t, received[0].UpdateWikiPageTitle)
			assert.Nil(t, received[0].WikiPageTitle)
		})
	}
}
