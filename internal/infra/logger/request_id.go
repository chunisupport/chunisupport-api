package logger

import (
	"context"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/info"
)

type requestIDContextKey struct{}

// WithRequestID はアプリケーションログから参照できる相関IDをコンテキストへ載せます。
// ミドルウェアが受理または生成したIDを、後続の slog.*Context から同じ値で取り出すためです。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext はコンテキストに格納された相関IDを返します。未設定なら空文字です。
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// contextHandler はリクエストコンテキストの相関IDをログ属性へ追加します。
// slog.Info のように context.Background で記録するログへは付けず、リクエスト処理中のログだけを相関させます。
type contextHandler struct {
	handler slog.Handler
}

func newContextHandler(handler slog.Handler) slog.Handler {
	return &contextHandler{handler: handler}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		record.AddAttrs(slog.String(info.LogAttrRequestID, requestID))
	}
	return h.handler.Handle(ctx, record)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{handler: h.handler.WithGroup(name)}
}
