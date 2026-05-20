package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubReadOptimizedCapableVerifier struct {
	verifyIDTokenCalled                       bool
	verifyIDTokenWithoutRevocationCheckCalled bool
	wantIDToken                               string
	verifyIDTokenWithoutRevocationCheckErr    error
}

func (s *stubReadOptimizedCapableVerifier) VerifyIDToken(_ context.Context, idToken string) (string, error) {
	s.verifyIDTokenCalled = true
	s.wantIDToken = idToken
	return "strict-uid", nil
}

func (s *stubReadOptimizedCapableVerifier) VerifyIDTokenWithoutRevocationCheck(_ context.Context, idToken string) (string, error) {
	s.verifyIDTokenWithoutRevocationCheckCalled = true
	s.wantIDToken = idToken
	if s.verifyIDTokenWithoutRevocationCheckErr != nil {
		return "", s.verifyIDTokenWithoutRevocationCheckErr
	}
	return "read-uid", nil
}

type stubOnlyTokenVerifier struct {
	called bool
}

func (s *stubOnlyTokenVerifier) VerifyIDToken(_ context.Context, _ string) (string, error) {
	s.called = true
	return "only-strict-uid", nil
}

func TestNewReadOptimizedTokenVerifier(t *testing.T) {
	t.Run("nil を渡すと nil を返す", func(t *testing.T) {
		got := NewReadOptimizedTokenVerifier(nil)
		assert.Nil(t, got)
	})

	t.Run("readOptimizedCapableVerifier を渡すとラッパー化され失効確認なし検証へ委譲される", func(t *testing.T) {
		strict := &stubReadOptimizedCapableVerifier{}

		got := NewReadOptimizedTokenVerifier(strict)
		wrapped, ok := got.(*ReadOptimizedTokenVerifier)
		require.True(t, ok)
		require.NotNil(t, wrapped)

		uid, err := got.VerifyIDToken(context.Background(), "token-1")
		require.NoError(t, err)
		assert.Equal(t, "read-uid", uid)
		assert.True(t, strict.verifyIDTokenWithoutRevocationCheckCalled)
		assert.False(t, strict.verifyIDTokenCalled)
		assert.Equal(t, "token-1", strict.wantIDToken)
	})

	t.Run("readOptimizedCapableVerifier を実装しない TokenVerifier はそのまま返す", func(t *testing.T) {
		strict := &stubOnlyTokenVerifier{}

		got := NewReadOptimizedTokenVerifier(strict)
		assert.Same(t, strict, got)

		uid, err := got.VerifyIDToken(context.Background(), "token-2")
		require.NoError(t, err)
		assert.Equal(t, "only-strict-uid", uid)
		assert.True(t, strict.called)
	})

	t.Run("ラッパー化後に失効確認なし検証が失敗したらエラーを返す", func(t *testing.T) {
		strict := &stubReadOptimizedCapableVerifier{verifyIDTokenWithoutRevocationCheckErr: errors.New("verify failed")}
		got := NewReadOptimizedTokenVerifier(strict)

		uid, err := got.VerifyIDToken(context.Background(), "token-3")
		require.Error(t, err)
		assert.ErrorContains(t, err, "verify failed")
		assert.Empty(t, uid)
	})
}
