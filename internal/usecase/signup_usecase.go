package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// SignupUsecase はFirebase Bearerトークンを用いた初回ユーザー作成を扱います。
type SignupUsecase interface {
	// Signup はFirebase IDトークンとユーザー名でアプリ内ユーザーを作成します。
	Signup(ctx context.Context, idToken string, usernameStr string) (*dto_internal.UserDTO, error)
}

type signupUsecase struct {
	tm                  TransactionManager
	userRepo            repository.UserRepository
	tokenVerifier       TokenVerifier
	accountTypeProvider AccountTypeProvider
}

// NewSignupUsecase は signup 用ユースケースを生成します。
func NewSignupUsecase(
	tm TransactionManager,
	userRepo repository.UserRepository,
	tokenVerifier TokenVerifier,
	accountTypeProvider AccountTypeProvider,
) SignupUsecase {
	if tm == nil {
		panic("signupUsecase: TransactionManager is nil")
	}
	if userRepo == nil {
		panic("signupUsecase: UserRepository is nil")
	}
	if tokenVerifier == nil {
		panic("signupUsecase: TokenVerifier is nil")
	}
	if accountTypeProvider == nil {
		panic("signupUsecase: AccountTypeProvider is nil")
	}

	return &signupUsecase{
		tm:                  tm,
		userRepo:            userRepo,
		tokenVerifier:       tokenVerifier,
		accountTypeProvider: accountTypeProvider,
	}
}

func (u *signupUsecase) Signup(ctx context.Context, idToken string, usernameStr string) (*dto_internal.UserDTO, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, ErrInvalidIDToken
	}

	uid, err := u.tokenVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIDToken):
			return nil, err
		case errors.Is(err, ErrInternalError):
			return nil, err
		default:
			return nil, errors.Join(ErrInternalError, err)
		}
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, errors.Join(ErrInternalError, errors.New("firebase uid is empty"))
	}

	un, err := username.NewUserName(usernameStr)
	if err != nil {
		return nil, convertUsernameError(err)
	}

	var newUser *entity.User
	if err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if _, err := u.userRepo.FindByFirebaseUID(ctx, tx, uid); err == nil {
			return ErrFirebaseUIDAlreadyLinked
		} else if !errors.Is(err, repository.ErrUserNotFound) {
			return err
		}

		if _, err := u.userRepo.FindByUsername(ctx, tx, un.String()); err == nil {
			return ErrUsernameTaken
		} else if !errors.Is(err, repository.ErrUserNotFound) {
			return err
		}

		newUser = entity.NewFirebaseUser(un, uid, info.AccountTypePlayer)
		if err := u.userRepo.Save(ctx, tx, newUser); err != nil {
			if errors.Is(err, repository.ErrDuplicateUsername) {
				return ErrUsernameTaken
			}
			if errors.Is(err, repository.ErrFirebaseUIDAlreadyLinked) {
				return ErrFirebaseUIDAlreadyLinked
			}
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	accountTypeName := u.accountTypeProvider.GetAccountTypeNameByID(newUser.AccountTypeID)
	return dto_internal.ToUserDTO(newUser, accountTypeName, newUser.IsPrivate, nil), nil
}

var _ SignupUsecase = (*signupUsecase)(nil)
