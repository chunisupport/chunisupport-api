package firebaseauth

import (
	"context"
	"fmt"
	"strings"

	firebase "firebase.google.com/go/v4"
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
