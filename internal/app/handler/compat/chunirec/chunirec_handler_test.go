package chunirec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubChunirecUserUsecase struct {
	profile   *repository.ChunirecProfile
	records   []*usecase.ChunirecRecordOutput
	err       error
	username  string
	requester *entity.User
}

func (s *stubChunirecUserUsecase) GetProfile(_ context.Context, username string, requester *entity.User) (*repository.ChunirecProfile, error) {
	s.username, s.requester = username, requester
	return s.profile, s.err
}
func (s *stubChunirecUserUsecase) GetRecords(_ context.Context, username string, requester *entity.User) ([]*usecase.ChunirecRecordOutput, error) {
	s.username, s.requester = username, requester
	return s.records, s.err
}

var _ usecase.ChunirecUsecase = (*stubChunirecUserUsecase)(nil)

func TestChunirecHandler_GetUserShow_プレイヤー未連携ではHTTP200とnullを返す(t *testing.T) {
	// Given
	e := echo.New()
	handler := NewChunirecHandler(nil, &stubChunirecUserUsecase{}, nil, time.UTC)
	e.GET("/compat/chunirec/2.0/users/show", handler.GetUserShow)
	req := httptest.NewRequest(http.MethodGet, "/compat/chunirec/2.0/users/show?user_name=testuser", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `null`, rec.Body.String())
}

func TestChunirecHandler_互換レスポンス契約(t *testing.T) {
	constant, err := chartconstant.NewChartConstant(10.7)
	require.NoError(t, err)
	now := time.Date(2026, 10, 3, 3, 4, 5, 0, time.UTC)
	query := &stubChunirecUserUsecase{
		profile: &repository.ChunirecProfile{Name: "プレイヤー", Level: 100, Rating: new(17.23), Title: new("称号"), TitleRarity: new("platina"), UpdatedAt: now},
		records: []*usecase.ChunirecRecordOutput{{ChunirecRecord: &repository.ChunirecRecord{ID: "6a88218b1a936bd3", Title: "楽曲", Difficulty: "ULTIMA", Genre: "POPS & ANIME", Const: constant, Score: 0, ClearLamp: new("FAILED"), ComboLamp: new("ALL JUSTICE"), FullChain: new("FULL CHAIN GOLD"), UpdatedAt: now}, Rating: 0}},
	}
	tests := []struct{ path, expected string }{
		{"users/show", `{"user_id":0,"player_name":"プレイヤー","title":"称号","title_rarity":"platinum","level":100,"rating":"17.23","rating_max":"17.23","classemblem":null,"classemblem_base":null,"is_joined_team":null,"updated_at":"2026-10-03T12:04:05+09:00"}`},
		{"records/showall", `{"records":[{"id":"6a88218b1a936bd3","diff":"ULT","level":10.5,"title":"楽曲","const":10.7,"score":0,"rating":0,"is_const_unknown":false,"is_clear":false,"is_fullcombo":true,"is_alljustice":true,"is_fullchain":true,"genre":"POPS&ANIME","updated_at":"2026-10-03T12:04:05+0900","is_played":true}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			e := echo.New()
			h := NewChunirecHandler(nil, query, nil, time.FixedZone("Asia/Tokyo", 9*60*60))
			e.GET("/users/show", h.GetUserShow)
			e.GET("/records/showall", h.GetRecordsShowAll)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+tt.path+"?user_name=testuser", nil))
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.JSONEq(t, tt.expected, rec.Body.String())
			assert.Equal(t, "testuser", query.username)
		})
	}
}

func TestChunirecHandler_対象ユーザーと互換エラー(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		authenticated bool
		status        int
	}{
		{name: "トークン所有者", authenticated: true, status: 200},
		{name: "未認証", status: 503},
		{name: "非公開", err: usecase.ErrUserPrivate, authenticated: true, status: 404},
		{name: "不存在", err: usecase.ErrUserNotFound, authenticated: true, status: 404},
		{name: "切断", err: context.Canceled, authenticated: true, status: 499},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, records := range []bool{false, true} {
				e := echo.New()
				query := &stubChunirecUserUsecase{err: tt.err}
				h := NewChunirecHandler(nil, query, nil, time.UTC)
				path := "/users/show"
				fn := h.GetUserShow
				if records {
					path = "/records/showall"
					fn = h.GetRecordsShowAll
				}
				c := e.NewContext(httptest.NewRequest(http.MethodGet, path, nil), httptest.NewRecorder())
				requester := &entity.User{ID: 1, Username: usernametest.New(t, "tester")}
				if tt.authenticated {
					c.Set("userEntity", requester)
				}
				require.NoError(t, ChunirecErrorHandlerMiddleware()(fn)(c))
				response, _ := echo.UnwrapResponse(c.Response())
				assert.Equal(t, tt.status, response.Status)
				if tt.authenticated {
					assert.Equal(t, "tester", query.username)
					assert.Same(t, requester, query.requester)
				}
			}
		})
	}
}
func TestChunirecHandler_GetRecordsShowAll_未連携では空配列を返す(t *testing.T) {
	e := echo.New()
	h := NewChunirecHandler(nil, &stubChunirecUserUsecase{}, nil, time.UTC)
	e.GET("/records/showall", h.GetRecordsShowAll)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/records/showall?user_name=testuser", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"records":[]}`, rec.Body.String())
}
