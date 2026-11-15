package app

import (
	"errors"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	vo_username "github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/go-playground/validator/v10"
)

// CustomValidator はリクエストの検証を行うための構造体です。
type CustomValidator struct {
	Validator *validator.Validate
}

// NewCustomValidator は新しいCustomValidatorを生成します。
func NewCustomValidator() *CustomValidator {
	v := validator.New()
	if err := v.RegisterValidation("username", validateUsername); err != nil {
		panic(err)
	}
	return &CustomValidator{Validator: v}
}

func validateUsername(fl validator.FieldLevel) bool {
	_, err := vo_username.NewUserName(fl.Field().String())
	return err == nil
}

// Validate は与えられた構造体を検証します。
func (cv *CustomValidator) Validate(i any) error {
	if err := cv.Validator.Struct(i); err != nil {
		slog.Warn("Validation error", "error", err.Error())
		if validationErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
			return apierror.ErrValidationFailed.WithInternal(apierror.ValidationErrors(validationErrors))
		}
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	return nil
}
