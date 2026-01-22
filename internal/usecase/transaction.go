package usecase

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
)

// TransactionManager はトランザクションを管理するインターフェースです。
// Clean Architectureの原則に従い、Usecase層はインフラの実装詳細に依存しません。
type TransactionManager interface {
	// Transactional は、渡された関数をトランザクション内で実行します。
	// 関数内でエラーが発生した場合はトランザクションをロールバックし、そうでなければコミットします。
	Transactional(ctx context.Context, f func(tx repository.Executor) error) error
}
