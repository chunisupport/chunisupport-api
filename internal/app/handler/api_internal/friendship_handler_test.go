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
	acceptFunc       func(ctx context.Context, userID int, requesterID int) error
	rejectFunc       func(ctx context.Context, userID int, requesterID int) error
	cancelFunc       func(ctx context.Context, userID int, targetUserID int) error
	removeFunc       func(ctx context.Context, userID int, friendUserID int) error
	listReceivedFunc func(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error)
	listSentFunc     func(ctx context.Context, userID int) ([]*usecase.FriendshipUserOutput, error)
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

func (m *mockFriendshipUsecase) AcceptRequest(ctx context.Context, userID int, requesterID int) error {
	if m.acceptFunc != nil {
		return m.acceptFunc(ctx, userID, requesterID)
	}
	return nil
}

func (m *mockFriendshipUsecase) RejectRequest(ctx context.Context, userID int, requesterID int) error {
	if m.rejectFunc != nil {
		return m.rejectFunc(ctx, userID, requesterID)
	}
	return nil
}

func (m *mockFriendshipUsecase) CancelRequest(ctx context.Context, userID int, targetUserID int) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, userID, targetUserID)
	}
	return nil
}

func (m *mockFriendshipUsecase) Remove(ctx context.Context, userID int, friendUserID int) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, userID, friendUserID)
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
					UserID: 2, Username: "frienduser", PlayerLevel: &level, PlayerName: &name, Rating: &rating,
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
		assert.JSONEq(t, `{"items":[{"user_id":2,"username":"frienduser","player_level":42,"player_name":"PLAYER","rating":15.25,"requested_at":"`+now.Add(-time.Hour).Format(time.RFC3339Nano)+`","accepted_at":"`+now.Format(time.RFC3339Nano)+`"}]}`, rec.Body.String())
	})
}

func TestFriendshipHandler_AcceptRejectRemove(t *testing.T) {
	t.Run("承認はpathのuser_idを渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			acceptFunc: func(ctx context.Context, userID int, requesterID int) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, 2, requesterID)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests/2/accept", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "user_id", Value: "2"}})

		// When
		err := handler.AcceptRequest(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("拒否はpathのuser_idを渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			rejectFunc: func(ctx context.Context, userID int, requesterID int) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, 2, requesterID)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/friends/requests/2/reject", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "user_id", Value: "2"}})

		// When
		err := handler.RejectRequest(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("取り消しはpathのuser_idを送信先として渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			cancelFunc: func(ctx context.Context, userID int, targetUserID int) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, 2, targetUserID)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodDelete, "/internal/friends/requests/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "user_id", Value: "2"}})

		// When
		err := handler.CancelRequest(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("解除はpathのuser_idを渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewFriendshipHandler(&mockFriendshipUsecase{
			removeFunc: func(ctx context.Context, userID int, friendUserID int) error {
				called = true
				assert.Equal(t, 1, userID)
				assert.Equal(t, 2, friendUserID)
				return nil
			},
		})
		req := httptest.NewRequest(http.MethodDelete, "/internal/friends/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})
		c.SetPathValues(echo.PathValues{{Name: "user_id", Value: "2"}})

		// When
		err := handler.Remove(c)

		// Then
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}
