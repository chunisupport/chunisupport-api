package firebaseauth

import (
	"context"

	firebaseauthsdk "firebase.google.com/go/v4/auth"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

type firebaseUserGetter interface {
	GetUser(ctx context.Context, uid string) (*firebaseauthsdk.UserRecord, error)
}

type providerAccountFinder struct {
	client firebaseUserGetter
}

// NewProviderAccountFinder は Firebase Admin SDK の auth.Client を使う FirebaseProviderAccountFinder を生成します。
// 連携情報はクライアントから送られた値ではなく Firebase に問い合わせて得るため、利用者が書き換えることはできません。
func NewProviderAccountFinder(client *firebaseauthsdk.Client) usecase.FirebaseProviderAccountFinder {
	return &providerAccountFinder{client: client}
}

func (f *providerAccountFinder) FindProviderUID(ctx context.Context, firebaseUID string, providerID string) (string, error) {
	record, err := f.client.GetUser(ctx, firebaseUID)
	if err != nil {
		// 退会処理と並行して届いた場合など、Firebase ユーザーがもういなければ連携もないものとして扱います。
		if firebaseauthsdk.IsUserNotFound(err) {
			return "", nil
		}
		return "", err
	}
	for _, provider := range record.ProviderUserInfo {
		if provider.ProviderID == providerID {
			return provider.UID, nil
		}
	}
	return "", nil
}
