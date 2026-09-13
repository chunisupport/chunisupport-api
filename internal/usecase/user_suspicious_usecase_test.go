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

func TestUserSuspiciousUsecase_ChangeSuspicious(t *testing.T) {
	tests := []struct {
		name      string
		requester *entity.User
		requested bool
		setup     func(*MockUserRepository)
		wantErr   error
	}{
		{
			name:      "ADMINは不審フラグを有効化できる",
			requester: &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			requested: true,
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 2}, nil).Once()
				repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(&entity.User{ID: 2, IsSuspicious: false}, nil).Once()
				repo.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
					return user.ID == 2 && user.IsSuspicious
				})).Return(nil).Once()
			},
		},
		{
			name:      "ADMINは不審フラグを無効化できる",
			requester: &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			requested: false,
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 2}, nil).Once()
				repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(&entity.User{ID: 2, IsSuspicious: true}, nil).Once()
				repo.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
					return user.ID == 2 && !user.IsSuspicious
				})).Return(nil).Once()
			},
		},
		{
			name:      "ADMIN以外は変更できない",
			requester: &entity.User{ID: 1, AccountTypeID: info.AccountTypeEditor},
			requested: true,
			setup:     func(*MockUserRepository) {},
			wantErr:   ErrAdminRequired,
		},
		{
			name:      "対象ユーザーが存在しない場合はエラー",
			requester: &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			requested: true,
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(nil, repository.ErrUserNotFound).Once()
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:      "同じ値なら保存を省略する",
			requester: &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			requested: true,
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 2}, nil).Once()
				repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(&entity.User{ID: 2, IsSuspicious: true}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := new(MockUserRepository)
			tt.setup(repo)
			uc := NewUserSuspiciousUsecase(&MockExecutor{}, &mockTransactionManager{}, repo)

			// When
			err := uc.ChangeSuspicious(context.Background(), tt.requester, "targetuser", tt.requested)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserSuspiciousUsecase_ChangeSuspicious_保存失敗を返す(t *testing.T) {
	// Given
	repo := new(MockUserRepository)
	saveErr := errors.New("save failed")
	repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 2}, nil).Once()
	repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(&entity.User{ID: 2}, nil).Once()
	repo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(saveErr).Once()
	uc := NewUserSuspiciousUsecase(&MockExecutor{}, &mockTransactionManager{}, repo)

	// When
	err := uc.ChangeSuspicious(context.Background(), &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin}, "targetuser", true)

	// Then
	assert.ErrorIs(t, err, saveErr)
	repo.AssertExpectations(t)
}
