package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment"
)

var (
	ErrCommentTooLong     = errors.New("profile comment is too long")
	ErrCommentInvalidChar = errors.New("profile comment contains invalid characters")
)

// UserProfileCommentUsecase は独立したプロフィールコメントの取得と更新を扱います。
type UserProfileCommentUsecase interface {
	// Get は閲覧可能なユーザーのコメントを返します。未設定はnilです。
	Get(ctx context.Context, username string, requester *entity.User) (*string, error)
	// Update は空白除去後に空なら削除し、それ以外は検証して保存します。
	Update(ctx context.Context, userID int, comment string) (*string, error)
}

type userProfileCommentUsecase struct {
	db             repository.Executor
	repo           repository.UserProfileRepository
	userRepo       repository.UserRepository
	friendshipRepo repository.FriendshipRepository
}

// NewUserProfileCommentUsecase はプロフィールと共通の閲覧判定を使うユースケースを生成します。
func NewUserProfileCommentUsecase(db repository.Executor, repo repository.UserProfileRepository, userRepo repository.UserRepository, friendshipRepo repository.FriendshipRepository) UserProfileCommentUsecase {
	return &userProfileCommentUsecase{db: db, repo: repo, userRepo: userRepo, friendshipRepo: friendshipRepo}
}

func (u *userProfileCommentUsecase) Get(ctx context.Context, username string, requester *entity.User) (*string, error) {
	user, err := getAccessibleUser(ctx, u.db, u.userRepo, u.friendshipRepo, username, requester)
	if err != nil {
		return nil, err
	}
	profile, err := u.repo.FindByUserID(ctx, u.db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: find user profile: %w", ErrInternalError, err)
	}
	if profile == nil {
		return nil, nil
	}
	value := profile.Comment().String()
	return &value, nil
}

func (u *userProfileCommentUsecase) Update(ctx context.Context, userID int, raw string) (*string, error) {
	if strings.TrimSpace(raw) == "" {
		if err := u.repo.DeleteByUserID(ctx, u.db, userID); err != nil {
			return nil, fmt.Errorf("%w: delete user profile: %w", ErrInternalError, err)
		}
		return nil, nil
	}
	comment, err := profilecomment.NewProfileComment(raw)
	if err != nil {
		switch {
		case errors.Is(err, profilecomment.ErrTooLong):
			return nil, fmt.Errorf("%w: %w", ErrCommentTooLong, err)
		default:
			return nil, fmt.Errorf("%w: %w", ErrCommentInvalidChar, err)
		}
	}
	profile, err := entity.NewUserProfile(userID, comment)
	if err != nil {
		return nil, fmt.Errorf("%w: create user profile: %w", ErrInternalError, err)
	}
	if err := u.repo.Save(ctx, u.db, profile); err != nil {
		return nil, fmt.Errorf("%w: save user profile: %w", ErrInternalError, err)
	}
	value := comment.String()
	return &value, nil
}
