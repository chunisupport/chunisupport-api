package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
)

type advisoryLockProvider struct {
	db *sqlx.DB
}

// NewAdvisoryLockProvider はMySQLの接続単位アドバイザリロックを生成します。
func NewAdvisoryLockProvider(database *sqlx.DB) repository.BatchLockProvider {
	return &advisoryLockProvider{db: database}
}

func (p *advisoryLockProvider) TryAcquire(ctx context.Context, name string) (repository.BatchLock, bool, error) {
	conn, err := p.db.Connx(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired sql.NullInt64
	if err := conn.GetContext(ctx, &acquired, `SELECT GET_LOCK(?, 0)`, name); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !acquired.Valid {
		_ = conn.Close()
		return nil, false, fmt.Errorf("GET_LOCK returned NULL")
	}
	if acquired.Int64 == 0 {
		_ = conn.Close()
		return nil, false, nil
	}
	return &mysqlAdvisoryLock{conn: conn, name: name}, true, nil
}

type mysqlAdvisoryLock struct {
	conn *sqlx.Conn
	name string
}

func (l *mysqlAdvisoryLock) Release(ctx context.Context) error {
	defer l.conn.Close()
	var released sql.NullInt64
	if err := l.conn.GetContext(ctx, &released, `SELECT RELEASE_LOCK(?)`, l.name); err != nil {
		return err
	}
	if !released.Valid || released.Int64 != 1 {
		return fmt.Errorf("RELEASE_LOCK failed")
	}
	return nil
}
