package app

import (
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
)

type managedLogHandler interface {
	Reopen() error
	Close() error
}

// LogManager はアプリログとアクセスログのライフサイクルをまとめて管理します。
type LogManager struct {
	AppHandler   managedLogHandler
	AccessWriter logger.Reopenable
}

func (m *LogManager) ReopenAll() error {
	var errs []error
	if m.AppHandler != nil {
		if err := m.AppHandler.Reopen(); err != nil {
			errs = append(errs, fmt.Errorf("reopen app log: %w", err))
		}
	}
	if m.AccessWriter != nil {
		if err := m.AccessWriter.Reopen(); err != nil {
			errs = append(errs, fmt.Errorf("reopen access log: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (m *LogManager) Close() error {
	var errs []error
	if m.AppHandler != nil {
		if err := m.AppHandler.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close app log: %w", err))
		}
	}
	if m.AccessWriter != nil {
		if err := m.AccessWriter.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close access log: %w", err))
		}
	}
	return errors.Join(errs...)
}
