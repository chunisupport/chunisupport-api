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
	// ErrCommentInappropriate は内容の判定で不適切とされたコメントを保存しなかった場合に返します。
	ErrCommentInappropriate = errors.New("profile comment is inappropriate")
	// ErrCommentModerationUnavailable は内容を判定できず、コメントを保存しなかった場合に返します。
	ErrCommentModerationUnavailable = errors.New("profile comment moderation unavailable")
)

// CommentModerator はプロフィールコメントの内容判定を抽象化します。
type CommentModerator interface {
	// IsFlagged はコメントが不適切と判定された場合に true を返します。
	// 判定結果を得られなかった場合はエラーを返します。
	IsFlagged(ctx context.Context, comment string) (bool, error)
}

// UserProfileCommentUsecase は独立したプロフィールコメントの取得と更新を扱います。
type UserProfileCommentUsecase interface {
	// Get は閲覧可能なユーザーのコメントを返します。未設定はnilです。
	Get(ctx context.Context, username string, requester *entity.User) (*string, error)
	// Update は空白除去後に空なら削除し、それ以外は検証と内容判定を通ったものだけ保存します。
	// 不適切と判定された場合は ErrCommentInappropriate、判定できなかった場合は ErrCommentModerationUnavailable を返します。
	Update(ctx context.Context, userID int, comment string) (*string, error)
}

type userProfileCommentUsecase struct {
	db             repository.Executor
	repo           repository.UserProfileRepository
	userRepo       repository.UserRepository
	friendshipRepo repository.FriendshipRepository
	moderator      CommentModerator
}

// NewUserProfileCommentUsecase はプロフィールと共通の閲覧判定を使うユースケースを生成します。
func NewUserProfileCommentUsecase(db repository.Executor, repo repository.UserProfileRepository, userRepo repository.UserRepository, friendshipRepo repository.FriendshipRepository, moderator CommentModerator) UserProfileCommentUsecase {
	return &userProfileCommentUsecase{db: db, repo: repo, userRepo: userRepo, friendshipRepo: friendshipRepo, moderator: moderator}
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
	// 判定できないコメントを公開しないため、判定の失敗時も保存せずに止めます。
	flagged, err := u.moderator.IsFlagged(ctx, comment.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCommentModerationUnavailable, err)
	}
	if flagged {
		return nil, ErrCommentInappropriate
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
