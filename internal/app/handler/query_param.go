package handler

import (
	"fmt"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
)

// ParseBoolQuery は真偽値のクエリパラメータを解析します。
// 未指定は false とします。クライアントの誤りを検出できるよう、"true" / "false" 以外は既定値へ丸めず検証エラーにします。
func ParseBoolQuery(name string, value string) (bool, *apierror.APIError) {
	switch value {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, apierror.ErrValidationFailed.WithInternal(fmt.Errorf("%s must be true or false", name))
	}
}

// ParseEnumQuery は列挙値のクエリパラメータを解析します。
// 未指定は空文字を返し、許可された値以外は既定の挙動へ丸めず検証エラーにします。
func ParseEnumQuery(name string, value string, allowed ...string) (string, *apierror.APIError) {
	if value == "" || slices.Contains(allowed, value) {
		return value, nil
	}
	return "", apierror.ErrValidationFailed.WithInternal(fmt.Errorf("%s must be one of %v", name, allowed))
}
