package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlayerFavoriteSongUsecase struct {
	listFunc   func(ctx context.Context, username string, requester *entity.User) ([]*usecase.PlayerFavoriteSongOutput, error)
	addFunc    func(ctx context.Context, userID int, displayID displayid.DisplayID) error
	removeFunc func(ctx context.Context, userID int, displayID displayid.DisplayID) error
}

func (m *mockPlayerFavoriteSongUsecase) List(ctx context.Context, username string, requester *entity.User) ([]*usecase.PlayerFavoriteSongOutput, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, username, requester)
	}
	return nil, nil
}

func (m *mockPlayerFavoriteSongUsecase) Add(ctx context.Context, userID int, displayID displayid.DisplayID) error {
	if m.addFunc != nil {
		return m.addFunc(ctx, userID, displayID)
	}
	return nil
}

func (m *mockPlayerFavoriteSongUsecase) Remove(ctx context.Context, userID int, displayID displayid.DisplayID) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, userID, displayID)
	}
	return nil
}

func TestPlayerFavoriteSongHandler_List(t *testing.T) {
	t.Run("お気に入り一覧を返す", func(t *testing.T) {
		e := echo.New()
		requester := &entity.User{ID: 1}
		now := time.Now()
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{
			listFunc: func(ctx context.Context, username string, gotRequester *entity.User) ([]*usecase.PlayerFavoriteSongOutput, error) {
				assert.Equal(t, "testuser", username)
				assert.Same(t, requester, gotRequester)
				return []*usecase.PlayerFavoriteSongOutput{
					{DisplayID: "1234567890abcdef", Title: "テスト楽曲", Jacket: strPtrFH("test.jpg"), FavoritedAt: now},
				}, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/favorite-songs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", requester)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"items":[{"display_id":"1234567890abcdef","title":"テスト楽曲","jacket":"test.jpg","favorited_at":"`+now.Format(time.RFC3339Nano)+`"}]}`, rec.Body.String())
	})

	t.Run("不正なusernameは拒否する", func(t *testing.T) {
		e := echo.New()
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{})
		req := httptest.NewRequest(http.MethodGet, "/internal/users/InvalidUser/favorite-songs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "InvalidUser"}})

		err := handler.List(c)
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUsernameInvalidChar, apiErr.Code)
		}
	})

	t.Run("空一覧がitems配列になる", func(t *testing.T) {
		e := echo.New()
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{
			listFunc: func(ctx context.Context, username string, requester *entity.User) ([]*usecase.PlayerFavoriteSongOutput, error) {
				return []*usecase.PlayerFavoriteSongOutput{}, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/favorite-songs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"items":[]}`, rec.Body.String())
	})
}

func TestPlayerFavoriteSongHandler_Add(t *testing.T) {
	t.Run("登録が204を返す", func(t *testing.T) {
		e := echo.New()
		e.Validator = &testValidator{validator: validator.New()}

		called := false
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{
			addFunc: func(ctx context.Context, userID int, did displayid.DisplayID) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, "1234567890abcdef", did.String())
				return nil
			},
		})
		body := `{"display_id":"1234567890abcdef"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/favorite-songs", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := handler.Add(c)
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("認証ユーザー不在でエラー", func(t *testing.T) {
		e := echo.New()
		e.Validator = &testValidator{validator: validator.New()}
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{})
		body := `{"display_id":"1234567890abcdef"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/favorite-songs", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Add(c)
		require.Error(t, err)
	})

	t.Run("不正なJSONでbad_request", func(t *testing.T) {
		e := echo.New()
		e.Validator = &testValidator{validator: validator.New()}
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{})
		body := `{invalid json}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/favorite-songs", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := handler.Add(c)
		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	})

	t.Run("不正なdisplay_idでvalidation_failed", func(t *testing.T) {
		e := echo.New()
		e.Validator = &testValidator{validator: validator.New()}
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{})
		body := `{"display_id":"invalid"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/favorite-songs", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := handler.Add(c)
		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})

	t.Run("未知フィールドを拒否する", func(t *testing.T) {
		e := echo.New()
		e.Validator = &testValidator{validator: validator.New()}
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{})
		body := `{"display_id":"1234567890abcdef","unknown_field":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/favorite-songs", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := handler.Add(c)
		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	})

	t.Run("上限エラーが400になる", func(t *testing.T) {
		e := echo.New()
		e.Validator = &testValidator{validator: validator.New()}
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{
			addFunc: func(ctx context.Context, userID int, did displayid.DisplayID) error {
				return usecase.ErrPlayerFavoriteSongLimitExceeded
			},
		})
		body := `{"display_id":"1234567890abcdef"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/me/favorite-songs", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := handler.Add(c)
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
			assert.Equal(t, apierror.CodeFavoriteSongLimitExceeded, apiErr.Code)
		}
	})
}

func TestPlayerFavoriteSongHandler_Remove(t *testing.T) {
	t.Run("解除が204を返す", func(t *testing.T) {
		e := echo.New()
		called := false
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{
			removeFunc: func(ctx context.Context, userID int, did displayid.DisplayID) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, "1234567890abcdef", did.String())
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodDelete, "/internal/me/favorite-songs/1234567890abcdef", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "displayid", Value: "1234567890abcdef"}})

		err := handler.Remove(c)
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("不正なdisplay_idでvalidation_failed", func(t *testing.T) {
		e := echo.New()
		handler := NewPlayerFavoriteSongHandler(&mockPlayerFavoriteSongUsecase{})
		req := httptest.NewRequest(http.MethodDelete, "/internal/me/favorite-songs/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "displayid", Value: "invalid"}})

		err := handler.Remove(c)
		require.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})
}

func strPtrFH(s string) *string {
	return &s
}
