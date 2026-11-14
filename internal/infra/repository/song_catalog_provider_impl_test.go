package repository

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
)

func TestSongCatalogProvider_Refresh_先行する再構築の待機中もctxの終了で戻る(t *testing.T) {
	// Given: 先行する再構築がDB応答待ちで止まっている状況を、直列化の枠を占有して再現します。
	provider := newSongCatalogProvider(nil, nil, nil)
	provider.refreshSem <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// When
	done := make(chan error, 1)
	go func() { done <- provider.Refresh(ctx) }()

	// Then
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.ErrorIs(t, err, domainrepo.ErrRepositoryOperationFailed)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "先行する再構築の待機が ctx の期限で終了しませんでした")
	}
}
