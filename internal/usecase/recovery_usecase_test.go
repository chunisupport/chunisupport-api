package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRecoveryUsecase_IssueRecoveryCodes(t *testing.T) {
	t.Run("IssueRecoveryCodes_正常系_リカバリーコード再発行が成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockRecoveryRepo := new(MockRecoveryCodeRepository)
		tm := &mockTransactionManager{}
		authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
		mockRecoveryRepo.On("DeleteByUserID", mock.Anything, mock.Anything, 1).Return(nil).Once()
		mockRecoveryRepo.On("CreateBatch", mock.Anything, mock.Anything, mock.MatchedBy(func(codes []*entity.RecoveryCode) bool {
			return len(codes) == info.RecoveryCodeCount
		})).Return(nil).Once()

		codes, err := authService.IssueRecoveryCodes(context.Background(), 1)
		assert.NoError(t, err)
		assert.Len(t, codes, info.RecoveryCodeCount)
		for _, code := range codes {
			segments := strings.Split(code, "-")
			assert.Len(t, segments, info.RecoveryCodeSegmentCount)
			for _, segment := range segments {
				assert.Len(t, segment, info.RecoveryCodeSegmentLength)
			}
		}
		mockUserRepo.AssertExpectations(t)
		mockRecoveryRepo.AssertExpectations(t)
	})

	t.Run("IssueRecoveryCodes_異常系_ユーザーが存在しない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockRecoveryRepo := new(MockRecoveryCodeRepository)
		tm := &mockTransactionManager{}
		authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(nil, sql.ErrNoRows).Once()

		_, err := authService.IssueRecoveryCodes(context.Background(), 1)
		assert.ErrorIs(t, err, ErrUserNotFound)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("IssueRecoveryCodes_異常系_リカバリーコード削除に失敗する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockRecoveryRepo := new(MockRecoveryCodeRepository)
		tm := &mockTransactionManager{}
		authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
		mockRecoveryRepo.On("DeleteByUserID", mock.Anything, mock.Anything, 1).Return(errors.New("delete error")).Once()

		_, err := authService.IssueRecoveryCodes(context.Background(), 1)
		assert.Error(t, err)
		mockUserRepo.AssertExpectations(t)
		mockRecoveryRepo.AssertExpectations(t)
	})
}

func TestRecoveryUsecase_RecoverWithRecoveryCode(t *testing.T) {
	pepper := "test-pepper"
	errDB := errors.New("db error")
	recoveryCode := "ABCD-EFGH-IJKL"
	normalized := normalizeRecoveryCode(recoveryCode)
	hash := hashRecoveryCode(normalized)
	un, _ := username.NewUserName("testuser")
	oldHash, _ := utils.HashPasswordWithPepper("old-password", pepper)
	ph, _ := passwordhash.NewPasswordHash(oldHash)
	samePasswordHash, _ := utils.HashPasswordWithPepper("same-password", pepper)
	samePasswordPH, _ := passwordhash.NewPasswordHash(samePasswordHash)

	newTestCode := func() *entity.RecoveryCode { return &entity.RecoveryCode{UserID: 1, CodeHash: hash} }
	newActiveUser := func() *entity.User { return &entity.User{ID: 1, Username: un, PasswordHash: ph} }
	newDeletedUser := func() *entity.User { return &entity.User{ID: 1, Username: un, PasswordHash: ph, IsDeleted: true} }
	newSamePasswordUser := func() *entity.User { return &entity.User{ID: 1, Username: un, PasswordHash: samePasswordPH} }

	tests := []struct {
		name        string
		newPassword string
		setupMock   func(*MockUserRepository, *MockRecoveryCodeRepository)
		wantErr     error
	}{
		{
			name:        "RecoverWithRecoveryCode_正常系_リカバリーコードで復旧できる",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newActiveUser(), nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				recoveryRepo.On("DeleteByID", mock.Anything, mock.Anything, mock.AnythingOfType("uint32")).Return(nil).Once()
			},
		},
		{
			name:        "RecoverWithRecoveryCode_異常系_リカバリーコードが見つからない",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(nil, sql.ErrNoRows).Once()
			},
			wantErr: ErrInvalidRecoveryCredentials,
		},
		{
			name:        "RecoverWithRecoveryCode_異常系_ユーザーが見つからない",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(nil, sql.ErrNoRows).Once()
			},
			wantErr: ErrInvalidRecoveryCredentials,
		},
		{
			name:        "RecoverWithRecoveryCode_異常系_非アクティブユーザー",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newDeletedUser(), nil).Once()
			},
			wantErr: ErrInvalidRecoveryCredentials,
		},
		{
			name:        "RecoverWithRecoveryCode_異常系_新しいパスワードが現在と同じ",
			newPassword: "same-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newSamePasswordUser(), nil).Once()
			},
			wantErr: ErrInvalidPassword,
		},
		{
			name:        "RecoverWithRecoveryCode_異常系_ユーザー更新に失敗する",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newActiveUser(), nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errDB).Once()
			},
			wantErr: errDB,
		},
		{
			name:        "RecoverWithRecoveryCode_異常系_リカバリーコード削除に失敗する",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newActiveUser(), nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				recoveryRepo.On("DeleteByID", mock.Anything, mock.Anything, mock.AnythingOfType("uint32")).Return(errDB).Once()
			},
			wantErr: errDB,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockSessionRepo := new(MockSessionRepository)
			mockRecoveryRepo := new(MockRecoveryCodeRepository)
			tm := &mockTransactionManager{}
			authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, pepper, newMockMasterCache())

			tc.setupMock(mockUserRepo, mockRecoveryRepo)
			err := authService.RecoverWithRecoveryCode(context.Background(), recoveryCode, tc.newPassword)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			mockUserRepo.AssertExpectations(t)
			mockRecoveryRepo.AssertExpectations(t)
		})
	}
}
