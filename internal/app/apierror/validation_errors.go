package apierror

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationErrors はクライアントへ返してよい入力形式エラーのみを保持する型です。
type ValidationErrors []validator.FieldError

// Error はerrorインターフェースを実装します。
func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(v))
	for _, fe := range v {
		parts = append(parts, fmt.Sprintf("%s:%s", fe.Field(), fe.Tag()))
	}
	return strings.Join(parts, ",")
}

// Details はクライアント表示用のメッセージ一覧を返します。
func (v ValidationErrors) Details() []ValidationErrorDetail {
	details := make([]ValidationErrorDetail, 0, len(v))
	for _, fe := range v {
		detail := ValidationErrorDetail{Field: strings.ToLower(fe.Field())}
		switch fe.Tag() {
		case "required":
			detail.Message = "必須項目です。"
		case "min":
			detail.Message = fmt.Sprintf("%s文字以上で入力してください。", fe.Param())
		case "max":
			detail.Message = fmt.Sprintf("%s文字以下で入力してください。", fe.Param())
		case "username":
			detail.Message = "5〜50文字の小文字英数字で入力してください。"
		default:
			detail.Message = "入力値の形式が不正です。"
		}
		details = append(details, detail)
	}
	return details
}
