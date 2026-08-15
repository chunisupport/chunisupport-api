package config

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeDataTransferHMACSecret(t *testing.T) {
	validSecret := bytes.Repeat([]byte{0x42}, 32)
	tests := []struct {
		name      string
		encoded   string
		required  bool
		wantError string
	}{
		{name: "アプリ以外では不要", required: false},
		{name: "32バイトのBase64鍵", encoded: base64.StdEncoding.EncodeToString(validSecret), required: true},
		{name: "未設定", required: true, wantError: "DATA_TRANSFER_HMAC_SECRET environment variable is required"},
		{name: "Base64不正", encoded: "not-base64", required: true, wantError: "DATA_TRANSFER_HMAC_SECRET must be valid Base64"},
		{name: "短すぎる鍵", encoded: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 31)), required: true, wantError: "DATA_TRANSFER_HMAC_SECRET must decode to at least 32 bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := decodeDataTransferHMACSecret(tt.encoded, tt.required)

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				assert.Nil(t, secret)
				return
			}
			require.NoError(t, err)
			if tt.required {
				assert.Equal(t, validSecret, secret)
			} else {
				assert.Nil(t, secret)
			}
		})
	}
}
