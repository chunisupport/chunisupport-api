package datatransfer

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var codecTestSecret = bytes.Repeat([]byte{0x42}, 32)

func TestCodecEncodeDecode(t *testing.T) {
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	snapshot := codecTestSnapshot(t)

	encoded, err := codec.Encode(snapshot)
	require.NoError(t, err)

	var envelope testEnvelope
	require.NoError(t, json.Unmarshal(encoded, &envelope))
	assert.NotContains(t, envelope.Protected, "=")
	assert.NotContains(t, envelope.Payload, "=")
	assert.NotContains(t, envelope.Signature, "=")
	headerBytes, err := base64.RawURLEncoding.DecodeString(envelope.Protected)
	require.NoError(t, err)
	var header map[string]any
	require.NoError(t, json.Unmarshal(headerBytes, &header))
	assert.Equal(t, map[string]any{"format": info.DataTransferFormat, "schema_version": float64(info.DataTransferSchemaVersion)}, header)

	decoded, err := codec.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, snapshot.Player.Name.String(), decoded.Player.Name.String())
	assert.NotNil(t, decoded.Records)
	assert.NotNil(t, decoded.Goals.Groups)
	assert.NotNil(t, decoded.Goals.Ungrouped)
}
func TestCodecDecodeRejectsTampering(t *testing.T) {
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	encoded := encodeCodecTestFile(t, codec)

	tests := []struct {
		name   string
		mutate func(testEnvelope) testEnvelope
	}{
		{
			name: "protectedの改変",
			mutate: func(envelope testEnvelope) testEnvelope {
				envelope.Protected = mutateBase64URL(envelope.Protected)
				return envelope
			},
		},
		{
			name: "payloadの改変",
			mutate: func(envelope testEnvelope) testEnvelope {
				envelope.Payload = mutateBase64URL(envelope.Payload)
				return envelope
			},
		},
		{
			name: "signatureの改変",
			mutate: func(envelope testEnvelope) testEnvelope {
				envelope.Signature = mutateBase64URL(envelope.Signature)
				return envelope
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var envelope testEnvelope
			require.NoError(t, json.Unmarshal(encoded, &envelope))
			mutated, err := json.Marshal(tt.mutate(envelope))
			require.NoError(t, err)

			_, err = codec.Decode(mutated)

			assert.ErrorIs(t, err, usecase.ErrDataTransferInvalidSignature)
		})
	}

	otherCodec, err := NewCodec(bytes.Repeat([]byte{0x24}, 32))
	require.NoError(t, err)
	_, err = otherCodec.Decode(encoded)
	assert.ErrorIs(t, err, usecase.ErrDataTransferInvalidSignature)
}

func TestCodecDecodeValidatesHeader(t *testing.T) {
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	encoded := encodeCodecTestFile(t, codec)

	t.Run("未対応スキーマを拒否する", func(t *testing.T) {
		mutated := mutateProtectedJSON(t, encoded, func(header map[string]any) {
			header["schema_version"] = float64(99)
		})

		_, err := codec.Decode(mutated)
		assert.ErrorIs(t, err, usecase.ErrDataTransferUnsupportedSchema)
	})

	t.Run("ヘッダーの未知フィールドを拒否する", func(t *testing.T) {
		mutated := mutateProtectedJSON(t, encoded, func(header map[string]any) {
			header["unknown"] = true
		})

		_, err := codec.Decode(mutated)
		assert.ErrorIs(t, err, usecase.ErrDataTransferInvalidFile)
	})
}

