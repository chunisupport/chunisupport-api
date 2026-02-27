package api_internal_test

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/stretchr/testify/mock"
)

// mockRecoveryUsecase は usecase.RecoveryUsecase のモックです。
type mockRecoveryUsecase struct {
	mock.Mock
}

func (m *mockRecoveryUsecase) IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockRecoveryUsecase) RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error {
	args := m.Called(ctx, recoveryCode, newPassword)
	return args.Error(0)
}

func newRecoveryHandlerWithMock() (*api_internal.RecoveryHandler, *mockRecoveryUsecase) {
	recoveryMock := new(mockRecoveryUsecase)
	return api_internal.NewRecoveryHandler(recoveryMock), recoveryMock
}
