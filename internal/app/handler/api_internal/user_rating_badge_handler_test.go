package api_internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserHandler_GetOfficialRatingBadge(t *testing.T) {
	tests := []struct {
		name        string
		rating      *float64
		usecaseErr  error
		wantMessage string
		wantColor   string
		wantErr     error
	}{
		{
			name:        "公式RATINGを小数第2位まで表示する",
			rating:      func() *float64 { value := 17.2; return &value }(),
			wantMessage: "17.20",
			wantColor:   "#3597ed",
		},
		{
			name:        "プレイヤー未連携ならno dataを表示する",
			wantMessage: "no data",
			wantColor:   "lightgrey",
		},
		{
			name:       "非公開ユーザーは404にする",
			usecaseErr: usecase.ErrUserPrivate,
			wantErr:    apierror.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(mockUserUsecase)
			mockUsecase.On("GetPublicOfficialRating", mock.Anything, "testuser").Return(tt.rating, tt.usecaseErr).Once()
			h := api_internal.NewUserHandler(mockUsecase)
			e := newTestEcho()
			req := httptest.NewRequest(http.MethodGet, "/badges/users/testuser/rating", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

			err := h.GetOfficialRatingBadge(c)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				mockUsecase.AssertExpectations(t)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, float64(1), body["schemaVersion"])
			assert.Equal(t, "CHUNITHM RATING", body["label"])
			assert.Equal(t, tt.wantMessage, body["message"])
			assert.Equal(t, tt.wantColor, body["color"])
			mockUsecase.AssertExpectations(t)
		})
	}
}

func (m *mockUserUsecase) GetPublicOfficialRating(ctx context.Context, username string) (*float64, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*float64), args.Error(1)
}
