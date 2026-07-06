package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// UserUpdatedAtQueryResult はユーザー更新日時取得に必要な情報だけを保持します。
type UserUpdatedAtQueryResult struct {
	User             *entity.User
	PlayerUpdatedAt  *time.Time
	RecordsUpdatedAt *time.Time
}

// UserUpdatedAtQueryService はユーザー更新日時表示用の読み取りを提供します。
type UserUpdatedAtQueryService interface {
	FindByUsername(ctx context.Context, exec Executor, username string) (*UserUpdatedAtQueryResult, error)
}
