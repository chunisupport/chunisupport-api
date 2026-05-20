package usecase

import "context"

type readOptimizedCapableVerifier interface {
	TokenVerifier
	VerifyIDTokenWithoutRevocationCheck(ctx context.Context, idToken string) (string, error)
}

// ReadOptimizedTokenVerifier は読み取り系向けに失効チェックなし検証を優先するラッパーです。
type ReadOptimizedTokenVerifier struct {
	strict readOptimizedCapableVerifier
}

// NewReadOptimizedTokenVerifier は read 系向け TokenVerifier を生成します。
func NewReadOptimizedTokenVerifier(strict TokenVerifier) TokenVerifier {
	if strict == nil {
		return nil
	}

	verifier, ok := strict.(readOptimizedCapableVerifier)
	if !ok {
		return strict
	}

	return &ReadOptimizedTokenVerifier{strict: verifier}
}

func (v *ReadOptimizedTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	return v.strict.VerifyIDTokenWithoutRevocationCheck(ctx, idToken)
}
