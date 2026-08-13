package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/info"
)

func decodeDataTransferHMACSecret(encodedSecret string, required bool) ([]byte, error) {
	if !required {
		return nil, nil
	}
	encodedSecret = strings.TrimSpace(encodedSecret)
	if encodedSecret == "" {
		return nil, errors.New("DATA_TRANSFER_HMAC_SECRET environment variable is required")
	}
	secret, err := base64.StdEncoding.Strict().DecodeString(encodedSecret)
	if err != nil {
		return nil, errors.New("DATA_TRANSFER_HMAC_SECRET must be valid Base64")
	}
	if len(secret) < info.DataTransferHMACSecretMinBytes {
		return nil, errors.New("DATA_TRANSFER_HMAC_SECRET must decode to at least 32 bytes")
	}
	return bytes.Clone(secret), nil
}
