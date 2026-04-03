package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockFirebaseAuthUsecase struct {
	mock.Mock
}

func (m *mockFirebaseAuthUsecase) Authenticate(ctx context.Context, idToken string) (*entity.User, error) {
	args := m.Called(ctx, idToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

type mockSessionIssuer struct {
	mock.Mock
}

func (m *mockSessionIssuer) IssueSession(ctx context.Context, user *entity.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func TestFirebaseLoginUsecase_LoginWithFirebase(t *testing.T) {
	tests := []struct {
		name      string
		idToken   string
		setup     func(authUsecase *mockFirebaseAuthUsecase, sessionIssuer *mockSessionIssuer)
		wantToken string
		wantErr   error
	}{
		{
			name:    "Firebase認証に成功したらセッションを発行する",
			idToken: "valid-token",
			setup: func(authUsecase *mockFirebaseAuthUsecase, sessionIssuer *mockSessionIssuer) {
				user := &entity.User{ID: 1}
				authUsecase.On("Authenticate", mock.Anything, "valid-token").Return(user, nil).Once()
				sessionIssuer.On("IssueSession", mock.Anything, user).Return("jwt-token", nil).Once()
			},
			wantToken: "jwt-token",
		},
		{
			name:    "Firebase認証エラーはそのまま返す",
			idToken: "invalid-token",
			setup: func(authUsecase *mockFirebaseAuthUsecase, sessionIssuer *mockSessionIssuer) {
				authUsecase.On("Authenticate", mock.Anything, "invalid-token").Return(nil, ErrInvalidIDToken).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
		{
			name:    "セッション発行エラーはそのまま返す",
			idToken: "session-error-token",
			setup: func(authUsecase *mockFirebaseAuthUsecase, sessionIssuer *mockSessionIssuer) {
				user := &entity.User{ID: 2}
				authUsecase.On("Authenticate", mock.Anything, "session-error-token").Return(user, nil).Once()
				sessionIssuer.On("IssueSession", mock.Anything, user).Return("", ErrInternalError).Once()
			},
			wantErr: ErrInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			authUsecase := new(mockFirebaseAuthUsecase)
			sessionIssuer := new(mockSessionIssuer)
			service := NewFirebaseLoginUsecase(authUsecase, sessionIssuer)
			tt.setup(authUsecase, sessionIssuer)

			// When
			token, err := service.LoginWithFirebase(context.Background(), tt.idToken)

			// Then
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}

			authUsecase.AssertExpectations(t)
			sessionIssuer.AssertExpectations(t)
		})
	}
}
