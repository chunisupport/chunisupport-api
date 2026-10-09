package handler

import (
	"net/http"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/stretchr/testify/assert"
)

func TestParseBoolQuery(t *testing.T) {
	tests := []struct {
		name string
		// Given
		value string
		// Then
		want    bool
		wantErr bool
	}{
		{name: "未指定はfalse", value: "", want: false},
		{name: "trueはtrue", value: "true", want: true},
		{name: "falseはfalse", value: "false", want: false},
		{name: "大文字のTRUEは不正", value: "TRUE", wantErr: true},
		{name: "1は不正", value: "1", wantErr: true},
		{name: "任意の文字列は不正", value: "yes", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, apiErr := ParseBoolQuery("include_deleted", tt.value)

			// Then
			if tt.wantErr {
				if assert.NotNil(t, apiErr) {
					assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
					assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
				}
				return
			}
			assert.Nil(t, apiErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseEnumQuery(t *testing.T) {
	tests := []struct {
		name string
		// Given
		value string
		// Then
		want    string
		wantErr bool
	}{
		{name: "未指定は空文字", value: "", want: ""},
		{name: "許可された値はそのまま返す", value: "rating", want: "rating"},
		{name: "許可されていない値は不正", value: "all", wantErr: true},
		{name: "大文字小文字が異なる値は不正", value: "Rating", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, apiErr := ParseEnumQuery("view", tt.value, "rating", "record")

			// Then
			if tt.wantErr {
				if assert.NotNil(t, apiErr) {
					assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
					assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
				}
				return
			}
			assert.Nil(t, apiErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
