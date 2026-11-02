package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
)

// UserLogAttrs は認証済みユーザーの識別子をログ属性として返します。
// エラー通知からどのユーザーのリクエストで発生したかを追えるようにするためのものです。
// 未認証のリクエストでは nil を返し、プレイヤー未連携のユーザーには player_id を付けません。
func UserLogAttrs(c *echo.Context) []any {
	user, ok := c.Get(contextKeyUserEntity).(*entity.User)
	if !ok || user == nil {
		return nil
	}

	attrs := []any{"user_id", user.ID, "username", user.Username.String()}
	if user.PlayerID != nil {
		attrs = append(attrs, "player_id", *user.PlayerID)
	}
	return attrs
}
