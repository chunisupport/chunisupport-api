package api_internal_test

import (
	"context"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/stretchr/testify/mock"
)

// mockUserCredentialUsecase は usecase.UserCredentialUsecase のモックです。
type mockUserCredentialUsecase struct {
	mock.Mock
}

func (m *mockUserCredentialUsecase) GetUser(ctx context.Context, id int) (*dto_internal.UserDTO, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserDTO), args.Error(1)
}

func (m *mockUserCredentialUsecase) UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error {
	args := m.Called(ctx, userID, isPrivate)
	return args.Error(0)
}

func (m *mockUserCredentialUsecase) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	args := m.Called(ctx, userID, currentPassword, newPassword)
	return args.Error(0)
}

func (m *mockUserCredentialUsecase) DeleteUser(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func newProfileHandlerWithMocks(secureCookie bool, sameSite http.SameSite) (*api_internal.ProfileHandler, *mockAuthUsecase, *mockUserCredentialUsecase) {
	authMock := new(mockAuthUsecase)
	userCredentialMock := new(mockUserCredentialUsecase)

	h := api_internal.NewProfileHandler(authMock, userCredentialMock, secureCookie, sameSite)

	return h, authMock, userCredentialMock
}
