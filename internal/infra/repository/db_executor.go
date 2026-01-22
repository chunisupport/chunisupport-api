package repository

import (
	domainRepo "github.com/Qman110101/chunisupport-api/internal/domain/repository"
)

// DBorTx は後方互換性のためのエイリアスです。
// 新しいコードでは domainRepo.Executor を直接使用してください。
// DEPRECATED: Use domainRepo.Executor instead.
type DBorTx = domainRepo.Executor
