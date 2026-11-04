package api_internal

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
)

// getUserEntityFromContext は認証必須エンドポイント向けにリクエストユーザーを取り出します。
// 未設定または型が一致しない場合のみ 401 を返します。値が nil の *entity.User は従来どおりエラーにしません。
func getUserEntityFromContext(c *echo.Context) (*entity.User, error) {
	user, apiErr := apphandler.RequireRequester(c)
	if apiErr != nil {
		return nil, apiErr
	}
	return user, nil
}

// getUser は認証必須エンドポイント向けにリクエストユーザーを取り出します。
// 未設定、型不一致、nil ユーザーはいずれも 401 を返します。
func getUser(c *echo.Context) (*entity.User, error) {
	user, apiErr := apphandler.RequireRequester(c)
	if apiErr != nil || user == nil {
		return nil, apierror.ErrUnauthorized
	}
	return user, nil
}
