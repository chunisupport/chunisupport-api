package handler

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
)

// contextKeyUserEntity は認証ミドルウェアがリクエストユーザーを格納するキーです。
// internal/app/middleware の contextKeyUserEntity と同じ値に揃えています。
const contextKeyUserEntity = "userEntity"

// Requester はコンテキストからリクエストユーザーを取り出します。
// 未設定、または *entity.User でない場合は nil を返します。
// 認証を必須としないエンドポイントで、未認証を nil として扱う現行の取り出しに使います。
func Requester(c *echo.Context) *entity.User {
	user, _ := c.Get(contextKeyUserEntity).(*entity.User)
	return user
}

// RequireRequester は認証必須エンドポイント向けにリクエストユーザーを取り出します。
// 未設定、または *entity.User でない場合は 401 を返します。
// 型は一致するが値が nil の場合は (nil, nil) を返します。nil を未認証とみなすかは呼び出し側の既存判定に任せ、
// ok だけを見て 401 にしている箇所の挙動を変えないためです。
func RequireRequester(c *echo.Context) (*entity.User, *apierror.APIError) {
	user, ok := c.Get(contextKeyUserEntity).(*entity.User)
	if !ok {
		return nil, apierror.ErrUnauthorized
	}
	return user, nil
}
