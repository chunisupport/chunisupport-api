package api_internal

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
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
	UploadToken string    `json:"uploadToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type commitTemporaryPlayerDataRequest struct {
	UploadToken string `json:"uploadToken" validate:"required,uuid4"`
}

// CreateTemporaryData は未ログインユーザーの一時アップロードを受け付けます。
// gzip圧縮されたプレイヤーデータJSONを受け取るため BindStrictJSON は使いません。
// 高圧縮率の本文でメモリや保持容量を消費されないよう、未認証の受付時はJSONをデコードせず、受信したgzip本文をそのまま保持します。
func (h *TemporaryPlayerDataHandler) CreateTemporaryData(c *echo.Context) error {
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

	hash := sha256.New()
	uncompressedBytes, err := io.Copy(hash, io.LimitReader(gzipReader, int64(info.TempDataMaxUncompressedBytes)+1))
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if uncompressedBytes > info.TempDataMaxUncompressedBytes {
		return apierror.ErrPayloadTooLarge
	}

	result, err := h.temporaryPlayerDataUsecase.Create(c.Request().Context(), usecase.CreateTemporaryPlayerDataInput{
		IPAddress: middleware.ClientIPKey(c),
		Payload:   compressedBody,
		BodyHash:  hex.EncodeToString(hash.Sum(nil)),
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
		ExpiresAt:   result.ExpiresAt,
	})
}

// DecodeTemporaryPlayerDataPayload は一時保存されたgzip本文をプレイヤーデータへデコードします。
// 入力JSONの形式はこの層が持つため、確定保存時にユースケースへ渡して使います。未知フィールドは json.Unmarshal により無視されます。
func DecodeTemporaryPlayerDataPayload(stored []byte) (*usecase.PlayerDataPayload, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(stored))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	jsonBody, exceeded, err := readAllWithMaxBytes(gzipReader, info.TempDataMaxUncompressedBytes)
	if err != nil {
		return nil, err
	}
	if exceeded {
		return nil, errors.New("temporary player data exceeds max uncompressed size")
	}

	var request playerDataRequest
	if err := json.Unmarshal(jsonBody, &request); err != nil {
		return nil, err
	}
	payload := request.toUsecase()
	return &payload, nil
}

func readAllWithMaxBytes(r io.Reader, maxBytes int) ([]byte, bool, error) {
	if maxBytes < 0 {
		return nil, false, errors.New("maxBytes must be non-negative")
	}

	body, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return nil, false, err
	}
	if len(body) > maxBytes {
		return nil, true, nil
	}

	return body, false, nil
}

// CommitTemporaryData は一時データを認証済みユーザーに紐づけて確定保存します。
func (h *TemporaryPlayerDataHandler) CommitTemporaryData(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
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
		case errors.Is(err, usecase.ErrTemporaryPlayerDataNotFound):
			return apierror.ErrNotFound.WithInternal(err)
		case errors.Is(err, usecase.ErrTemporaryPlayerDataInProgress):
			return apierror.ErrConflict.WithInternal(err)
		case errors.Is(err, usecase.ErrTempDataPayloadInvalidJSON):
			return apierror.ErrBadRequest.WithInternal(err)
		default:
			return apierror.FromUsecaseError(err)
		}
	}

	return c.JSON(http.StatusOK, toPlayerDataResponse(result))
}
