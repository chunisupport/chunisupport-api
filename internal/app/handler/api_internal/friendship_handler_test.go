package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	vo_username "github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFriendshipUsecase struct {
	sendFunc         func(ctx context.Context, userID int, username string) error
	listFriendsFunc  func(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error)
	acceptFunc       func(ctx context.Context, userID int, requesterUsername string) error
	rejectFunc       func(ctx context.Context, userID int, requesterUsername string) error
	cancelFunc       func(ctx context.Context, userID int, targetUsername string) error
	removeFunc       func(ctx context.Context, userID int, friendUsername string) error
	listReceivedFunc func(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error)
	listSentFunc     func(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error)
}

func TestFriendshipHandler_SendRequest_不正なUsernameはHTTP422を返す(t *testing.T) {
	// Given
	e := newFriendshipTestEcho(t)
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	handler := NewFriendshipHandler(&mockFriendshipUsecase{})
	e.POST("/internal/friends/requests", func(c *echo.Context) error {
		c.Set("userEntity", &entity.User{ID: 1})
		return handler.SendRequest(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests", strings.NewReader(`{"username":"InvalidUser"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.JSONEq(t, `{"error":{"status":422,"code":"validation_failed","message":"入力値の形式を確認してください。"}}`, rec.Body.String())
}

func (m *mockFriendshipUsecase) SendRequest(ctx context.Context, userID int, username string) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, userID, username)
	}
	return nil
}

func (m *mockFriendshipUsecase) ListFriends(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error) {
	if m.listFriendsFunc != nil {
		return m.listFriendsFunc(ctx, userID)
	}
	return []*usecase.FriendshipUserOutput{}, nil
}

func (m *mockFriendshipUsecase) ListReceivedRequests(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error) {
	if m.listReceivedFunc != nil {
		return m.listReceivedFunc(ctx, userID)
	}
	return []*usecase.FriendshipUserOutput{}, nil
}

func (m *mockFriendshipUsecase) ListSentRequests(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error) {
	if m.listSentFunc != nil {
		return m.listSentFunc(ctx, userID)
	}
	return []*usecase.FriendshipUserOutput{}, nil
}

func (m *mockFriendshipUsecase) AcceptRequest(ctx context.Context, userID int, requesterUsername string) error {
	if m.acceptFunc != nil {
		return m.acceptFunc(ctx, userID, requesterUsername)
	}
	return nil
}

func (m *mockFriendshipUsecase) RejectRequest(ctx context.Context, userID int, requesterUsername string) error {
	if m.rejectFunc != nil {
		return m.rejectFunc(ctx, userID, requesterUsername)
	}
	return nil
}

func (m *mockFriendshipUsecase) CancelRequest(ctx context.Context, userID int, targetUsername string) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, userID, targetUsername)
	}
	return nil
}

func (m *mockFriendshipUsecase) Remove(ctx context.Context, userID int, friendUsername string) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, userID, friendUsername)
	}
	return nil
}

func TestFriendshipHandler_SendRequest(t *testing.T) {
	t.Run("usernameで申請を送る", func(t *testing.T) {
		// Given
		e := newFriendshipTestEcho(t)
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			sendFunc: func(ctx context.Context, userID int, username string) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, "targetuser", username)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests", strings.NewReader(`{"username":"targetuser"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		// When
		err := handler.SendRequest(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("不正なusernameはvalidation_failed", func(t *testing.T) {
		// Given
		e := newFriendshipTestEcho(t)
		handler := NewFriendshipHandler(&mockFriendshipUsecase{})
		req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests", strings.NewReader(`{"username":"InvalidUser"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		// When
		err := handler.SendRequest(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
			assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		}
	})
}

func newFriendshipTestEcho(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	v := validator.New()
	require.NoError(t, v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		_, err := vo_username.NewUserName(fl.Field().String())
		return err == nil
	}))
	e.Validator = &friendshipTestValidator{validator: v}
	return e
}

type friendshipTestValidator struct {
	validator *validator.Validate
}

func (v *friendshipTestValidator) Validate(i any) error {
	if err := v.validator.Struct(i); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	return nil
}

func TestFriendshipHandler_ListFriends(t *testing.T) {
	t.Run("一覧をitems配列で返す", func(t *testing.T) {
		// Given
		e := echo.New()
		now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
		level := 42
		name := "PLAYER"
		rating := 15.25
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			listFriendsFunc: func(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error) {
				assert.Equal(t, 1, userID)
				return []*usecase.FriendshipUserOutput{{
					Username: "frienduser", PlayerLevel: &level, PlayerName: &name, Rating: &rating,
					RequestedAt: now.Add(-time.Hour), AcceptedAt: &now,
				}}, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/internal/friends", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		// When
		err := handler.ListFriends(c)

		// Then
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"items":[{"username":"frienduser","player_level":42,"player_name":"PLAYER","rating":15.25,"is_private":false,"requested_at":"`+now.Add(-time.Hour).Format(time.RFC3339Nano)+`","accepted_at":"`+now.Format(time.RFC3339Nano)+`"}]}`, rec.Body.String())
	})
}

func TestFriendshipHandler_ListSentRequests_非公開ユーザーはUsernameだけ識別情報として返す(t *testing.T) {
	// Given
	e := echo.New()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	handler := NewFriendshipHandler(&mockFriendshipUsecase{
		listSentFunc: func(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error) {
			return []*usecase.FriendshipUserOutput{{
				Username:    "privateuser",
				IsPrivate:   true,
				RequestedAt: now,
			}}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/internal/friends/requests/sent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	// When
	err := handler.ListSentRequests(c)

	// Then
	require.NoError(t, err)
	assert.JSONEq(t, `{"items":[{"username":"privateuser","player_level":null,"player_name":null,"rating":null,"is_private":true,"requested_at":"`+now.Format(time.RFC3339Nano)+`"}]}`, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "user_id")
}

func TestFriendshipHandler_AcceptRejectRemove(t *testing.T) {
	t.Run("不正なpath usernameは詳細なusernameエラーを返す", func(t *testing.T) {
		// Given
		e := echo.New()
		handler := NewFriendshipHandler(&mockFriendshipUsecase{})
		req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests/InvalidUser/accept", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "InvalidUser"}})

		// When
		err := handler.AcceptRequest(c)

		// Then
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, apierror.CodeUsernameInvalidChar, apiErr.Code)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("承認はpathのusernameを渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			acceptFunc: func(ctx context.Context, userID int, requesterUsername string) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, "frienduser", requesterUsername)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests/frienduser/accept", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "frienduser"}})

		// When
		err := handler.AcceptRequest(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("拒否はpathのusernameを渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			rejectFunc: func(ctx context.Context, userID int, requesterUsername string) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, "frienduser", requesterUsername)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests/frienduser/reject", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "frienduser"}})

		// When
		err := handler.RejectRequest(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("取り消しはpathのusernameを送信先として渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			cancelFunc: func(ctx context.Context, userID int, targetUsername string) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, "frienduser", targetUsername)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodDelete, "/internal/friends/requests/frienduser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "frienduser"}})

		// When
		err := handler.CancelRequest(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("解除はpathのusernameを渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			removeFunc: func(ctx context.Context, userID int, friendUsername string) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, "frienduser", friendUsername)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodDelete, "/internal/friends/frienduser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "frienduser"}})

		// When
		err := handler.Remove(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}
