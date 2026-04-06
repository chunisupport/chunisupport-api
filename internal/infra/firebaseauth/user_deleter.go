package firebaseauth

import (
	"context"
	"errors"
	"strings"

	firebaseauthsdk "firebase.google.com/go/v4/auth"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

type firebaseUserDeleter struct {
	client *firebaseauthsdk.Client
}

// NewFirebaseUserDeleter は Firebase Admin SDK の auth.Client を使う FirebaseUserDeleter を生成します。
func NewFirebaseUserDeleter(client *firebaseauthsdk.Client) usecase.FirebaseUserDeleter {
	return &firebaseUserDeleter{client: client}
}

func (d *firebaseUserDeleter) DeleteUser(ctx context.Context, uid string) error {
	if d.client == nil {
		return errors.New("firebase auth client is nil")
	}
	trimmedUID := strings.TrimSpace(uid)
	if trimmedUID == "" {
		return errors.New("firebase uid is empty")
	}

	return d.client.DeleteUser(ctx, trimmedUID)
}
