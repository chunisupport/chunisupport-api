package datatransfer

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

type Codec struct {
	secret []byte
	limits codecLimits
}

type codecLimits struct {
	EnvelopeMaxBytes          int
	CompressedPayloadMaxBytes int
	PayloadMaxBytes           int
}

// NewCodec は、呼び出し元による鍵バッファの変更が署名結果へ影響しないコーデックを生成します。
func NewCodec(secret []byte) (usecase.UserDataTransferCodec, error) {
	return newCodec(secret, codecLimits{
		EnvelopeMaxBytes:          info.DataTransferEnvelopeMaxBytes,
		CompressedPayloadMaxBytes: info.DataTransferCompressedPayloadMaxBytes,
		PayloadMaxBytes:           info.DataTransferPayloadMaxBytes,
	})
}

func newCodec(secret []byte, limits codecLimits) (*Codec, error) {
	if len(secret) < info.DataTransferHMACSecretMinBytes {
		return nil, errors.New("data transfer HMAC secret is too short")
	}
	if limits.EnvelopeMaxBytes <= 0 || limits.CompressedPayloadMaxBytes <= 0 || limits.PayloadMaxBytes <= 0 {
		return nil, errors.New("data transfer codec limits must be positive")
	}
	return &Codec{secret: bytes.Clone(secret), limits: limits}, nil
}

func (codec *Codec) Encode(snapshot *entity.UserDataTransferSnapshot) ([]byte, error) {
	if snapshot == nil {
		return nil, usecase.ErrDataTransferInvalidData
	}
	if err := snapshot.Validate(); err != nil {
		return nil, fmt.Errorf("%w: snapshot validation failed", usecase.ErrDataTransferInvalidData)
	}

	headerBytes, err := json.Marshal(protectedHeaderV1{
		Format:        info.DataTransferFormat,
		SchemaVersion: info.DataTransferSchemaVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode data transfer header", usecase.ErrInternalError)
	}

	payloadBytes, err := json.Marshal(newPayloadV1(snapshot))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode data transfer payload", usecase.ErrInternalError)
	}
	if len(payloadBytes) > codec.limits.PayloadMaxBytes {
		return nil, usecase.ErrDataTransferPayloadTooLarge
	}
	compressedPayload, err := gzipPayload(payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to compress data transfer payload", usecase.ErrInternalError)
	}
	if len(compressedPayload) > codec.limits.CompressedPayloadMaxBytes {
		return nil, usecase.ErrDataTransferPayloadTooLarge
	}

	protected := base64.RawURLEncoding.EncodeToString(headerBytes)
	payload := base64.RawURLEncoding.EncodeToString(compressedPayload)
	signature := codec.sign(protected, payload)
	encoded, err := json.Marshal(envelope{
		Protected: protected,
		Payload:   payload,
		Signature: base64.RawURLEncoding.EncodeToString(signature),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode data transfer envelope", usecase.ErrInternalError)
	}
	if len(encoded) > codec.limits.EnvelopeMaxBytes {
		return nil, usecase.ErrDataTransferPayloadTooLarge
	}
	return encoded, nil
}

func (codec *Codec) Decode(encoded []byte) (*entity.UserDataTransferSnapshot, error) {
	if len(encoded) > codec.limits.EnvelopeMaxBytes {
		return nil, usecase.ErrDataTransferPayloadTooLarge
	}
	if len(encoded) == 0 {
		return nil, usecase.ErrDataTransferInvalidFile
	}

	var transferEnvelope envelope
	if err := decodeStrictJSON(encoded, &transferEnvelope); err != nil {
		return nil, fmt.Errorf("%w: invalid envelope", usecase.ErrDataTransferInvalidFile)
	}
	if transferEnvelope.Protected == "" || transferEnvelope.Payload == "" || transferEnvelope.Signature == "" {
		return nil, fmt.Errorf("%w: envelope fields are required", usecase.ErrDataTransferInvalidFile)
	}

	headerBytes, err := decodeBase64URL(transferEnvelope.Protected)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid protected header encoding", usecase.ErrDataTransferInvalidFile)
	}
	compressedPayload, err := decodeBase64URL(transferEnvelope.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid payload encoding", usecase.ErrDataTransferInvalidFile)
	}
	if len(compressedPayload) > codec.limits.CompressedPayloadMaxBytes {
		return nil, usecase.ErrDataTransferPayloadTooLarge
	}
	signature, err := decodeBase64URL(transferEnvelope.Signature)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid signature encoding", usecase.ErrDataTransferInvalidFile)
	}
	if len(signature) != sha256.Size || !hmac.Equal(signature, codec.sign(transferEnvelope.Protected, transferEnvelope.Payload)) {
		return nil, usecase.ErrDataTransferInvalidSignature
	}

	var header protectedHeaderV1
	if err := decodeStrictJSON(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("%w: invalid protected header", usecase.ErrDataTransferInvalidFile)
	}
	if header.Format == "" || header.SchemaVersion == 0 {
		return nil, fmt.Errorf("%w: header format and schema version are required", usecase.ErrDataTransferInvalidFile)
	}
	if header.Format != info.DataTransferFormat ||
		header.SchemaVersion < info.DataTransferMinSupportedSchemaVersion ||
		header.SchemaVersion > info.DataTransferSchemaVersion {
		return nil, usecase.ErrDataTransferUnsupportedSchema
	}

	payloadBytes, err := gunzipPayload(compressedPayload, codec.limits.PayloadMaxBytes)
	if err != nil {
		return nil, err
	}
	var payload payloadV1
	if err := decodeStrictJSON(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("%w: invalid payload", usecase.ErrDataTransferInvalidFile)
	}
	snapshot, err := payload.toSnapshot()
	if err != nil {
		return nil, fmt.Errorf("%w: payload conversion failed", usecase.ErrDataTransferInvalidData)
	}
	if err := snapshot.Validate(); err != nil {
		return nil, fmt.Errorf("%w: snapshot validation failed", usecase.ErrDataTransferInvalidData)
	}

	return snapshot, nil
}

func (codec *Codec) sign(protected, payload string) []byte {
	mac := hmac.New(sha256.New, codec.secret)
	_, _ = io.WriteString(mac, protected)
	_, _ = io.WriteString(mac, ".")
	_, _ = io.WriteString(mac, payload)
	return mac.Sum(nil)
}

func decodeBase64URL(value string) ([]byte, error) {
	if value == "" || strings.ContainsAny(value, "=\r\n\t ") {
		return nil, errors.New("non-canonical Base64URL")
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, err
	}
	if base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, errors.New("non-canonical Base64URL")
	}
	return decoded, nil
}

func decodeStrictJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func gzipPayload(payload []byte) ([]byte, error) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(payload); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func gunzipPayload(compressed []byte, limit int) ([]byte, error) {
	compressedReader := bytes.NewReader(compressed)
	reader, err := gzip.NewReader(compressedReader)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid gzip payload", usecase.ErrDataTransferInvalidFile)
	}
	reader.Multistream(false)
	payload, readErr := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	closeErr := reader.Close()
	if len(payload) > limit {
		return nil, usecase.ErrDataTransferPayloadTooLarge
	}
	if readErr != nil || closeErr != nil || compressedReader.Len() != 0 {
		return nil, fmt.Errorf("%w: invalid gzip payload", usecase.ErrDataTransferInvalidFile)
	}
	return payload, nil
}
