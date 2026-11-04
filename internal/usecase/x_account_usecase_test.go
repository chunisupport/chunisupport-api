package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockXAccountRepository struct {
	mock.Mock
}

func (m *mockXAccountRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.XAccount, error) {
	args := m.Called(ctx, exec, userID)
	account, _ := args.Get(0).(*entity.XAccount)
	return account, args.Error(1)
}

func (m *mockXAccountRepository) Save(ctx context.Context, exec repository.Executor, account *entity.XAccount) error {
	args := m.Called(ctx, exec, account)
	return args.Error(0)
}

type mockFirebaseProviderAccountFinder struct {
	mock.Mock
}

func (m *mockFirebaseProviderAccountFinder) FindProviderUID(ctx context.Context, firebaseUID string, providerID string) (string, error) {
	args := m.Called(ctx, firebaseUID, providerID)
	return args.String(0), args.Error(1)
}

type mockXUserClient struct {
	mock.Mock
}

func (m *mockXUserClient) GetAuthenticatedUser(ctx context.Context, accessToken string, accessTokenSecret string) (XAuthenticatedUser, error) {
	args := m.Called(ctx, accessToken, accessTokenSecret)
	user, _ := args.Get(0).(XAuthenticatedUser)
	return user, args.Error(1)
}

func newXAccountTestUser(firebaseUID *string) *entity.User {
	return &entity.User{ID: 7, FirebaseUID: firebaseUID}
}

func TestXAccountUsecase_Get(t *testing.T) {
	saved, err := entity.NewXAccount(7, "123", "chuni")
	require.NoError(t, err)

	tests := []struct {
		name      string
		found     *entity.XAccount
		findErr   error
		want      *XAccountOutput
		wantErrIs error
	}{
		{name: "保存済みならIDとユーザー名を返す", found: saved, want: &XAccountOutput{ID: "123", Username: "chuni"}},
		{name: "未保存ならnilを返す", found: nil, want: nil},
		{name: "取得失敗は内部エラーにする", findErr: errors.New("db down"), wantErrIs: ErrInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := new(mockXAccountRepository)
			repo.On("FindByUserID", mock.Anything, mock.Anything, 7).Return(tt.found, tt.findErr)
			uc := NewXAccountUsecase(nil, repo, new(mockFirebaseProviderAccountFinder), new(mockXUserClient))

			// When
			got, err := uc.Get(context.Background(), 7)

			// Then
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestXAccountUsecase_Sync_X連携中のアカウントと一致すれば保存する(t *testing.T) {
	// Given
	firebaseUID := "firebase-uid"
	repo := new(mockXAccountRepository)
	finder := new(mockFirebaseProviderAccountFinder)
	client := new(mockXUserClient)
	finder.On("FindProviderUID", mock.Anything, firebaseUID, info.FirebaseProviderIDX).Return("123", nil)
	client.On("GetAuthenticatedUser", mock.Anything, "token", "secret").Return(XAuthenticatedUser{ID: "123", Username: "chuni"}, nil)
	repo.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(account *entity.XAccount) bool {
		return account.UserID() == 7 && account.XUserID() == "123" && account.Username() == "chuni"
	})).Return(nil)
	uc := NewXAccountUsecase(nil, repo, finder, client)

	// When
	got, err := uc.Sync(context.Background(), newXAccountTestUser(&firebaseUID), XAccountSyncInput{AccessToken: " token ", AccessTokenSecret: "secret"})

	// Then
	require.NoError(t, err)
	assert.Equal(t, &XAccountOutput{ID: "123", Username: "chuni"}, got)
	repo.AssertExpectations(t)
}

func TestXAccountUsecase_Sync_保存しない場合(t *testing.T) {
	firebaseUID := "firebase-uid"

	tests := []struct {
		name        string
		user        *entity.User
		input       XAccountSyncInput
		linkedXID   string
		finderErr   error
		xUser       XAuthenticatedUser
		xErr        error
		wantErrIs   error
		callsFinder bool
		callsX      bool
	}{
		{
			name:      "トークンが空なら認証情報エラー",
			user:      newXAccountTestUser(&firebaseUID),
			input:     XAccountSyncInput{AccessToken: " ", AccessTokenSecret: "secret"},
			wantErrIs: ErrInvalidXCredentials,
		},
		{
			name:      "Firebase未連携ならX未連携エラー",
			user:      newXAccountTestUser(nil),
			input:     XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"},
			wantErrIs: ErrXAccountNotLinked,
		},
		{
			name:        "FirebaseにXが連携されていなければX未連携エラー",
			user:        newXAccountTestUser(&firebaseUID),
			input:       XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"},
			linkedXID:   "",
			wantErrIs:   ErrXAccountNotLinked,
			callsFinder: true,
		},
		{
			name:        "Firebaseの照会に失敗したら内部エラー",
			user:        newXAccountTestUser(&firebaseUID),
			input:       XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"},
			finderErr:   errors.New("firebase down"),
			wantErrIs:   ErrInternalError,
			callsFinder: true,
		},
		{
			name:        "Xがトークンを拒否したら認証情報エラー",
			user:        newXAccountTestUser(&firebaseUID),
			input:       XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"},
			linkedXID:   "123",
			xErr:        ErrInvalidXCredentials,
			wantErrIs:   ErrInvalidXCredentials,
			callsFinder: true,
			callsX:      true,
		},
		{
			name:        "X APIの想定外の失敗はX API利用不可エラー",
			user:        newXAccountTestUser(&firebaseUID),
			input:       XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"},
			linkedXID:   "123",
			xErr:        errors.New("connection reset"),
			wantErrIs:   ErrXAPIUnavailable,
			callsFinder: true,
			callsX:      true,
		},
		{
			name:        "連携中とは別のXアカウントのトークンなら認証情報エラー",
			user:        newXAccountTestUser(&firebaseUID),
			input:       XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"},
			linkedXID:   "123",
			xUser:       XAuthenticatedUser{ID: "999", Username: "someone"},
			wantErrIs:   ErrInvalidXCredentials,
			callsFinder: true,
			callsX:      true,
		},
		{
			name:        "Xが想定外のユーザー名を返したらX API利用不可エラー",
			user:        newXAccountTestUser(&firebaseUID),
			input:       XAccountSyncInput{AccessToken: "token", AccessTokenSecret: "secret"},
			linkedXID:   "123",
			xUser:       XAuthenticatedUser{ID: "123", Username: "invalid name"},
			wantErrIs:   ErrXAPIUnavailable,
			callsFinder: true,
			callsX:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := new(mockXAccountRepository)
			finder := new(mockFirebaseProviderAccountFinder)
			client := new(mockXUserClient)
			if tt.callsFinder {
				finder.On("FindProviderUID", mock.Anything, firebaseUID, info.FirebaseProviderIDX).Return(tt.linkedXID, tt.finderErr)
			}
			if tt.callsX {
				client.On("GetAuthenticatedUser", mock.Anything, "token", "secret").Return(tt.xUser, tt.xErr)
			}
			uc := NewXAccountUsecase(nil, repo, finder, client)

			// When
			got, err := uc.Sync(context.Background(), tt.user, tt.input)

			// Then
			assert.ErrorIs(t, err, tt.wantErrIs)
			assert.Nil(t, got)
			finder.AssertExpectations(t)
			client.AssertExpectations(t)
			repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
