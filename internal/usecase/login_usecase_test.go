package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLoginUsecase_Login(t *testing.T) {
	tests := []struct {
		name      string
		idToken   string
		turnstile string
		remoteIP  string
		setup     func(authUsecase *mockFirebaseAuthUsecase, turnstileVerifier *mockTurnstileVerifier)
		wantUser  string
		wantErr   error
	}{
		{
			name:      "TurnstileとFirebaseトークンが有効ならユーザーDTOを返す",
			idToken:   "valid-token",
			turnstile: "turnstile-token",
			remoteIP:  "203.0.113.1",
			setup: func(authUsecase *mockFirebaseAuthUsecase, turnstileVerifier *mockTurnstileVerifier) {
				un := username.MustNewUserName("loginuser")
				user := &entity.User{ID: 10, Username: un, AccountTypeID: info.AccountTypePlayer}
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "203.0.113.1").Return(nil).Once()
				authUsecase.On("Authenticate", mock.Anything, "valid-token").Return(user, nil).Once()
			},
			wantUser: "loginuser",
		},
		{
			name:      "Turnstileトークンが空ならFirebase検証に進まない",
			idToken:   "valid-token",
			turnstile: " ",
			setup: func(authUsecase *mockFirebaseAuthUsecase, turnstileVerifier *mockTurnstileVerifier) {
			},
			wantErr: ErrInvalidTurnstileToken,
		},
		{
			name:      "Turnstile検証に失敗したらErrInvalidTurnstileTokenを返す",
			idToken:   "valid-token",
			turnstile: "invalid-turnstile-token",
			setup: func(authUsecase *mockFirebaseAuthUsecase, turnstileVerifier *mockTurnstileVerifier) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "invalid-turnstile-token", "").Return(ErrInvalidTurnstileToken).Once()
			},
			wantErr: ErrInvalidTurnstileToken,
		},
		{
			name:      "Firebaseトークンが無効ならErrInvalidIDTokenを返す",
			idToken:   "invalid-token",
			turnstile: "turnstile-token",
			setup: func(authUsecase *mockFirebaseAuthUsecase, turnstileVerifier *mockTurnstileVerifier) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				authUsecase.On("Authenticate", mock.Anything, "invalid-token").Return(nil, errors.Join(ErrInvalidIDToken, errors.New("invalid token"))).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
		{
			name:      "未登録Firebase UIDならErrInvalidIDTokenを返す",
			idToken:   "missing-user-token",
			turnstile: "turnstile-token",
			setup: func(authUsecase *mockFirebaseAuthUsecase, turnstileVerifier *mockTurnstileVerifier) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				authUsecase.On("Authenticate", mock.Anything, "missing-user-token").Return(nil, ErrInvalidIDToken).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
		{
			name:      "Authenticateが(nil, nil)を返す場合はErrInternalErrorを返す",
			idToken:   "nil-nil-token",
			turnstile: "turnstile-token",
			setup: func(authUsecase *mockFirebaseAuthUsecase, turnstileVerifier *mockTurnstileVerifier) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				authUsecase.On("Authenticate", mock.Anything, "nil-nil-token").Return(nil, nil).Once()
			},
			wantErr: ErrInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authUsecase := new(mockFirebaseAuthUsecase)
			turnstileVerifier := new(mockTurnstileVerifier)
			service := NewLoginUsecase(authUsecase, turnstileVerifier, newMockMasterCache(), maintenanceStateStub{})
			tt.setup(authUsecase, turnstileVerifier)

			got, err := service.Login(context.Background(), tt.idToken, tt.turnstile, tt.remoteIP)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.wantUser, got.Username)
				assert.Equal(t, "PLAYER", got.AccountType)
			}

			turnstileVerifier.AssertExpectations(t)
			authUsecase.AssertExpectations(t)
		})
	}
}

func TestLoginUsecase_Login_メンテナンス中のロール制御(t *testing.T) {
	tests := []struct {
		name          string
		accountTypeID int
		wantErr       error
	}{
		{
			name:          "ADMINはログインできる",
			accountTypeID: info.AccountTypeAdmin,
		},
		{
			name:          "EDITORはログインできる",
			accountTypeID: info.AccountTypeEditor,
		},
		{
			name:          "PLAYERはメンテナンスエラーになる",
			accountTypeID: info.AccountTypePlayer,
			wantErr:       ErrMaintenanceMode,
		},
		{
			name:          "EXTDEVはメンテナンスエラーになる",
			accountTypeID: info.AccountTypeExtDev,
			wantErr:       ErrMaintenanceMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			authUsecase := new(mockFirebaseAuthUsecase)
			turnstileVerifier := new(mockTurnstileVerifier)
			un := username.MustNewUserName("loginuser")
			user := &entity.User{ID: 10, Username: un, AccountTypeID: tt.accountTypeID}
			turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "203.0.113.1").Return(nil).Once()
			authUsecase.On("Authenticate", mock.Anything, "valid-token").Return(user, nil).Once()
			service := NewLoginUsecase(
				authUsecase,
				turnstileVerifier,
				newMockMasterCache(),
				maintenanceStateStub{enabled: true},
			)

			// When
			got, err := service.Login(context.Background(), "valid-token", "turnstile-token", "203.0.113.1")

			// Then
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, "loginuser", got.Username)
			}
			turnstileVerifier.AssertExpectations(t)
			authUsecase.AssertExpectations(t)
		})
	}
}

