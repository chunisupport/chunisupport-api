package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type strictJSONSample struct {
	Name string `json:"name"`
}

func TestDecodeStrictJSON(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     string
		wantName    string
	}{
		{
			name:        "正常なJSONをデコードできる",
			contentType: echo.MIMEApplicationJSON,
			body:        `{"name":"ok"}`,
			wantName:    "ok",
		},
		{
			name:        "charset付きContent-Typeを許可する",
			contentType: "application/json; charset=utf-8",
			body:        `{"name":"ok"}`,
			wantName:    "ok",
		},
		{
			name:        "先頭JSONの後の空白のみは許可する",
			contentType: echo.MIMEApplicationJSON,
			body:        "{\"name\":\"ok\"}\n  ",
			wantName:    "ok",
		},
		{
			name:    "Content-Typeなしは拒否する",
			body:    `{"name":"ok"}`,
			wantErr: "content-type header is missing",
		},
		{
			name:        "Content-Type不正は拒否する",
			contentType: "text/plain",
			body:        `{"name":"ok"}`,
			wantErr:     "content-type must be application/json",
		},
		{
			name:        "未知フィールドは拒否する",
			contentType: echo.MIMEApplicationJSON,
			body:        `{"name":"ok","unknown":1}`,
			wantErr:     "unknown field",
		},
		{
			name:        "複数JSON値は拒否する",
			contentType: echo.MIMEApplicationJSON,
			body:        `{"name":"ok"} {}`,
			wantErr:     "unexpected trailing JSON value",
		},
		{
			name:        "構文不正なJSONは拒否する",
			contentType: echo.MIMEApplicationJSON,
			body:        `{"name":`,
			wantErr:     "EOF",
		},
		{
			name:        "空ボディは拒否する",
			contentType: echo.MIMEApplicationJSON,
			body:        "",
			wantErr:     "EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			if tt.contentType != "" {
				header.Set(echo.HeaderContentType, tt.contentType)
			}

			var out strictJSONSample
			err := DecodeStrictJSON(bytes.NewBufferString(tt.body), header, &out)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, out.Name)
		})
	}
}

func TestBindStrictJSON(t *testing.T) {
	e := echo.New()

	t.Run("正常なJSONをデコードできる", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c := e.NewContext(req, httptest.NewRecorder())

		var out strictJSONSample
		err := BindStrictJSON(c, &out)

		require.NoError(t, err)
		assert.Equal(t, "ok", out.Name)
	})

	t.Run("未知フィールドは拒否する", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok","unknown":true}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c := e.NewContext(req, httptest.NewRecorder())

		var out strictJSONSample
		err := BindStrictJSON(c, &out)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})
}

func TestDecodeStrictJSON_ネストした未知フィールドも拒否する(t *testing.T) {
	type nested struct {
		Item strictJSONSample `json:"item"`
	}
	header := http.Header{}
	header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	var out nested
	err := DecodeStrictJSON(bytes.NewBufferString(`{"item":{"name":"ok","unknown":1}}`), header, &out)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestDecodeStrictJSON_配列要素の未知フィールドも拒否する(t *testing.T) {
	header := http.Header{}
	header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	var out []*strictJSONSample
	err := DecodeStrictJSON(bytes.NewBufferString(`[{"name":"ok","unknown":1}]`), header, &out)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestDecodeStrictJSON_ioEOF以外の後続エラーを返す(t *testing.T) {
	header := http.Header{}
	header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	var out strictJSONSample
	err := DecodeStrictJSON(bytes.NewBufferString(`{"name":"ok"} {`), header, &out)

	require.Error(t, err)
	assert.NotEqual(t, "unexpected trailing JSON value", err.Error())
	assert.NotErrorIs(t, err, io.EOF)
}

func TestDecodeStrictJSON_後続オブジェクトの未知フィールドも拒否する(t *testing.T) {
	header := http.Header{}
	header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	var out strictJSONSample
	err := DecodeStrictJSON(bytes.NewBufferString(`{"name":"ok"} {"name":"ng"}`), header, &out)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}