func TestCodecDecodeRejectsInvalidEnvelopeAndPayload(t *testing.T) {
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	encoded := encodeCodecTestFile(t, codec)

	tests := []struct {
		name      string
		encoded   func(*testing.T) []byte
		wantError error
	}{
		{
			name: "エンベロープの未知フィールド",
			encoded: func(t *testing.T) []byte {
				var value map[string]any
				require.NoError(t, json.Unmarshal(encoded, &value))
				value["unknown"] = true
				result, err := json.Marshal(value)
				require.NoError(t, err)
				return result
			},
			wantError: usecase.ErrDataTransferInvalidFile,
		},
		{
			name: "パディング付きBase64URL",
			encoded: func(t *testing.T) []byte {
				var envelope testEnvelope
				require.NoError(t, json.Unmarshal(encoded, &envelope))
				envelope.Signature += "="
				result, err := json.Marshal(envelope)
				require.NoError(t, err)
				return result
			},
			wantError: usecase.ErrDataTransferInvalidFile,
		},
		{
			name: "gzipでないpayload",
			encoded: func(t *testing.T) []byte {
				return replacePayloadBytes(t, encoded, []byte("not-gzip"))
			},
			wantError: usecase.ErrDataTransferInvalidFile,
		},
		{
			name: "payloadの未知フィールド",
			encoded: func(t *testing.T) []byte {
				return mutatePayloadJSON(t, encoded, func(payload map[string]any) {
					payload["unknown"] = true
				})
			},
			wantError: usecase.ErrDataTransferInvalidFile,
		},
		{
			name: "null配列",
			encoded: func(t *testing.T) []byte {
				return mutatePayloadJSON(t, encoded, func(payload map[string]any) {
					payload["records"] = nil
				})
			},
			wantError: usecase.ErrDataTransferInvalidData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.Decode(tt.encoded(t))
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

func TestCodecSizeLimits(t *testing.T) {
	tests := []struct {
		name   string
		limits codecLimits
	}{
		{
			name: "エンベロープ上限",
			limits: codecLimits{
				EnvelopeMaxBytes:          1,
				CompressedPayloadMaxBytes: 1024,
				PayloadMaxBytes:           4096,
			},
		},
		{
			name: "圧縮後上限",
			limits: codecLimits{
				EnvelopeMaxBytes:          4096,
				CompressedPayloadMaxBytes: 1,
				PayloadMaxBytes:           4096,
			},
		},
		{
			name: "解凍後上限",
			limits: codecLimits{
				EnvelopeMaxBytes:          4096,
				CompressedPayloadMaxBytes: 1024,
				PayloadMaxBytes:           1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec, err := newCodec(codecTestSecret, tt.limits)
			require.NoError(t, err)

			_, err = codec.Encode(codecTestSnapshot(t))

			assert.ErrorIs(t, err, usecase.ErrDataTransferPayloadTooLarge)
		})
	}

	t.Run("gzip爆弾相当を解凍上限で停止する", func(t *testing.T) {
		codec, err := newCodec(codecTestSecret, codecLimits{
			EnvelopeMaxBytes:          64 * 1024,
			CompressedPayloadMaxBytes: 32 * 1024,
			PayloadMaxBytes:           100,
		})
		require.NoError(t, err)
		normalCodec, err := NewCodec(codecTestSecret)
		require.NoError(t, err)
		encoded := encodeCodecTestFile(t, normalCodec)
		bomb := gzipCodecTestBytes(t, bytes.Repeat([]byte("a"), 4096))

		_, err = codec.Decode(replacePayloadBytes(t, encoded, bomb))

		assert.ErrorIs(t, err, usecase.ErrDataTransferPayloadTooLarge)
	})
}

func TestNewCodecCopiesAndValidatesSecret(t *testing.T) {
	t.Run("32バイト未満を拒否する", func(t *testing.T) {
		_, err := NewCodec(bytes.Repeat([]byte{1}, 31))
		assert.Error(t, err)
	})

	t.Run("呼び出し元の秘密鍵変更に影響されない", func(t *testing.T) {
		secret := bytes.Repeat([]byte{0x42}, 32)
		codec, err := NewCodec(secret)
		require.NoError(t, err)
		encoded := encodeCodecTestFile(t, codec)
		clear(secret)

		_, err = codec.Decode(encoded)

		assert.NoError(t, err)
	})
}

type testEnvelope struct {
	Protected string `json:"protected"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

func codecTestSnapshot(t *testing.T) *entity.UserDataTransferSnapshot {
	t.Helper()
	name, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)
	return &entity.UserDataTransferSnapshot{
		Player: entity.UserDataTransferPlayer{
			Name:              name,
			Level:             1,
			OfficialRating:    0,
			OfficialOverpower: 0,
			CreatedAt:         time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		},
		Records:                  []entity.UserDataTransferRecord{},
		RecordHistories:          []entity.UserDataTransferRecordHistory{},
		WorldsendRecords:         []entity.UserDataTransferWorldsendRecord{},
		WorldsendRecordHistories: []entity.UserDataTransferWorldsendRecordHistory{},
		MetricHistories:          []entity.UserDataTransferMetricHistory{},
		CourseRecords:            []entity.UserDataTransferCourseRecord{},
		Honors:                   []entity.UserDataTransferHonor{},
		FavoriteSongs:            []entity.UserDataTransferFavoriteSong{},
		LockedSongs:              []entity.UserDataTransferLockedSong{},
		Goals: entity.UserDataTransferGoals{
			Groups:    []entity.UserDataTransferGoalGroup{},
			Ungrouped: []entity.UserDataTransferGoal{},
		},
		RecordFilters: []entity.UserDataTransferRecordFilter{},
	}
}

func encodeCodecTestFile(t *testing.T, codec usecase.UserDataTransferCodec) []byte {
	t.Helper()
	encoded, err := codec.Encode(codecTestSnapshot(t))
	require.NoError(t, err)
	return encoded
}

func mutateBase64URL(value string) string {
	replacement := byte('A')
	if value[len(value)-1] == replacement {
		replacement = 'B'
	}
	return value[:len(value)-1] + string(replacement)
}

func mutateProtectedJSON(t *testing.T, encoded []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var envelope testEnvelope
	require.NoError(t, json.Unmarshal(encoded, &envelope))
	headerBytes, err := base64.RawURLEncoding.DecodeString(envelope.Protected)
	require.NoError(t, err)
	var header map[string]any
	require.NoError(t, json.Unmarshal(headerBytes, &header))
	mutate(header)
	headerBytes, err = json.Marshal(header)
	require.NoError(t, err)
	envelope.Protected = base64.RawURLEncoding.EncodeToString(headerBytes)
	return signCodecTestEnvelope(t, envelope)
}

func mutatePayloadJSON(t *testing.T, encoded []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var envelope testEnvelope
	require.NoError(t, json.Unmarshal(encoded, &envelope))
	compressed, err := base64.RawURLEncoding.DecodeString(envelope.Payload)
	require.NoError(t, err)
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(reader).Decode(&payload))
	require.NoError(t, reader.Close())
	mutate(payload)
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)
	return replacePayloadBytes(t, encoded, gzipCodecTestBytes(t, payloadBytes))
}

func replacePayloadBytes(t *testing.T, encoded, payload []byte) []byte {
	t.Helper()
	var envelope testEnvelope
	require.NoError(t, json.Unmarshal(encoded, &envelope))
	envelope.Payload = base64.RawURLEncoding.EncodeToString(payload)
	return signCodecTestEnvelope(t, envelope)
}

func signCodecTestEnvelope(t *testing.T, envelope testEnvelope) []byte {
	t.Helper()
	mac := hmac.New(sha256.New, codecTestSecret)
	_, err := mac.Write([]byte(envelope.Protected + "." + envelope.Payload))
	require.NoError(t, err)
	envelope.Signature = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	encoded, err := json.Marshal(envelope)
	require.NoError(t, err)
	return encoded
}

func gzipCodecTestBytes(t *testing.T, value []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write(value)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return compressed.Bytes()
}

func assertErrorDoesNotLeak(t *testing.T, err error, values ...string) {
	t.Helper()
	for _, value := range values {
		assert.False(t, strings.Contains(err.Error(), value))
	}
}
