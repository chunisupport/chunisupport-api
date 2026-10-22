package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/reauthtoken/reauthtokentest"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type recordingFirebaseUserDeleter struct {
	deletedUIDs []string
	err         error
}

func (d *recordingFirebaseUserDeleter) DeleteUser(_ context.Context, uid string) error {
	d.deletedUIDs = append(d.deletedUIDs, uid)
	return d.err
}

func TestUserCredentialUsecase_ChangeFirebaseAccount(t *testing.T) {
	currentTime := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	recentAuthTime := currentTime.Add(-time.Minute)
	oldUID := "old-firebase-uid"
	newUID := "new-firebase-uid"

	newUsecase := func(repo *MockUserRepository, verifier *mockRecentSignInVerifier, deleter *recordingFirebaseUserDeleter) UserCredentialUsecase {
		uc := NewUserCredentialUsecaseWithFirebaseServices(&MockExecutor{}, &mockTransactionManager{}, repo, &stubPlayerRecordRepository{}, &stubGoalRepo{}, verifier, deleter, newMockMasterCache())
		uc.(*userCredentialUsecaseImpl).clock = fixedClock{now: currentTime}
		return uc
	}
	newUser := func() *entity.User {
		uid := oldUID
		return &entity.User{ID: 1, Username: usernametest.New(t, "testuser"), FirebaseUID: &uid, AccountTypeID: 1}
	}

	t.Run("両アカウントの再認証後にFirebase UIDを差し替えて旧Firebaseユーザーを削除する", func(t *testing.T) {
		// Given
		repo := new(MockUserRepository)
		verifier := new(mockRecentSignInVerifier)
		deleter := &recordingFirebaseUserDeleter{}
		verifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: oldUID, AuthTime: recentAuthTime}, nil).Once()
		verifier.On("VerifyRecentSignIn", mock.Anything, "new-account-token").Return(&RecentSignInInfo{UID: newUID, AuthTime: recentAuthTime}, nil).Once()
		repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(newUser(), nil).Once()
		repo.On("LinkFirebaseUID", mock.Anything, mock.Anything, 1, ptrString(oldUID), newUID, mock.AnythingOfType("time.Time")).Return(nil).Once()
		uc := newUsecase(repo, verifier, deleter)

		// When
		err := uc.ChangeFirebaseAccount(context.Background(), 1, reauthtokentest.New(t, "reauth-token"), reauthtokentest.New(t, "new-account-token"))

		// Then
		assert.NoError(t, err)
		assert.Equal(t, []string{oldUID}, deleter.deletedUIDs)
		repo.AssertExpectations(t)
		verifier.AssertExpectations(t)
	})

	t.Run("旧Firebaseユーザーの削除に失敗しても差し替えは成功として扱う", func(t *testing.T) {
		// Given
		repo := new(MockUserRepository)
		verifier := new(mockRecentSignInVerifier)
		deleter := &recordingFirebaseUserDeleter{err: errors.New("firebase unavailable")}
		verifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: oldUID, AuthTime: recentAuthTime}, nil).Once()
		verifier.On("VerifyRecentSignIn", mock.Anything, "new-account-token").Return(&RecentSignInInfo{UID: newUID, AuthTime: recentAuthTime}, nil).Once()
		repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(newUser(), nil).Once()
		repo.On("LinkFirebaseUID", mock.Anything, mock.Anything, 1, ptrString(oldUID), newUID, mock.AnythingOfType("time.Time")).Return(nil).Once()
		uc := newUsecase(repo, verifier, deleter)
		logBuffer := captureDefaultSlog(t)

		// When
		err := uc.ChangeFirebaseAccount(context.Background(), 1, reauthtokentest.New(t, "reauth-token"), reauthtokentest.New(t, "new-account-token"))

		// Then
		assert.NoError(t, err)
		assert.Contains(t, logBuffer.String(), "failed to delete previous firebase user after firebase account change")
	})

	t.Run("変更先が現在と同じアカウントなら更新しない", func(t *testing.T) {
		// Given
		repo := new(MockUserRepository)
		verifier := new(mockRecentSignInVerifier)
		deleter := &recordingFirebaseUserDeleter{}
		verifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: oldUID, AuthTime: recentAuthTime}, nil).Once()
		verifier.On("VerifyRecentSignIn", mock.Anything, "new-account-token").Return(&RecentSignInInfo{UID: oldUID, AuthTime: recentAuthTime}, nil).Once()
		repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(newUser(), nil).Once()
		uc := newUsecase(repo, verifier, deleter)

		// When
		err := uc.ChangeFirebaseAccount(context.Background(), 1, reauthtokentest.New(t, "reauth-token"), reauthtokentest.New(t, "new-account-token"))

		// Then
		assert.ErrorIs(t, err, ErrFirebaseAccountUnchanged)
		repo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		assert.Empty(t, deleter.deletedUIDs)
	})

	t.Run("変更先が他ユーザーに連携済みなら競合エラーを返し旧Firebaseユーザーを削除しない", func(t *testing.T) {
		// Given
		repo := new(MockUserRepository)
		verifier := new(mockRecentSignInVerifier)
		deleter := &recordingFirebaseUserDeleter{}
		verifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: oldUID, AuthTime: recentAuthTime}, nil).Once()
		verifier.On("VerifyRecentSignIn", mock.Anything, "new-account-token").Return(&RecentSignInInfo{UID: newUID, AuthTime: recentAuthTime}, nil).Once()
		repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(newUser(), nil).Once()
		repo.On("LinkFirebaseUID", mock.Anything, mock.Anything, 1, ptrString(oldUID), newUID, mock.AnythingOfType("time.Time")).Return(repository.ErrFirebaseUIDAlreadyLinked).Once()
		uc := newUsecase(repo, verifier, deleter)

		// When
		err := uc.ChangeFirebaseAccount(context.Background(), 1, reauthtokentest.New(t, "reauth-token"), reauthtokentest.New(t, "new-account-token"))

		// Then
		assert.ErrorIs(t, err, ErrFirebaseUIDAlreadyLinked)
		assert.Empty(t, deleter.deletedUIDs)
	})

	t.Run("現在アカウントの再認証UIDが一致しなければ認証失敗として記録する", func(t *testing.T) {
		// Given
		repo := new(MockUserRepository)
		verifier := new(mockRecentSignInVerifier)
		deleter := &recordingFirebaseUserDeleter{}
		verifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: "other-uid", AuthTime: recentAuthTime}, nil).Once()
		verifier.On("VerifyRecentSignIn", mock.Anything, "new-account-token").Return(&RecentSignInInfo{UID: newUID, AuthTime: recentAuthTime}, nil).Once()
		repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(newUser(), nil).Once()
		uc := newUsecase(repo, verifier, deleter)
		logBuffer := captureDefaultSlog(t)

		// When
		err := uc.ChangeFirebaseAccount(context.Background(), 1, reauthtokentest.New(t, "reauth-token"), reauthtokentest.New(t, "new-account-token"))

		// Then
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Contains(t, logBuffer.String(), "suspicious change_firebase_account authentication failure")
		assert.Contains(t, logBuffer.String(), "change_firebase_account_reauth_uid_mismatch")
		repo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		assert.Empty(t, deleter.deletedUIDs)
	})

	t.Run("変更先アカウントのrecent sign-inが期限切れならユーザーを取得しない", func(t *testing.T) {
		// Given
		repo := new(MockUserRepository)
		verifier := new(mockRecentSignInVerifier)
		deleter := &recordingFirebaseUserDeleter{}
		verifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: oldUID, AuthTime: recentAuthTime}, nil).Once()
		verifier.On("VerifyRecentSignIn", mock.Anything, "new-account-token").Return(&RecentSignInInfo{UID: newUID, AuthTime: currentTime.Add(-time.Hour)}, nil).Once()
		uc := newUsecase(repo, verifier, deleter)

		// When
		err := uc.ChangeFirebaseAccount(context.Background(), 1, reauthtokentest.New(t, "reauth-token"), reauthtokentest.New(t, "new-account-token"))

		// Then
		assert.ErrorIs(t, err, ErrRecentSignInRequired)
		repo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
	})
}
