package api_internal

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// TemporaryPlayerDataHandler はプレイヤーデータ一時受付・確定保存APIを扱います。
type TemporaryPlayerDataHandler struct {
	temporaryPlayerDataUsecase usecase.TemporaryPlayerDataUsecase
}

// NewTemporaryPlayerDataHandler はハンドラを生成します。
func NewTemporaryPlayerDataHandler(temporaryPlayerDataUsecase usecase.TemporaryPlayerDataUsecase) *TemporaryPlayerDataHandler {
	return &TemporaryPlayerDataHandler{temporaryPlayerDataUsecase: temporaryPlayerDataUsecase}
}

type createTemporaryPlayerDataResponse struct {
	UploadToken string `json:"uploadToken"`
	ExpiresAt   string `json:"expiresAt"`
}

type commitTemporaryPlayerDataRequest struct {
	UploadToken string `json:"uploadToken" validate:"required,uuid4"`
}

// CreateTemporaryData は未ログインユーザーの一時アップロードを受け付けます。
func (h *TemporaryPlayerDataHandler) CreateTemporaryData(c echo.Context) error {
	if userObj := c.Get("userEntity"); userObj != nil {
		return apierror.ErrBadRequest
	}

	if !strings.EqualFold(c.Request().Header.Get(echo.HeaderContentEncoding), "gzip") {
		return apierror.ErrBadRequest
	}
	if err := apphandler.ValidateJSONContentType(c.Request().Header); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	compressedBody, exceeded, err := readAllWithMaxBytes(c.Request().Body, info.TempDataMaxCompressedBytes)
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if exceeded {
		return apierror.ErrPayloadTooLarge
	}
	if len(compressedBody) == 0 {
		return apierror.ErrBadRequest
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedBody))
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	defer gzipReader.Close()

	jsonBody, exceeded, err := readAllWithMaxBytes(gzipReader, info.TempDataMaxUncompressedBytes)
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if exceeded {
		return apierror.ErrPayloadTooLarge
	}

	hash := sha256.Sum256(jsonBody)
	result, err := h.temporaryPlayerDataUsecase.Create(c.Request().Context(), usecase.CreateTemporaryPlayerDataInput{
		IPAddress: c.RealIP(),
		Payload:   jsonBody,
		BodyHash:  hex.EncodeToString(hash[:]),
	})
	if err != nil {
		switch {
		case err == usecase.ErrTempDataPerIPLimitExceeded:
			return apierror.ErrConflict.WithInternal(err)
		case err == usecase.ErrTempDataCapacityExceeded:
			return apierror.ErrServiceUnavailable.WithInternal(err)
		default:
			return apierror.FromUsecaseError(err)
		}
	}

	return c.JSON(http.StatusCreated, createTemporaryPlayerDataResponse{
		UploadToken: result.UploadToken,
		ExpiresAt:   result.ExpiresAt.Format(time.RFC3339),
	})
}

func readAllWithMaxBytes(r io.Reader, maxBytes int) ([]byte, bool, error) {
	if maxBytes < 0 {
		return nil, false, errors.New("maxBytes must be non-negative")
	}

	limited := &io.LimitedReader{R: r, N: int64(maxBytes)}
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}

	if limited.N > 0 {
		return body, false, nil
	}

	var probe [1]byte
	n, err := r.Read(probe[:])
	if err == nil || (err != nil && !errors.Is(err, io.EOF)) {
		if err != nil {
			return nil, false, err
		}
		return nil, n > 0, nil
	}

	return body, n > 0, nil
}

// CommitTemporaryData は一時データを認証済みユーザーに紐づけて確定保存します。
func (h *TemporaryPlayerDataHandler) CommitTemporaryData(c echo.Context) error {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok || user == nil {
		return apierror.ErrUnauthorized
	}

	var req commitTemporaryPlayerDataRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	result, err := h.temporaryPlayerDataUsecase.Commit(c.Request().Context(), usecase.CommitTemporaryPlayerDataInput{
		User:        user,
		UploadToken: req.UploadToken,
	})
	if err != nil {
		switch {
		case err == usecase.ErrTemporaryPlayerDataNotFound:
			return apierror.ErrNotFound.WithInternal(err)
		default:
			return apierror.FromUsecaseError(err)
		}
	}

	return c.JSON(http.StatusOK, result)
}
