package api_internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
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
		// Given: 公式RATINGの色分けのしきい値とその直前の値（比較は表示の丸め前の生の値で行われる）
		// When: 公開ハンドラー経由でバッジを取得する
		// Then: レスポンスの color に期待する帯の色が返る（色コードは info 定数を参照しない固定値）
		{
			name:        "公式RATINGが17.00なら虹EX色になる",
			rating:      new(17.00),
			wantMessage: "17.00",
			wantColor:   "#3597ed",
		},
		{
			name:        "公式RATINGが16.99なら虹色になる",
			rating:      new(16.99),
			wantMessage: "16.99",
			wantColor:   "#73b2f1",
		},
		{
			name:        "公式RATINGが16.00なら虹色になる",
			rating:      new(16.00),
			wantMessage: "16.00",
			wantColor:   "#73b2f1",
		},
		{
			name:        "公式RATINGが15.99ならプラチナ色になる",
			rating:      new(15.99),
			wantMessage: "15.99",
			wantColor:   "#ffeac9",
		},
		{
			name:        "公式RATINGが15.25ならプラチナ色になる",
			rating:      new(15.25),
			wantMessage: "15.25",
			wantColor:   "#ffeac9",
		},
		{
			name:        "公式RATINGが15.24なら金色になる",
			rating:      new(15.24),
			wantMessage: "15.24",
			wantColor:   "#fcd200",
		},
		{
			name:        "公式RATINGが14.50なら金色になる",
			rating:      new(14.50),
			wantMessage: "14.50",
			wantColor:   "#fcd200",
		},
		{
			name:        "公式RATINGが14.49なら銀色になる",
			rating:      new(14.49),
			wantMessage: "14.49",
			wantColor:   "#cccccc",
		},
		{
			name:        "公式RATINGが13.25なら銀色になる",
			rating:      new(13.25),
			wantMessage: "13.25",
			wantColor:   "#cccccc",
		},
		{
			name:        "公式RATINGが13.24なら銅色になる",
			rating:      new(13.24),
			wantMessage: "13.24",
			wantColor:   "#ff9f5e",
		},
		{
			name:        "公式RATINGが12.00なら銅色になる",
			rating:      new(12.00),
			wantMessage: "12.00",
			wantColor:   "#ff9f5e",
		},
		{
			name:        "公式RATINGが11.99なら紫色になる",
			rating:      new(11.99),
			wantMessage: "11.99",
			wantColor:   "#aa56d0",
		},
		{
			name:        "公式RATINGが10.00なら紫色になる",
			rating:      new(10.00),
			wantMessage: "10.00",
			wantColor:   "#aa56d0",
		},
		{
			name:        "公式RATINGが9.99なら赤色になる",
			rating:      new(9.99),
			wantMessage: "9.99",
			wantColor:   "#dd4343",
		},
		{
			name:        "公式RATINGが7.00なら赤色になる",
			rating:      new(7.00),
			wantMessage: "7.00",
			wantColor:   "#dd4343",
		},
		{
			name:        "公式RATINGが6.99なら橙色になる",
			rating:      new(6.99),
			wantMessage: "6.99",
			wantColor:   "#d8b800",
		},
		{
			name:        "公式RATINGが4.00なら橙色になる",
			rating:      new(4.00),
			wantMessage: "4.00",
			wantColor:   "#d8b800",
		},
		{
			name:        "公式RATINGが3.99なら緑色になる",
			rating:      new(3.99),
			wantMessage: "3.99",
			wantColor:   "#67ac09",
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

func (m *mockUserUsecase) GetPublicBadgePlayer(ctx context.Context, username string) (*entity.Player, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}
