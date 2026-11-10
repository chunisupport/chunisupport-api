package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// UserProfileCommentHandler は独立したプロフィールコメントAPIを処理します。
type UserProfileCommentHandler struct {
	commentUsecase usecase.UserProfileCommentUsecase
}

// NewUserProfileCommentHandler はコメントのハンドラーを生成します。
func NewUserProfileCommentHandler(commentUsecase usecase.UserProfileCommentUsecase) *UserProfileCommentHandler {
	return &UserProfileCommentHandler{commentUsecase: commentUsecase}
}

// Update は認証ユーザーのコメントを保存し、空文字列なら未設定に戻します。
func (h *UserProfileCommentHandler) Update(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var req internaldto.UpdateUserCommentRequest
	if err := handler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if req.Comment == nil {
		return apierror.ErrBadRequest
	}
	comment, err := h.commentUsecase.Update(c.Request().Context(), user.ID, *req.Comment)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &internaldto.UserCommentResponse{Comment: comment})
}

// Get は既存のプロフィール取得と同じ認可とエラー応答でコメントを返します。
func (h *UserProfileCommentHandler) Get(c *echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	comment, err := h.commentUsecase.Get(c.Request().Context(), username, handler.Requester(c))
	if err != nil {
		return handleUserProfileError(c.Request().Context(), err, username, "user comment")
	}
	return c.JSON(http.StatusOK, &internaldto.UserCommentResponse{Comment: comment})
}
