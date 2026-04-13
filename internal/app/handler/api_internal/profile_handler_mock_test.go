package api_internal_test

import (
	"context"

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

func (m *mockUserCredentialUsecase) DeleteOwnAccount(ctx context.Context, userID int, reauthToken string) error {
	args := m.Called(ctx, userID, reauthToken)
	return args.Error(0)
}

func newProfileHandlerWithMocks() (*api_internal.ProfileHandler, *mockUserCredentialUsecase) {
	userCredentialMock := new(mockUserCredentialUsecase)

	h := api_internal.NewProfileHandler(userCredentialMock)

	return h, userCredentialMock
}
