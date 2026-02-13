package api_internal

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

const (
	maxPlayerDataPayloadSize = 5 * 1024 * 1024 // 5MB
)

// MeHandler は認証済みユーザー向けエンドポイントを扱います。
type MeHandler struct {
	playerDataUsecase usecase.PlayerDataUsecase
}

// NewMeHandler は MeHandler のインスタンスを生成します。
func NewMeHandler(playerDataUsecase usecase.PlayerDataUsecase) *MeHandler {
	return &MeHandler{playerDataUsecase: playerDataUsecase}
}

// RegisterData はプレイヤーデータの登録を受け付けます。
// デフォルトではbase64+gzip圧縮形式を受け入れ、クエリパラメータformat=jsonの場合は生JSONを受け入れます。
func (h *MeHandler) RegisterData(c echo.Context) error {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok || user == nil {
		return apierror.ErrUnauthorized
	}

	// クエリパラメータでフォーマットを確認
	format := c.QueryParam("format")

	limitedReader := io.LimitReader(c.Request().Body, maxPlayerDataPayloadSize+1)
	raw, err := io.ReadAll(limitedReader)
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if len(raw) == 0 {
		return apierror.ErrBadRequest
	}
	if len(raw) > maxPlayerDataPayloadSize {
		return apierror.ErrPayloadTooLarge
	}

	var jsonData []byte
	var hash [32]byte

	if format == "json" {
		// 生JSON形式（デバッグ用）
		jsonData = raw
		hash = sha256.Sum256(raw)
	} else {
		// デフォルト: base64+gzip形式
		// ハッシュは圧縮前のJSONデータに対して計算
		decompressed, err := decodeAndDecompressGzipBase64(raw)
		if err != nil {
			return apierror.ErrBadRequest.WithInternal(err)
		}

		// 解凍後のサイズチェック
		if len(decompressed) > maxPlayerDataPayloadSize {
			return apierror.ErrPayloadTooLarge
		}

		jsonData = decompressed
		hash = sha256.Sum256(decompressed)
	}

	hashText := hex.EncodeToString(hash[:])

	// 未知のフィールドを検出するため、まずmapにデコード
	var rawMap map[string]any
	if err := json.Unmarshal(jsonData, &rawMap); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	// PlayerDataPayloadの既知のフィールド（jsonタグ）
	knownFields := map[string]bool{
		"app_ver":      true,
		"name":         true,
		"level":        true,
		"rating":       true,
		"last_played":  true,
		"overpower":    true,
		"class_emblem": true,
		"team":         true,
		"honors":       true,
		"scores":       true,
		"updated_at":   true,
	}

	// 未知のフィールドを検出
	var unknownFields []string
	for key := range rawMap {
		if !knownFields[key] {
			unknownFields = append(unknownFields, key)
		}
	}

	// 未知のフィールドがあれば警告ログを出力
	if len(unknownFields) > 0 {
		c.Logger().Warnf("unknown fields in player data payload: %v (user: %s)", unknownFields, user.ID)
	}

	// 正式な構造体にデコード（未知フィールドは無視される）
	var payload usecase.PlayerDataPayload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	result, err := h.playerDataUsecase.Register(c.Request().Context(), user, &payload, hashText)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, result)
}

// DeletePlayerData はプレイヤーデータの削除（連携解除）を扱います。
func (h *MeHandler) DeletePlayerData(c echo.Context) error {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok || user == nil {
		return apierror.ErrUnauthorized
	}

	if err := h.playerDataUsecase.Delete(c.Request().Context(), user); err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// decodeAndDecompressGzipBase64 はbase64エンコードされたgzip圧縮データをデコード・解凍します。
func decodeAndDecompressGzipBase64(data []byte) ([]byte, error) {
	// Base64デコード
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		return nil, err
	}
	decoded = decoded[:n]

	// Gzip解凍（Gzip Bomb対策: 解凍後サイズに上限を設定）
	gzipReader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	// io.LimitReaderで解凍後サイズを制限し、圧縮爆弾によるメモリ枯渇を防止
	limitedReader := io.LimitReader(gzipReader, int64(maxPlayerDataPayloadSize)+1)
	decompressed, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}
