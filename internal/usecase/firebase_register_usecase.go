package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// FirebaseRegisterUsecase は Firebase ID トークンを使った新規ユーザー登録を扱います。
type FirebaseRegisterUsecase interface {
	// RegisterWithFirebase は Firebase ID トークンと希望ユーザー名で新規ユーザーを登録し、
	// 自動ログイン後の JWT トークンを返します。
	RegisterWithFirebase(ctx context.Context, idToken string, usernameStr string) (string, error)
}

type firebaseRegisterUsecase struct {
	tm            TransactionManager
	userRepo      repository.UserRepository
	tokenVerifier TokenVerifier
	sessionIssuer SessionIssuer
}

// NewFirebaseRegisterUsecase は Firebase 登録ユースケースを生成します。
// 依存関係が nil の場合はプログラムの設定ミスであるため、起動時にパニックします。
func NewFirebaseRegisterUsecase(tm TransactionManager, userRepo repository.UserRepository, tokenVerifier TokenVerifier, sessionIssuer SessionIssuer) FirebaseRegisterUsecase {
	if tm == nil {
		panic("firebaseRegisterUsecase: TransactionManager is nil")
	}
	if userRepo == nil {
		panic("firebaseRegisterUsecase: UserRepository is nil")
	}
	if tokenVerifier == nil {
		panic("firebaseRegisterUsecase: TokenVerifier is nil")
	}
	if sessionIssuer == nil {
		panic("firebaseRegisterUsecase: SessionIssuer is nil")
	}
	return &firebaseRegisterUsecase{
		tm:            tm,
		userRepo:      userRepo,
		tokenVerifier: tokenVerifier,
		sessionIssuer: sessionIssuer,
	}
}

func (u *firebaseRegisterUsecase) RegisterWithFirebase(ctx context.Context, idToken string, usernameStr string) (string, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return "", ErrInvalidIDToken
	}

	uid, err := u.tokenVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", err
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", errors.Join(ErrInternalError, errors.New("firebase uid is empty"))
	}

	un, err := username.NewUserName(usernameStr)
	if err != nil {
		return "", convertUsernameError(err)
	}

	var newUser *entity.User
	if err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
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
		return "", err
	}

	// NOTE: セッション発行はトランザクション外で実行している。
	// 理論上はユーザー作成が確定した後にセッション発行が失敗すると、
	// ユーザーだけが存在する中途半端な状態になりうる。
	// ただし、今後 Firebase に認証を一任し DB セッションを廃止する予定のため、
	// トランザクションへの組み込みは行わず、この設計は移行時に解消する。
	token, err := u.sessionIssuer.IssueSession(ctx, newUser)
	if err != nil {
		slog.Error("Firebase登録後のセッション発行に失敗しました", "user_id", newUser.ID, "error", err)
		return "", err
	}

	return token, nil
}

var _ FirebaseRegisterUsecase = (*firebaseRegisterUsecase)(nil)
