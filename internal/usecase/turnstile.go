package usecase

import (
	"context"
	"errors"
	"strings"
)

func verifyTurnstile(ctx context.Context, verifier TurnstileVerifier, token string, remoteIP string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalidTurnstileToken
	}
	if verifier == nil {
		return errors.Join(ErrInternalError, errors.New("turnstile verifier is nil"))
	}

	if err := verifier.VerifyTurnstile(ctx, token, strings.TrimSpace(remoteIP)); err != nil {
		if errors.Is(err, ErrInvalidTurnstileToken) || errors.Is(err, ErrInternalError) {
			return err
		}

		return errors.Join(ErrInternalError, err)
	}

	return nil
}
