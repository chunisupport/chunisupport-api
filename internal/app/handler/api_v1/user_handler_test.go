package api_v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type mockV1UserUsecase struct {
	getUserProfileWithRecordsFunc func(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*dto_internal.UserProfileWithRecordsDTO, error)
}

func (m *mockV1UserUsecase) GetUserProfile(ctx context.Context, username string, requester *entity.User) (*dto_internal.UserProfileDTO, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*dto_internal.UserUpdatedAtDTO, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*dto_internal.UserProfileWithRecordsDTO, error) {
	if m.getUserProfileWithRecordsFunc != nil {
		return m.getUserProfileWithRecordsFunc(ctx, username, requester, includeNoPlay)
	}
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*dto_internal.UserProfileRatingViewDTO, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*dto_internal.UserProfileRecordViewDTO, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]dto_internal.AdminUserListResponse, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) DeleteUser(ctx context.Context, requester *entity.User, username string) error {
	return nil
}

func TestV1UserHandler_GetUser(t *testing.T) {
	t.Run("非公開ユーザーはuser_not_foundを返す", func(t *testing.T) {
		// Given
		e := echo.New()
		mockUsecase := &mockV1UserUsecase{
			getUserProfileWithRecordsFunc: func(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*dto_internal.UserProfileWithRecordsDTO, error) {
				assert.Equal(t, "private-user", username)
				assert.True(t, includeNoPlay)
				return nil, usecase.ErrUserPrivate
			},
		}
		handler := NewV1UserHandler(mockUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/users/private-user?include_noplay=true", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("private-user")

		// When
		err := handler.GetUser(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUserNotFound, apiErr.Code)
			assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		}
	})
}

var _ usecase.UserUsecase = (*mockV1UserUsecase)(nil)
