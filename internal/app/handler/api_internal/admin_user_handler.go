package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// AdminUserHandler はADMIN専用のユーザー関連HTTPリクエストを処理します。
type AdminUserHandler struct {
	userUsecase usecase.UserUsecase
}

// NewAdminUserHandler は新しいAdminUserHandlerを生成します。
func NewAdminUserHandler(userUsecase usecase.UserUsecase) *AdminUserHandler {
	return &AdminUserHandler{userUsecase: userUsecase}
}

// GetAllUsers handles GET /internal/users/
// ADMIN専用で、プライベート・削除済み・プレイヤー未紐付けアカウントを含むすべてのユーザーを返します。
func (h *AdminUserHandler) GetAllUsers(c echo.Context) error {
	pageParam := c.QueryParam("page")
	page := 1
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	name := c.QueryParam("name")

	limit := info.DefaultUserListLimit

	users, err := h.userUsecase.GetAllUsersForAdmin(c.Request().Context(), page, limit, name)
	if err != nil {
		// Logged in usecase
		return apierror.ErrInternalError
	}

	return c.JSON(http.StatusOK, users)
}

// UpdateUserAccountType はADMIN専用で、指定ユーザーの権限を変更します。
func (h *AdminUserHandler) UpdateUserAccountType(c echo.Context) error {
	usernameParam := c.Param("username")
	if _, err := vo_username.NewUserName(usernameParam); err != nil {
		return apierror.FromUsecaseError(err)
	}

	requester, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		return apierror.ErrUnauthorized
	}

	req, err := bindUpdateUserAccountTypeRequest(c)
	if err != nil {
		return err
	}

	result, err := h.userUsecase.ChangeUserAccountType(c.Request().Context(), requester, username, req.AccountType)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, newAdminUserAccountTypeResponse(result, req.AccountType))
}

// bindUpdateUserAccountTypeRequest はBind失敗時のAPIエラー変換をハンドラ本体から分離します。
func bindUpdateUserAccountTypeRequest(c echo.Context) (dto_internal.UpdateUserAccountTypeRequest, error) {
	var req dto_internal.UpdateUserAccountTypeRequest
	if err := c.Bind(&req); err != nil {
		return dto_internal.UpdateUserAccountTypeRequest{}, apierror.ErrBadRequest.WithInternal(err)
	}
	return req, nil
}

// newAdminUserAccountTypeResponse はレスポンス生成を集約し、ハンドラ本体をユースケース呼び出しに集中させます。
func newAdminUserAccountTypeResponse(user *entity.User, accountType string) dto_internal.AdminUserAccountTypeResponse {
	return dto_internal.AdminUserAccountTypeResponse{
		ID:          user.ID,
		UserName:    user.Username.String(),
		AccountType: accountType,
		UpdatedAt:   user.UpdatedAt,
	}
}
