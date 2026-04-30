package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeOptionalStrictJSON(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		header    http.Header
		wantName  string
		wantEmpty bool
		wantErr   string
	}{
		{
			name:      "空ボディの場合はデコードしない",
			body:      "",
			header:    http.Header{},
			wantEmpty: true,
		},
		{
			name:      "空白のみの場合はデコードしない",
			body:      " \n\t\r ",
			header:    http.Header{},
			wantEmpty: true,
		},
		{
			name: "JSONボディの場合はデコードする",
			body: `{"name":"token"}`,
			header: http.Header{
				echo.HeaderContentType: []string{echo.MIMEApplicationJSON},
			},
			wantName: "token",
		},
		{
			name: "先頭に空白があるJSONボディの場合はデコードする",
			body: " \n\t" + `{"name":"token"}`,
			header: http.Header{
				echo.HeaderContentType: []string{echo.MIMEApplicationJSON},
			},
			wantName: "token",
		},
		{
			name:    "JSONボディでContent-Typeがない場合はエラー",
			body:    `{"name":"token"}`,
			header:  http.Header{},
			wantErr: "content-type header is missing",
		},
		{
			name: "未知のキーがある場合はエラー",
			body: `{"name":"token","unknown":true}`,
			header: http.Header{
				echo.HeaderContentType: []string{echo.MIMEApplicationJSON},
			},
			wantErr: `json: unknown field "unknown"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var got struct {
				Name string `json:"name"`
			}

			// When
			err := DecodeOptionalStrictJSON(bytes.NewBufferString(tt.body), tt.header, &got)

			// Then
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantEmpty {
				assert.Empty(t, got.Name)
				return
			}
			assert.Equal(t, tt.wantName, got.Name)
		})
	}
}

func TestBindOptionalStrictJSONKeepsBufferedBodyReadable(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "Content-Typeエラー後もボディを読み取れる",
			body:    `{"name":"token"}`,
			wantErr: "content-type header is missing",
		},
		{
			name: "空白のみの場合もボディを読み取れる",
			body: " \n\t\r ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/tokens", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			var got struct {
				Name string `json:"name"`
			}

			// When
			err := BindOptionalStrictJSON(c, &got)
			body, readErr := io.ReadAll(c.Request().Body)

			// Then
			require.NoError(t, readErr)
			assert.Equal(t, tt.body, string(body))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Empty(t, got.Name)
		})
	}
}
