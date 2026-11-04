package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// XAccountHandler は連携中の X アカウント情報のリクエストを処理します。
type XAccountHandler struct {
	xAccountUsecase usecase.XAccountUsecase
}

// NewXAccountHandler は XAccountHandler を生成します。
func NewXAccountHandler(xAccountUsecase usecase.XAccountUsecase) *XAccountHandler {
	return &XAccountHandler{xAccountUsecase: xAccountUsecase}
}

// Get は保存済みの X アカウント情報を返します。
func (h *XAccountHandler) Get(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}

	account, err := h.xAccountUsecase.Get(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &internaldto.XAccountResponse{XAccount: toXAccountDTO(account)})
}

// Sync は X のアクセストークンで本人のユーザー名を取得し直して保存します。
func (h *XAccountHandler) Sync(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}

	var req internaldto.SyncXAccountRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	account, err := h.xAccountUsecase.Sync(c.Request().Context(), user, usecase.XAccountSyncInput{
		AccessToken:       req.AccessToken,
		AccessTokenSecret: req.AccessTokenSecret,
	})
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toXAccountDTO(account))
}

func toXAccountDTO(account *usecase.XAccountOutput) *internaldto.XAccountDTO {
	if account == nil {
		return nil
	}
	return &internaldto.XAccountDTO{ID: account.ID, Username: account.Username}
}