func TestLoginUsecase_Login_メンテナンス中も認証エラーを維持する(t *testing.T) {
	// Given
	authUsecase := new(mockFirebaseAuthUsecase)
	turnstileVerifier := new(mockTurnstileVerifier)
	maintenanceProvider := &countingMaintenanceStateStub{enabled: true}
	turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
	authUsecase.On("Authenticate", mock.Anything, "invalid-token").Return(nil, ErrInvalidIDToken).Once()
	service := NewLoginUsecase(
		authUsecase,
		turnstileVerifier,
		newMockMasterCache(),
		maintenanceProvider,
	)

	// When
	got, err := service.Login(context.Background(), "invalid-token", "turnstile-token", "")

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidIDToken)
	assert.NotErrorIs(t, err, ErrMaintenanceMode)
	assert.Nil(t, got)
	assert.Zero(t, maintenanceProvider.calls)
	turnstileVerifier.AssertExpectations(t)
	authUsecase.AssertExpectations(t)
}

func TestLoginUsecase_Login_メンテナンス中もTurnstileエラーを維持する(t *testing.T) {
	// Given
	authUsecase := new(mockFirebaseAuthUsecase)
	turnstileVerifier := new(mockTurnstileVerifier)
	maintenanceProvider := &countingMaintenanceStateStub{enabled: true}
	turnstileVerifier.On("VerifyTurnstile", mock.Anything, "invalid-turnstile-token", "").Return(ErrInvalidTurnstileToken).Once()
	service := NewLoginUsecase(
		authUsecase,
		turnstileVerifier,
		newMockMasterCache(),
		maintenanceProvider,
	)

	// When
	got, err := service.Login(context.Background(), "firebase-token", "invalid-turnstile-token", "")

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTurnstileToken)
	assert.NotErrorIs(t, err, ErrMaintenanceMode)
	assert.Nil(t, got)
	assert.Zero(t, maintenanceProvider.calls)
	turnstileVerifier.AssertExpectations(t)
	authUsecase.AssertNotCalled(t, "Authenticate", mock.Anything, mock.Anything)
}

func TestNewLoginUsecase_必須依存がnilならpanicする(t *testing.T) {
	tests := []struct {
		name                string
		authUsecase         FirebaseAuthUsecase
		turnstileVerifier   TurnstileVerifier
		accountTypeProvider AccountTypeProvider
		maintenanceProvider MaintenanceStateProvider
		wantPanic           string
	}{
		{
			name:                "FirebaseAuthUsecaseがnil",
			turnstileVerifier:   new(mockTurnstileVerifier),
			accountTypeProvider: newMockMasterCache(),
			maintenanceProvider: maintenanceStateStub{},
			wantPanic:           "loginUsecase: FirebaseAuthUsecase is nil",
		},
		{
			name:                "TurnstileVerifierがnil",
			authUsecase:         new(mockFirebaseAuthUsecase),
			accountTypeProvider: newMockMasterCache(),
			maintenanceProvider: maintenanceStateStub{},
			wantPanic:           "loginUsecase: TurnstileVerifier is nil",
		},
		{
			name:                "AccountTypeProviderがnil",
			authUsecase:         new(mockFirebaseAuthUsecase),
			turnstileVerifier:   new(mockTurnstileVerifier),
			maintenanceProvider: maintenanceStateStub{},
			wantPanic:           "loginUsecase: AccountTypeProvider is nil",
		},
		{
			name:                "MaintenanceStateProviderがnil",
			authUsecase:         new(mockFirebaseAuthUsecase),
			turnstileVerifier:   new(mockTurnstileVerifier),
			accountTypeProvider: newMockMasterCache(),
			wantPanic:           "loginUsecase: MaintenanceStateProvider is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithValue(t, tt.wantPanic, func() {
				NewLoginUsecase(
					tt.authUsecase,
					tt.turnstileVerifier,
					tt.accountTypeProvider,
					tt.maintenanceProvider,
				)
			})
		})
	}
}

type maintenanceStateStub struct {
	enabled bool
}

func (s maintenanceStateStub) Current() MaintenanceState {
	return MaintenanceState{Enabled: s.enabled}
}

type countingMaintenanceStateStub struct {
	enabled bool
	calls   int
}

func (s *countingMaintenanceStateStub) Current() MaintenanceState {
	s.calls++
	return MaintenanceState{Enabled: s.enabled}
}

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

func (m *mockFirebaseAuthUsecase) AuthenticateOptional(ctx context.Context, idToken string) (*entity.User, error) {
	args := m.Called(ctx, idToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

var _ FirebaseAuthUsecase = (*mockFirebaseAuthUsecase)(nil)
