package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	// ErrXAccountNotLinked はログイン中の Firebase ユーザーに X が連携されていない場合に返します。
	ErrXAccountNotLinked = errors.New("x account is not linked")
	// ErrInvalidXCredentials は X のアクセストークンが拒否された場合や、連携中とは別の X アカウントのトークンだった場合に返します。
	ErrInvalidXCredentials = errors.New("invalid x credentials")
	// ErrXAPIUnavailable は X API の障害や回数制限で本人情報を取得できなかった場合に返します。
	ErrXAPIUnavailable = errors.New("x api unavailable")
)

// FirebaseProviderAccountFinder は Firebase ユーザーに連携された認証プロバイダ側のユーザーIDを取得します。
type FirebaseProviderAccountFinder interface {
	// FindProviderUID は指定したプロバイダのユーザーIDを返します。連携されていない場合は空文字を返します。
	FindProviderUID(ctx context.Context, firebaseUID string, providerID string) (string, error)
}

// XAuthenticatedUser は X の認証済みユーザー情報です。
type XAuthenticatedUser struct {
	ID       string
	Username string
}

// XUserClient は利用者本人の X アクセストークンで X API を呼び出します。
type XUserClient interface {
	// GetAuthenticatedUser はトークンの持ち主の X アカウントを返します。
	// トークンが拒否された場合は ErrInvalidXCredentials、それ以外の取得失敗は ErrXAPIUnavailable を返します。
	GetAuthenticatedUser(ctx context.Context, accessToken string, accessTokenSecret string) (XAuthenticatedUser, error)
}

// XAccountSyncInput は X ログイン直後に得た OAuth 1.0a のアクセストークンです。
type XAccountSyncInput struct {
	AccessToken       string
	AccessTokenSecret string
}

// XAccountOutput は API に返す X アカウント情報です。
type XAccountOutput struct {
	ID       string
	Username string
}

// XAccountUsecase は連携中の X アカウントのユーザー名を扱います。
type XAccountUsecase interface {
	// Get は保存済みの X アカウント情報を返します。未保存の場合は nil を返します。
	Get(ctx context.Context, userID int) (*XAccountOutput, error)
	// Sync は X から本人のユーザー名を取得し、連携中のアカウントと一致した場合だけ保存します。
	Sync(ctx context.Context, user *entity.User, input XAccountSyncInput) (*XAccountOutput, error)
}

type xAccountUsecase struct {
	db             repository.Executor
	repo           repository.XAccountRepository
	providerFinder FirebaseProviderAccountFinder
	xUserClient    XUserClient
}

// NewXAccountUsecase は XAccountUsecase を生成します。
func NewXAccountUsecase(db repository.Executor, repo repository.XAccountRepository, providerFinder FirebaseProviderAccountFinder, xUserClient XUserClient) XAccountUsecase {
	return &xAccountUsecase{
		db:             db,
		repo:           repo,
		providerFinder: providerFinder,
		xUserClient:    xUserClient,
	}
}

func (u *xAccountUsecase) Get(ctx context.Context, userID int) (*XAccountOutput, error) {
	account, err := u.repo.FindByUserID(ctx, u.db, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: find x account: %w", ErrInternalError, err)
	}
	if account == nil {
		return nil, nil
	}
	return toXAccountOutput(account), nil
}

// Sync はクライアントから受け取ったユーザー名を信用せず、X API が返した本人情報だけを保存します。
// さらに X の数値IDを Firebase に連携済みのものと照合し、別の X アカウントのトークンで上書きされることを防ぎます。
func (u *xAccountUsecase) Sync(ctx context.Context, user *entity.User, input XAccountSyncInput) (*XAccountOutput, error) {
	accessToken := strings.TrimSpace(input.AccessToken)
	accessTokenSecret := strings.TrimSpace(input.AccessTokenSecret)
	if accessToken == "" || accessTokenSecret == "" {
		return nil, ErrInvalidXCredentials
	}
	if !user.HasLinkedFirebase() {
		return nil, ErrXAccountNotLinked
	}

	linkedXUserID, err := u.providerFinder.FindProviderUID(ctx, *user.FirebaseUID, info.FirebaseProviderIDX)
	if err != nil {
		return nil, fmt.Errorf("%w: find firebase provider: %w", ErrInternalError, err)
	}
	if linkedXUserID == "" {
		return nil, ErrXAccountNotLinked
	}

	xUser, err := u.xUserClient.GetAuthenticatedUser(ctx, accessToken, accessTokenSecret)
	if err != nil {
		if errors.Is(err, ErrInvalidXCredentials) || errors.Is(err, ErrXAPIUnavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", ErrXAPIUnavailable, err)
	}
	if xUser.ID != linkedXUserID {
		return nil, ErrInvalidXCredentials
	}

	account, err := entity.NewXAccount(user.ID, xUser.ID, xUser.Username)
	if err != nil {
		return nil, fmt.Errorf("%w: unexpected x user: %w", ErrXAPIUnavailable, err)
	}
	if err := u.repo.Save(ctx, u.db, account); err != nil {
		return nil, fmt.Errorf("%w: save x account: %w", ErrInternalError, err)
	}
	return toXAccountOutput(account), nil
}

func toXAccountOutput(account *entity.XAccount) *XAccountOutput {
	return &XAccountOutput{
		ID:       account.XUserID(),
		Username: account.Username(),
	}
}
