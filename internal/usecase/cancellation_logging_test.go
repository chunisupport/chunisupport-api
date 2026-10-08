package usecase

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCancellationLogging(t *testing.T) {
	ordinaryErr := errors.New("database unavailable")
	scenarios := []struct {
		name string
		run  func(error) error
	}{
		{
			name: "Firebase UID検索",
			run: func(repoErr error) error {
				ctx := context.Background()
				verifier := new(mockTokenVerifier)
				userRepo := new(MockUserRepository)
				verifier.On("VerifyIDToken", mock.Anything, "token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repoErr).Once()

				_, err := NewFirebaseAuthUsecase(nil, userRepo, verifier).Authenticate(ctx, "token")
				return err
			},
		},
		{
			name: "WORLD'S END一覧取得",
			run: func(repoErr error) error {
				ctx := context.Background()
				repo := new(MockWorldsendChartRepository)
				exec := new(MockExecutor)
				repo.On("FindAll", ctx, exec, false).Return(nil, repoErr).Once()

				_, err := NewWorldsendUsecase(repo, nil, exec).GetAllWorldsendSongs(ctx, false, nil)
				return err
			},
		},
		{
			name: "ユーザー検索",
			run: func(repoErr error) error {
				uc := NewUserUsecase(nil, &stubUserRepository{err: repoErr}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil, nil)
				_, err := uc.GetUserProfile(context.Background(), "tester", nil)
				return err
			},
		},
		{
			name: "プロフィールのプレイヤーレコード取得",
			run: func(repoErr error) error {
				userName, err := username.NewUserName("tester")
				if err != nil {
					return err
				}
				user := &entity.User{Username: userName, PlayerID: new(1)}
				player := &entity.Player{ID: 1}
				playerRepo := &stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{
					Player: player,
					Honors: []*entity.PlayerHonor{},
				}}
				uc := NewUserUsecase(nil, &stubUserRepository{user: user}, playerRepo, &stubPlayerRecordRepository{err: repoErr}, nil, nil, nil, nil, nil)

				_, err = uc.GetUserProfileWithRecords(context.Background(), "tester", nil)
				return err
			},
		},
	}
	failures := []struct {
		name      string
		err       error
		wantCause error
		wantLog   bool
	}{
		{
			name:      "context canceledは記録しない",
			err:       errors.Join(repository.ErrRepositoryOperationFailed, context.Canceled),
			wantCause: context.Canceled,
		},
		{
			name:      "通常エラーはERRORで記録する",
			err:       errors.Join(repository.ErrRepositoryOperationFailed, ordinaryErr),
			wantCause: ordinaryErr,
			wantLog:   true,
		},
		{
			name:      "deadline exceededはERRORで記録する",
			err:       errors.Join(repository.ErrRepositoryOperationFailed, context.DeadlineExceeded),
			wantCause: context.DeadlineExceeded,
			wantLog:   true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for _, failure := range failures {
				t.Run(failure.name, func(t *testing.T) {
					var logBuffer bytes.Buffer
					originalLogger := slog.Default()
					slog.SetDefault(slog.New(slog.NewTextHandler(&logBuffer, nil)))
					t.Cleanup(func() { slog.SetDefault(originalLogger) })

					err := scenario.run(failure.err)

					require.Error(t, err)
					assert.ErrorIs(t, err, repository.ErrRepositoryOperationFailed)
					assert.ErrorIs(t, err, failure.wantCause)
					if failure.wantLog {
						assert.Contains(t, logBuffer.String(), "level=ERROR")
					} else {
						assert.Empty(t, logBuffer.String())
					}
				})
			}
		})
	}
}
