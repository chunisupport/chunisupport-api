package firebaseauth

import (
	"context"
	"fmt"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauthsdk "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// NewTokenVerifierFromCredentialsFile はサービスアカウントJSONから TokenVerifier を生成します。
func NewTokenVerifierFromCredentialsFile(ctx context.Context, credentialsFile string) (usecase.TokenVerifier, error) {
	credentialsFile = strings.TrimSpace(credentialsFile)
	if credentialsFile == "" {
		return nil, fmt.Errorf("firebase credentials file is empty")
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, err
	}

	return NewTokenVerifierFromApp(ctx, app)
}

// NewAuthClientFromCredentialsFile はサービスアカウントJSONから auth.Client を生成します。
func NewAuthClientFromCredentialsFile(ctx context.Context, credentialsFile string) (*firebaseauthsdk.Client, error) {
	credentialsFile = strings.TrimSpace(credentialsFile)
	if credentialsFile == "" {
		return nil, fmt.Errorf("firebase credentials file is empty")
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, err
	}

	return app.Auth(ctx)
}

// NewFirebaseUserDeleterFromCredentialsFile はサービスアカウントJSONから FirebaseUserDeleter を生成します。
func NewFirebaseUserDeleterFromCredentialsFile(ctx context.Context, credentialsFile string) (usecase.FirebaseUserDeleter, error) {
	client, err := NewAuthClientFromCredentialsFile(ctx, credentialsFile)
	if err != nil {
		return nil, err
	}
	return NewFirebaseUserDeleter(client), nil
}
