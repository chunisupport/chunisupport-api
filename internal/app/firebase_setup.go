package app

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/firebaseauth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// SetupFirebaseAuthServices は Firebase 認証関連の依存関係をまとめて初期化します。
func SetupFirebaseAuthServices(ctx context.Context, cfg config.Config) (usecase.TokenVerifier, usecase.FirebaseUserDeleter, usecase.FirebaseProviderAccountFinder, error) {
	client, err := firebaseauth.NewAuthClientFromCredentialsFile(ctx, cfg.Firebase.CredentialsFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize firebase auth client: %w", err)
	}

	return firebaseauth.NewTokenVerifier(client), firebaseauth.NewFirebaseUserDeleter(client), firebaseauth.NewProviderAccountFinder(client), nil
}

// SetupFirebaseTokenVerifier は Firebase TokenVerifier を初期化します。
func SetupFirebaseTokenVerifier(ctx context.Context, cfg config.Config) (usecase.TokenVerifier, error) {
	verifier, _, _, err := SetupFirebaseAuthServices(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase token verifier: %w", err)
	}
	return verifier, nil
}

// SetupFirebaseUserDeleter は FirebaseUserDeleter を初期化します。
func SetupFirebaseUserDeleter(ctx context.Context, cfg config.Config) (usecase.FirebaseUserDeleter, error) {
	_, deleter, _, err := SetupFirebaseAuthServices(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase user deleter: %w", err)
	}
	return deleter, nil
}
