package app

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/firebaseauth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// SetupFirebaseTokenVerifier は Firebase TokenVerifier を初期化します。
func SetupFirebaseTokenVerifier(ctx context.Context, cfg config.Config) (usecase.TokenVerifier, error) {
	verifier, err := firebaseauth.NewTokenVerifierFromCredentialsFile(ctx, cfg.Firebase.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase token verifier: %w", err)
	}
	if verifier == nil {
		return nil, fmt.Errorf("failed to initialize firebase token verifier: verifier is nil")
	}

	return verifier, nil
}
