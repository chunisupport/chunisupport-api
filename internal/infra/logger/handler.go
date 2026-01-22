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

// CustomHandler はカスタムログハンドラーです。
// 標準出力には色付きで、ファイルには色なしでログを出力します。
type CustomHandler struct {
	stdoutHandler slog.Handler
	fileHandler   slog.Handler
	file          *os.File // クローズ用にファイルハンドルを保持
}

// NewCustomHandler は標準出力のみのCustomHandlerを作成します。
func NewCustomHandler() *CustomHandler {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	return &CustomHandler{
		stdoutHandler: slog.NewTextHandler(os.Stdout, opts),
		fileHandler:   nil,
		file:          nil,
	}
}

// NewCustomHandlerWithFile はファイル出力も行うCustomHandlerを作成します。
// 標準出力には色付きで、ファイルには色なしのテキスト形式で出力します。
func NewCustomHandlerWithFile(logDir string) (*CustomHandler, error) {
	// ログディレクトリが存在しない場合は作成
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 現在時刻からファイル名を生成
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(logDir, fmt.Sprintf("%s.log", timestamp))

	// ファイルを開く（存在しない場合は作成、存在する場合は追記）
	// #nosec G304 -- logDir comes from trusted configuration
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	return &CustomHandler{
		stdoutHandler: newColoredHandler(os.Stdout, opts),
		fileHandler:   slog.NewTextHandler(file, opts),
		file:          file,
	}, nil
}

// Close はログファイルをクローズします。
// ファイルがない場合（標準出力のみの場合）は何もしません。
func (h *CustomHandler) Close() error {
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

// Enabled はログレベルが有効かどうかを判定します。
func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stdoutHandler.Enabled(ctx, level)
}

// Handle はログレコードを処理します。
// 標準出力とファイルの両方に出力します。
func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	// 標準出力に出力
	if err := h.stdoutHandler.Handle(ctx, r); err != nil {
		return err
	}

	// ファイルハンドラーがある場合はファイルにも出力
	if h.fileHandler != nil {
		if err := h.fileHandler.Handle(ctx, r); err != nil {
			return err
		}
	}

	return nil
}

// WithAttrs は属性を追加したハンドラーを返します。
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &CustomHandler{
		stdoutHandler: h.stdoutHandler.WithAttrs(attrs),
		file:          h.file,
	}
	if h.fileHandler != nil {
		newHandler.fileHandler = h.fileHandler.WithAttrs(attrs)
	}
	return newHandler
}

// WithGroup はグループを追加したハンドラーを返します。
func (h *CustomHandler) WithGroup(name string) slog.Handler {
	newHandler := &CustomHandler{
		stdoutHandler: h.stdoutHandler.WithGroup(name),
		file:          h.file,
	}
	if h.fileHandler != nil {
		newHandler.fileHandler = h.fileHandler.WithGroup(name)
	}
	return newHandler
}

// coloredHandler は色付きログハンドラーです。
type coloredHandler struct {
	opts   *slog.HandlerOptions
	writer io.Writer
	attrs  []slog.Attr
	groups []string
}

// newColoredHandler は新しいcoloredHandlerを作成します。
func newColoredHandler(w io.Writer, opts *slog.HandlerOptions) *coloredHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &coloredHandler{
		opts:   opts,
		writer: w,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

func (h *coloredHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *coloredHandler) Handle(_ context.Context, r slog.Record) error {
	// 時刻とレベル
	t := r.Time.Format("2006/01/02 15:04:05")
	level := r.Level.String()

	// 色付きのメッセージを構築
	coloredTime := logColorize(r.Level, t)
	coloredLevel := logColorize(r.Level, fmt.Sprintf("[%s]", level))
	coloredMsg := logColorize(r.Level, r.Message)

	msg := fmt.Sprintf("%s %s %s", coloredTime, coloredLevel, coloredMsg)

	// グループと属性を追加
	if len(h.groups) > 0 {
		msg += " " + strings.Join(h.groups, ".")
	}

	// 保存された属性を追加
	for _, attr := range h.attrs {
		msg += fmt.Sprintf(" %s=%v", attr.Key, attr.Value)
	}

	// レコードの属性を追加
	r.Attrs(func(a slog.Attr) bool {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	msg += "\n"
	_, err := h.writer.Write([]byte(msg))
	return err
}

func (h *coloredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &coloredHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

func (h *coloredHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &coloredHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// logColorize はログレベルに応じてメッセージに色を付けます。
func logColorize(level slog.Level, msg string) string {
	switch level {
	case slog.LevelDebug:
		return fmt.Sprintf("\033[0;38;5;245m%s\033[0m", msg) // Gray
	case slog.LevelInfo:
		return fmt.Sprintf("\033[0;37m%s\033[0m", msg) // White
	case slog.LevelWarn:
		return fmt.Sprintf("\033[0;33m%s\033[0m", msg) // Yellow
	case slog.LevelError:
		return fmt.Sprintf("\033[0;35m%s\033[0m", msg) // Magenta
	default:
		return msg
	}
}
