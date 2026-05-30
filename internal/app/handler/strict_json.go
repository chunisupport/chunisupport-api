package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// BindStrictJSON は echo.Context からヘッダー/ボディを取り出して厳格なJSONデコードを行います。
func BindStrictJSON(c echo.Context, out any) error {
	return DecodeStrictJSON(c.Request().Body, c.Request().Header, out)
}

// ValidateJSONContentType は Content-Type が application/json かを検証します。
func ValidateJSONContentType(header http.Header) error {
	ct := header.Get(echo.HeaderContentType)
	if ct == "" {
		return errors.New("content-type header is missing")
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || !strings.EqualFold(mediaType, echo.MIMEApplicationJSON) {
		return errors.New("content-type must be application/json")
	}
	return nil
}

// DecodeStrictJSON は unknown field / trailing value を拒否する厳格なJSONデコードを行います。
func DecodeStrictJSON(body io.Reader, header http.Header, out any) error {
	if err := ValidateJSONContentType(header); err != nil {
		return err
	}

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
