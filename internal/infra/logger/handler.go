package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Handler は標準出力とファイルの両方に出力するログハンドラーです。
// 標準ライブラリの slog.TextHandler を内部で使用します。
type Handler struct {
	handler slog.Handler
	file    *os.File
}

// ParseLogLevel は文字列をslog.Levelに変換します。
// 不正な値の場合は slog.LevelInfo を返します。
func ParseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// NewHandler は標準出力のみのHandlerを作成します。
func NewHandler(level slog.Level) *Handler {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	return &Handler{
		handler: slog.NewTextHandler(os.Stdout, opts),
		file:    nil,
	}
}

// NewHandlerWithFile はファイル出力も行うHandlerを作成します。
// 標準出力とファイルの両方にテキスト形式で出力します。
func NewHandlerWithFile(logDir string, level slog.Level) (*Handler, error) {
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(logDir, fmt.Sprintf("%s.log", timestamp))

	// #nosec G304 -- logDir comes from trusted configuration
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// 標準出力とファイルの両方に書き込む
	multiWriter := io.MultiWriter(os.Stdout, file)

	return &Handler{
		handler: slog.NewTextHandler(multiWriter, opts),
		file:    file,
	}, nil
}

// Close はログファイルをクローズします。
func (h *Handler) Close() error {
	if h.file != nil {
		return h.file.Close()
	}
	return nil
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
		file:    h.file,
	}
}

// WithGroup は slog.Handler インターフェースを実装します。
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		handler: h.handler.WithGroup(name),
		file:    h.file,
	}
}
