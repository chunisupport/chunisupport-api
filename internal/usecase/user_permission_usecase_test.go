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

func TestUserPermissionUsecase_ChangePermission(t *testing.T) {
	tests := []struct {
		name       string
		requester  *entity.User
		permission string
		setup      func(*MockUserRepository)
		wantErr    error
	}{
		{
			name:       "ADMINは他者をEDITORへ変更できる",
			requester:  &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			permission: "EDITOR",
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 2}, nil).Once()
				repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(&entity.User{ID: 2, AccountTypeID: info.AccountTypePlayer}, nil).Once()
				repo.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
					return user.ID == 2 && user.AccountTypeID == info.AccountTypeEditor
				})).Return(nil).Once()
			},
		},
		{
			name:       "ADMINは他者をADMINへ昇格できる",
			requester:  &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			permission: "ADMIN",
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 2}, nil).Once()
				repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(&entity.User{ID: 2, AccountTypeID: info.AccountTypePlayer}, nil).Once()
				repo.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
					return user.ID == 2 && user.AccountTypeID == info.AccountTypeAdmin
				})).Return(nil).Once()
			},
		},
		{
			name:       "ADMINは自分をEDITORへ降格できない",
			requester:  &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			permission: "EDITOR",
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 1}, nil).Once()
				repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(&entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin}, nil).Once()
			},
			wantErr: entity.ErrCannotDemoteOwnAdmin,
		},
		{
			name:       "ADMINは自分をADMINのままにできる",
			requester:  &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			permission: "ADMIN",
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 1}, nil).Once()
				repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(&entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin}, nil).Once()

			},
		},
		{
			name:       "ADMIN以外は変更できない",
			requester:  &entity.User{ID: 1, AccountTypeID: info.AccountTypeEditor},
			permission: "EDITOR",
			setup:      func(*MockUserRepository) {},
			wantErr:    ErrAdminRequired,
		},
		{
			name:       "未知の権限は変更できない",
			requester:  &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			permission: "INVALID",
			setup:      func(*MockUserRepository) {},
			wantErr:    entity.ErrInvalidAccountType,
		},
		{
			name:       "対象ユーザーが存在しない場合はエラー",
			requester:  &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin},
			permission: "PLAYER",
			setup: func(repo *MockUserRepository) {
				repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(nil, repository.ErrUserNotFound).Once()
			},
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := new(MockUserRepository)
			tt.setup(repo)
			usecase := NewUserPermissionUsecase(&MockExecutor{}, &mockTransactionManager{}, repo)

			// When
			err := usecase.ChangePermission(context.Background(), tt.requester, "targetuser", tt.permission)

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

func TestUserPermissionUsecase_ChangePermission_保存失敗を返す(t *testing.T) {
	// Given
	repo := new(MockUserRepository)
	saveErr := errors.New("save failed")
	repo.On("FindByUsername", mock.Anything, mock.Anything, "targetuser").Return(&entity.User{ID: 2}, nil).Once()
	repo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(&entity.User{ID: 2, AccountTypeID: info.AccountTypePlayer}, nil).Once()
	repo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(saveErr).Once()
	usecase := NewUserPermissionUsecase(&MockExecutor{}, &mockTransactionManager{}, repo)

	// When
	err := usecase.ChangePermission(context.Background(), &entity.User{ID: 1, AccountTypeID: info.AccountTypeAdmin}, "targetuser", "EDITOR")

	// Then
	assert.ErrorIs(t, err, saveErr)
	repo.AssertExpectations(t)
}
