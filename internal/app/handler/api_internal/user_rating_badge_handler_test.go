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

func TestUserHandler_GetCalculatedRatingBadge(t *testing.T) {
	// Given: 計算RATINGバッジは小数4桁・レート色で表示し、値が無ければ no data、非公開・未存在は404にする
	// When: 公開ハンドラー経由でバッジを取得する
	// Then: レスポンス全体が仕様通りの固定値になり、usecaseへユーザー名が正しく渡る
	tests := []struct {
		name        string
		player      *entity.Player
		usecaseErr  error
		wantMessage string
		wantColor   string
		wantErr     error
	}{
		{
			// 小数4桁であること（小数2桁なら 17.12 になる）と、色がレート値由来であることが分かる値
			name:        "計算RATINGを小数第4位まで表示する",
			player:      &entity.Player{CalculatedRating: new(17.1234)},
			wantMessage: "17.1234",
			wantColor:   "#3597ed",
		},
		{
			name:        "計算RATINGが未取得ならno dataを表示する",
			player:      &entity.Player{},
			wantMessage: "no data",
			wantColor:   "lightgrey",
		},
		{
			name:       "存在しないユーザーは404にする",
			usecaseErr: usecase.ErrUserNotFound,
			wantErr:    apierror.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockUsecase := new(mockUserUsecase)
			mockUsecase.On("GetPublicBadgePlayer", mock.Anything, "testuser").Return(tt.player, tt.usecaseErr).Once()
			h := api_internal.NewUserHandler(mockUsecase)
			e := newTestEcho()
			req := httptest.NewRequest(http.MethodGet, "/badges/users/testuser/rating-calculated", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

			// When
			err := h.GetCalculatedRatingBadge(c)

			// Then
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

func TestUserHandler_GetOfficialOverpowerBadge(t *testing.T) {
	// Given: OVER POWER絶対値バッジは小数2桁・ポゼッション色で表示し、未連携は no data、非公開・未存在は404にする
	// When: 公開ハンドラー経由でバッジを取得する
	// Then: レスポンス全体が仕様通りの固定値になり、usecaseへユーザー名が正しく渡る
	tests := []struct {
		name        string
		player      *entity.Player
		usecaseErr  error
		wantMessage string
		wantColor   string
		wantErr     error
	}{
		{
			// 小数2桁の0埋め（12345.6 が 12345.60 になる）と、色がポゼッション由来（ID3は金色）であることが分かる値
			name:        "公式OVER POWERの絶対値を小数第2位まで表示する",
			player:      &entity.Player{OfficialOverpower: 12345.6, PossessionID: 3},
			wantMessage: "12345.60",
			wantColor:   "#fcd200",
		},
		{
			name:        "未知のポゼッションでも絶対値を維持し既定色を返す",
			player:      &entity.Player{OfficialOverpower: 12345.6, PossessionID: 6},
			wantMessage: "12345.60",
			wantColor:   "lightgrey",
		},
		{
			name:        "プレイヤー未連携ならno dataを表示する",
			player:      nil,
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
			// Given
			mockUsecase := new(mockUserUsecase)
			mockUsecase.On("GetPublicBadgePlayer", mock.Anything, "testuser").Return(tt.player, tt.usecaseErr).Once()
			h := api_internal.NewUserHandler(mockUsecase)
			e := newTestEcho()
			req := httptest.NewRequest(http.MethodGet, "/badges/users/testuser/overpower", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

			// When
			err := h.GetOfficialOverpowerBadge(c)

			// Then
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
			assert.Equal(t, "CHUNITHM OVER POWER", body["label"])
			assert.Equal(t, tt.wantMessage, body["message"])
			assert.Equal(t, tt.wantColor, body["color"])
			mockUsecase.AssertExpectations(t)
		})
	}
}

func TestUserHandler_GetOfficialOverpowerPercentBadge(t *testing.T) {
	// Given: OVER POWER割合バッジは小数2桁と % 接尾辞・ポゼッション色で表示し、割合未取得は no data、非公開・未存在は404にする
	// When: 公開ハンドラー経由でバッジを取得する
	// Then: レスポンス全体が仕様通りの固定値になり、usecaseへユーザー名が正しく渡る
	tests := []struct {
		name        string
		player      *entity.Player
		usecaseErr  error
		wantMessage string
		wantColor   string
		wantErr     error
	}{
		{
			// 絶対値（12345.6）とは異なる割合（76.25）が % 付きで出ることと、色がポゼッション由来（ID4はプラチナ色）であることが分かる値
			name:        "公式OVER POWERの割合を%付きで表示する",
			player:      &entity.Player{OfficialOverpower: 12345.6, OfficialOverpowerPercent: new(76.25), PossessionID: 4},
			wantMessage: "76.25%",
			wantColor:   "#ffeac9",
		},
		{
			name:        "未知のポゼッションでも割合を維持し既定色を返す",
			player:      &entity.Player{OfficialOverpowerPercent: new(76.25), PossessionID: 6},
			wantMessage: "76.25%",
			wantColor:   "lightgrey",
		},
		{
			// 割合が無ければ絶対値にフォールバックせず no data になり、ポゼッション色も漏れない
			name:        "割合が未取得ならno dataを表示する",
			player:      &entity.Player{OfficialOverpower: 12345.6, PossessionID: 3},
			wantMessage: "no data",
			wantColor:   "lightgrey",
		},
		{
			name:       "存在しないユーザーは404にする",
			usecaseErr: usecase.ErrUserNotFound,
			wantErr:    apierror.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockUsecase := new(mockUserUsecase)
			mockUsecase.On("GetPublicBadgePlayer", mock.Anything, "testuser").Return(tt.player, tt.usecaseErr).Once()
			h := api_internal.NewUserHandler(mockUsecase)
			e := newTestEcho()
			req := httptest.NewRequest(http.MethodGet, "/badges/users/testuser/overpower-percent", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

			// When
			err := h.GetOfficialOverpowerPercentBadge(c)

			// Then
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
			assert.Equal(t, "CHUNITHM OVER POWER", body["label"])
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
