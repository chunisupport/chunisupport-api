package api_internal_test

import (
	"context"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/stretchr/testify/mock"
)

// mockAuthUsecase は usecase.AuthUsecase のモックです。
type mockAuthUsecase struct {
	mock.Mock
}

func (m *mockAuthUsecase) Register(ctx context.Context, username, password string) (*dto_internal.UserDTO, string, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*dto_internal.UserDTO), args.String(1), args.Error(2)
}

func (m *mockAuthUsecase) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *mockAuthUsecase) Logout(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockAuthUsecase) Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error) {
	args := m.Called(ctx, userID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func newAuthHandlerWithAuthMock(secureCookie bool, sameSite http.SameSite) (*api_internal.AuthHandler, *mockAuthUsecase) {
	authMock := new(mockAuthUsecase)
	return api_internal.NewAuthHandler(authMock, secureCookie, sameSite), authMock
}
