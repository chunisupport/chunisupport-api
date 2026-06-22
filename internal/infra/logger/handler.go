package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/config"
)

// Reopenable はlogrotate後に同じパスのログファイルを開き直せるWriterです。
type Reopenable interface {
	io.WriteCloser
	Reopen() error
}

// Handler は標準出力とファイルに出力できるログハンドラーです。
type Handler struct {
	handler slog.Handler
	writer  Reopenable
}

// ParseLogLevel は検証済みのログレベル文字列をslog.Levelへ変換します。
func ParseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// NewHandler はlogging設定に従ってアプリケーションログのHandlerを作成します。
func NewHandler(cfg config.Logging) (*Handler, error) {
	level := ParseLogLevel(cfg.Level)
	writer, err := newLogWriter(cfg.Stdout, cfg.AppFile)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{Level: level}
	return &Handler{
		handler: slog.NewTextHandler(writer, opts),
		writer:  writer,
	}, nil
}

// Reopen はファイル出力が有効な場合にログファイルを開き直します。
func (h *Handler) Reopen() error {
	if h.writer == nil {
		return nil
	}
	return h.writer.Reopen()
}

// Close はログ出力先を閉じます。
func (h *Handler) Close() error {
	if h.writer == nil {
		return nil
	}
	return h.writer.Close()
}

// Enabled は slog.Handler インターフェースを実装します。
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle は slog.Handler インターフェースを実装します。
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	return h.handler.Handle(ctx, r)
}

// WithAttrs は slog.Handler インターフェースを実装します。
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		handler: h.handler.WithAttrs(attrs),
		writer:  h.writer,
	}
}

// WithGroup は slog.Handler インターフェースを実装します。
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		handler: h.handler.WithGroup(name),
		writer:  h.writer,
	}
}

// NewAccessLogWriter はEchoアクセスログ用のWriterを作成します。
func NewAccessLogWriter(cfg config.Logging) (Reopenable, error) {
	return newLogWriter(cfg.Stdout, cfg.AccessFile)
}

func newLogWriter(stdout bool, filePath string) (Reopenable, error) {
	if filePath == "" {
		if stdout {
			return noopReopenableWriteCloser{Writer: os.Stdout}, nil
		}
		return nil, fmt.Errorf("log output is not configured")
	}

	fileWriter, err := NewReopenableFileWriter(filePath)
	if err != nil {
		return nil, err
	}
	if stdout {
		return &multiReopenableWriteCloser{
			writer: io.MultiWriter(os.Stdout, fileWriter),
			file:   fileWriter,
		}, nil
	}
	return fileWriter, nil
}

type noopReopenableWriteCloser struct {
	io.Writer
}

func (w noopReopenableWriteCloser) Close() error {
	return nil
}

func (w noopReopenableWriteCloser) Reopen() error {
	return nil
}

type multiReopenableWriteCloser struct {
	writer io.Writer
	file   Reopenable
}

func (w *multiReopenableWriteCloser) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}

func (w *multiReopenableWriteCloser) Close() error {
	return w.file.Close()
}

func (w *multiReopenableWriteCloser) Reopen() error {
	return w.file.Reopen()
}

func ensureLogDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", dir, err)
	}
	return nil
}
