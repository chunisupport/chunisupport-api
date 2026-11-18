package handler

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
)

// contextKeyUserEntity は認証ミドルウェアがリクエストユーザーを格納するキーです。
// internal/app/middleware の contextKeyUserEntity と同じ値に揃えています。
const contextKeyUserEntity = "userEntity"

// contextKeyAPIToken はAPIトークン認証ミドルウェアが検証済みトークンを格納するキーです。
// internal/app/middleware の contextKeyAPIToken と同じ値に揃えています。
const contextKeyAPIToken = "apiToken"

// Requester はコンテキストからリクエストユーザーを取り出します。
// 未設定、または *entity.User でない場合は nil を返します。
// 認証を必須としないエンドポイントで、未認証を nil として扱う現行の取り出しに使います。
func Requester(c *echo.Context) *entity.User {
	user, _ := c.Get(contextKeyUserEntity).(*entity.User)
	return user
}

// Viewer は非公開ユーザーの閲覧判定に使う閲覧者を返します。
// 公開ユーザーのみに制限されたAPIトークンでは、本人・フレンドとしての閲覧を許さないため未認証と同じ nil を返します。
// 役割判定やレート制限はトークン所有者で行う必要があるため、Requester 自体は変えずに閲覧判定の箇所だけで使います。
func Viewer(c *echo.Context) *entity.User {
	if token, ok := c.Get(contextKeyAPIToken).(*entity.APIToken); ok && token != nil && !token.CanViewPrivateUsers() {
		return nil
	}
	return Requester(c)
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
