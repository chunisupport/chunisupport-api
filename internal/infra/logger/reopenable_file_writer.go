package logger

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// ReopenableFileWriter は固定パスのログファイルを安全に開き直せるWriterです。
type ReopenableFileWriter struct {
	path   string
	file   *os.File
	closed bool
	mu     sync.Mutex
}

// NewReopenableFileWriter は固定ファイルパスへ追記するWriterを作成します。
func NewReopenableFileWriter(path string) (*ReopenableFileWriter, error) {
	if err := ensureLogDir(path); err != nil {
		return nil, err
	}

	file, err := openLogFile(path)
	if err != nil {
		return nil, err
	}

	return &ReopenableFileWriter{
		path: path,
		file: file,
	}, nil
}

func openLogFile(path string) (*os.File, error) {
	// #nosec G304 -- ログパスは設定ファイルで管理される運用値です。
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", path, err)
	}
	return file, nil
}

func (w *ReopenableFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed || w.file == nil {
		return 0, fmt.Errorf("log file %s is closed: %w", w.path, os.ErrClosed)
	}
	return w.file.Write(p)
}

// Reopen は新しいファイルを開いてから差し替えます。
func (w *ReopenableFileWriter) Reopen() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("log file %s is closed: %w", w.path, os.ErrClosed)
	}
	w.mu.Unlock()

	if err := ensureLogDir(w.path); err != nil {
		return err
	}

	newFile, err := openLogFile(w.path)
	if err != nil {
		return err
	}

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		_ = newFile.Close()
		return fmt.Errorf("log file %s is closed: %w", w.path, os.ErrClosed)
	}
	oldFile := w.file
	w.file = newFile
	w.mu.Unlock()

	if oldFile != nil {
		if err := oldFile.Close(); err != nil {
			slog.Warn("Failed to close old log file after reopen", "path", w.path, "error", err)
		}
	}
	return nil
}

func (w *ReopenableFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.closed = true
	return err
}
