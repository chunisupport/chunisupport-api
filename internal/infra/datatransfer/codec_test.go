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
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var codecTestSecret = bytes.Repeat([]byte{0x42}, 32)

func TestCodecEncodeDecode(t *testing.T) {
	// Given: すべてのコレクションに2件ずつ入った有効なスナップショット
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	snapshot := codecTestSnapshot(t)
	// 入力ヘルパーが空に戻らないための固定点。各コレクションの件数を仕様の固定値で固める。
	require.Len(t, snapshot.Records, 2)
	require.Len(t, snapshot.RecordHistories, 2)
	require.Len(t, snapshot.WorldsendRecords, 2)
	require.Len(t, snapshot.WorldsendRecordHistories, 2)
	require.Len(t, snapshot.MetricHistories, 2)
	require.Len(t, snapshot.CourseRecords, 2)
	require.Len(t, snapshot.Honors, 2)
	require.Len(t, snapshot.FavoriteSongs, 2)
	require.Len(t, snapshot.LockedSongs, 2)
	require.Len(t, snapshot.Goals.Groups, 2)
	require.Len(t, snapshot.Goals.Ungrouped, 2)
	require.Len(t, snapshot.RecordFilters, 2)

	// When: 符号化して復号する
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
	// Then: スナップショット全体が往復で保たれる。入力の時刻はUTCの秒精度、配列は空ではなくnilでもないため正規化は不要である。
	assert.Equal(t, snapshot, decoded)
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

func TestCodecDecodeAcceptsVersion1WithoutOfficialOverpowerPercent(t *testing.T) {
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	encoded := encodeCodecTestFile(t, codec)
	withoutPercent := mutatePayloadJSON(t, encoded, func(payload map[string]any) {
		player := payload["player"].(map[string]any)
		delete(player, "official_overpower_percent")
		for _, rawHistory := range payload["metric_histories"].([]any) {
			delete(rawHistory.(map[string]any), "official_overpower_percent")
		}
	})
	version1 := mutateProtectedJSON(t, withoutPercent, func(header map[string]any) {
		header["schema_version"] = float64(1)
	})

	decoded, err := codec.Decode(version1)

	require.NoError(t, err)
	// Then: 指標履歴が2件あることを固定点とし、旧版では割合がnilになる。
	require.Len(t, decoded.MetricHistories, 2)
	assert.Nil(t, decoded.Player.OfficialOverpowerPercent)
	for _, history := range decoded.MetricHistories {
		assert.Nil(t, history.OfficialOverpowerPercent)
	}
}

func TestCodecDecodeDefaultsMissingPossessionNameToNormal(t *testing.T) {
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	encoded := encodeCodecTestFile(t, codec)
	withoutPossession := mutatePayloadJSON(t, encoded, func(payload map[string]any) {
		player := payload["player"].(map[string]any)
		delete(player, "possession_name")
	})

	tests := []struct {
		name          string
		schemaVersion float64
	}{
		{name: "スキーマバージョン1", schemaVersion: 1},
		{name: "スキーマバージョン2", schemaVersion: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versioned := mutateProtectedJSON(t, withoutPossession, func(header map[string]any) {
				header["schema_version"] = tt.schemaVersion
			})

			decoded, err := codec.Decode(versioned)

			require.NoError(t, err)
			require.NotNil(t, decoded.Player.PossessionName)
			assert.Equal(t, entity.PossessionNameNormal, *decoded.Player.PossessionName)
		})
	}
}

func TestCodecGoalGaugeStart(t *testing.T) {
	codec, err := NewCodec(codecTestSecret)
	require.NoError(t, err)
	snapshot := codecTestSnapshot(t)
	snapshot.Goals.Ungrouped = []entity.UserDataTransferGoal{{
		Title:                       "OP達成率99.8%",
		AchievementType:             "overpower_percent",
		AchievementParams:           json.RawMessage(`{"total":99.8}`),
		Attributes:                  json.RawMessage(`{}`),
		GaugeStart:                  99.6,
		ApplyGaugeStartToPercentage: true,
		SortOrder:                   1,
		CreatedAt:                   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}}
	encoded, err := codec.Encode(snapshot)
	require.NoError(t, err)

	t.Run("スタート位置と達成率への適用指定を往復で保持する", func(t *testing.T) {
		// When
		decoded, err := codec.Decode(encoded)

		// Then
		require.NoError(t, err)
		require.Len(t, decoded.Goals.Ungrouped, 1)
		assert.Equal(t, 99.6, decoded.Goals.Ungrouped[0].GaugeStart)
		assert.True(t, decoded.Goals.Ungrouped[0].ApplyGaugeStartToPercentage)
	})

	t.Run("項目追加前のデータは0とfalseとして読み込む", func(t *testing.T) {
		// Given
		withoutGaugeStart := mutatePayloadJSON(t, encoded, func(payload map[string]any) {
			goals := payload["goals"].(map[string]any)
			for _, rawGoal := range goals["ungrouped"].([]any) {
				goal := rawGoal.(map[string]any)
				delete(goal, "gauge_start")
				delete(goal, "apply_gauge_start_to_percentage")
			}
		})

		// When
		decoded, err := codec.Decode(withoutGaugeStart)

		// Then
		require.NoError(t, err)
		require.Len(t, decoded.Goals.Ungrouped, 1)
		assert.Equal(t, 0.0, decoded.Goals.Ungrouped[0].GaugeStart)
		assert.False(t, decoded.Goals.Ungrouped[0].ApplyGaugeStartToPercentage)
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

// codecTestSnapshot は往復テスト用の有効なスナップショットを組み立てます。
// すべてのコレクションに異なる値の2件を入れ、枠順位・指標履歴の割合・称号画像は値ありと値なしの両方を含めます。
// 時刻はUTCの秒精度に揃えているため、符号化・復号の前後で正規化せずに比較できます。
func codecTestSnapshot(t *testing.T) *entity.UserDataTransferSnapshot {
	t.Helper()
	name, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)
	// プレイヤー単体のnull可能項目は、値ありと値なしを項目ごとに分けて両方を含める。
	// 所有名はnilだとnormalに補正されるため、往復比較では値ありに固定する。
	officialOverpowerPercent := 98.75
	classEmblemName := "inf"
	possessionName := "rainbow"
	lastPlayedAt := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	dataCollectedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	scoreBest, err := score.NewScore(1_009_000)
	require.NoError(t, err)
	scoreNone, err := score.NewScore(850_000)
	require.NoError(t, err)
	scoreHistoryA, err := score.NewScore(1_000_000)
	require.NoError(t, err)
	scoreHistoryB, err := score.NewScore(950_000)
	require.NoError(t, err)
	scoreWorldsend, err := score.NewScore(1_009_000)
	require.NoError(t, err)
	scoreWorldsendLow, err := score.NewScore(700_000)
	require.NoError(t, err)
	scoreWorldsendHistory, err := score.NewScore(1_000_000)
	require.NoError(t, err)
	scoreWorldsendHistoryLow, err := score.NewScore(600_000)
	require.NoError(t, err)
	courseScoreClear, err := coursescore.New(3_000_000)
	require.NoError(t, err)
	courseScoreUnclear, err := coursescore.New(2_500_000)
	require.NoError(t, err)
	groupNameA, err := goalgroupname.NewGoalGroupName("今月")
	require.NoError(t, err)
	groupNameB, err := goalgroupname.NewGoalGroupName("来月")
	require.NoError(t, err)
	honorImageURL := "https://example.com/honor.png"
	metricPercent := 98.0
	bestOrder := 1
	return &entity.UserDataTransferSnapshot{
		Player: entity.UserDataTransferPlayer{
			Name:                     name,
			Level:                    100,
			OfficialRating:           17.25,
			OfficialOverpower:        12_345.67,
			OfficialOverpowerPercent: &officialOverpowerPercent,
			ClassEmblemName:          &classEmblemName,
			ClassEmblemBaseName:      nil,
			PossessionName:           &possessionName,
			LastPlayedAt:             &lastPlayedAt,
			DataCollectedAt:          &dataCollectedAt,
			CreatedAt:                time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		Records: []entity.UserDataTransferRecord{
			{
				SongOfficialIdx: "1001",
				Difficulty:      "MASTER",
				Score:           scoreBest,
				ClearLampName:   "CLEAR",
				ComboLampName:   "FULL COMBO",
				FullChainName:   "NONE",
				SlotName:        "best",
				SlotOrder:       &bestOrder,
				UpdatedAt:       time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
			},
			{
				SongOfficialIdx: "1002",
				Difficulty:      "ULTIMA",
				Score:           scoreNone,
				ClearLampName:   "FAILED",
				ComboLampName:   "NONE",
				FullChainName:   "NONE",
				SlotName:        "none",
				SlotOrder:       nil,
				UpdatedAt:       time.Date(2026, 8, 11, 1, 2, 3, 0, time.UTC),
			},
		},
		RecordHistories: []entity.UserDataTransferRecordHistory{
			{
				SongOfficialIdx: "1001",
				Difficulty:      "MASTER",
				Score:           scoreHistoryA,
				ClearLampName:   "CLEAR",
				ComboLampName:   "FULL COMBO",
				FullChainName:   "FULL CHAIN GOLD",
				UpdatedAt:       time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
			},
			{
				SongOfficialIdx: "1002",
				Difficulty:      "EXPERT",
				Score:           scoreHistoryB,
				ClearLampName:   "FAILED",
				ComboLampName:   "NONE",
				FullChainName:   "NONE",
				UpdatedAt:       time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC),
			},
		},
		WorldsendRecords: []entity.UserDataTransferWorldsendRecord{
			{
				SongOfficialIdx: "2001",
				Score:           scoreWorldsend,
				ClearLampName:   "CLEAR",
				ComboLampName:   "FULL COMBO",
				FullChainName:   "NONE",
				UpdatedAt:       time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
			},
			{
				SongOfficialIdx: "2002",
				Score:           scoreWorldsendLow,
				ClearLampName:   "FAILED",
				ComboLampName:   "NONE",
				FullChainName:   "NONE",
				UpdatedAt:       time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
			},
		},
		WorldsendRecordHistories: []entity.UserDataTransferWorldsendRecordHistory{
			{
				SongOfficialIdx: "2001",
				Score:           scoreWorldsendHistory,
				ClearLampName:   "CLEAR",
				ComboLampName:   "FULL COMBO",
				FullChainName:   "FULL CHAIN GOLD",
				UpdatedAt:       time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
			},
			{
				SongOfficialIdx: "2002",
				Score:           scoreWorldsendHistoryLow,
				ClearLampName:   "FAILED",
				ComboLampName:   "NONE",
				FullChainName:   "NONE",
				UpdatedAt:       time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC),
			},
		},
		MetricHistories: []entity.UserDataTransferMetricHistory{
			{
				OfficialRating:           17.00,
				OfficialOverpower:        12_000.00,
				OfficialOverpowerPercent: &metricPercent,
				DataCollectedAt:          time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
			},
			{
				OfficialRating:           17.10,
				OfficialOverpower:        12_100.50,
				OfficialOverpowerPercent: nil,
				DataCollectedAt:          time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
			},
		},
		CourseRecords: []entity.UserDataTransferCourseRecord{
			{
				CourseOfficialIdx: "course-1",
				Score:             courseScoreClear,
				IsClear:           true,
				ComboLampName:     "ALL JUSTICE",
				UpdatedAt:         time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
			},
			{
				CourseOfficialIdx: "course-2",
				Score:             courseScoreUnclear,
				IsClear:           false,
				ComboLampName:     "FULL COMBO",
				UpdatedAt:         time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
			},
		},
		Honors: []entity.UserDataTransferHonor{
			{
				Slot:       1,
				ImageURL:   &honorImageURL,
				Name:       "虹色称号",
				TypeName:   "rainbow",
				EquippedAt: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
			},
			{
				Slot:       2,
				ImageURL:   nil,
				Name:       "金称号",
				TypeName:   "gold",
				EquippedAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
			},
		},
		FavoriteSongs: []entity.UserDataTransferFavoriteSong{
			{
				SongOfficialIdx: "1001",
				FavoritedAt:     time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
			},
			{
				SongOfficialIdx: "1003",
				FavoritedAt:     time.Date(2026, 8, 10, 1, 2, 3, 0, time.UTC),
			},
		},
		LockedSongs: []entity.UserDataTransferLockedSong{
			{
				SongOfficialIdx: "1002",
				IsUltima:        false,
			},
			{
				SongOfficialIdx: "1004",
				IsUltima:        true,
			},
		},
		Goals: entity.UserDataTransferGoals{
			Groups: []entity.UserDataTransferGoalGroup{
				{
					Name:      groupNameA,
					SortOrder: 1,
					CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
					Goals: []entity.UserDataTransferGoal{
						{
							Title:             "SSSを取る",
							AchievementType:   "rank_count",
							AchievementParams: json.RawMessage(`{"count":1}`),
							Attributes:        json.RawMessage(`{"diff":["MASTER"]}`),
							SortOrder:         1,
							CreatedAt:         time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
						},
						{
							Title:                       "OP達成率99.8%",
							AchievementType:             "overpower_percent",
							AchievementParams:           json.RawMessage(`{"total":99.8}`),
							Attributes:                  json.RawMessage(`{}`),
							InvertValue:                 true,
							InvertPercentage:            true,
							GaugeStart:                  99.6,
							ApplyGaugeStartToPercentage: true,
							SortOrder:                   2,
							CreatedAt:                   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
						},
					},
				},
				{
					Name:      groupNameB,
					SortOrder: 2,
					CreatedAt: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
					Goals: []entity.UserDataTransferGoal{
						{
							Title:             "皆伝を取る",
							AchievementType:   "rank_count",
							AchievementParams: json.RawMessage(`{"count":5}`),
							Attributes:        json.RawMessage(`{"diff":["ULTIMA"]}`),
							SortOrder:         1,
							CreatedAt:         time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
						},
					},
				},
			},
			Ungrouped: []entity.UserDataTransferGoal{
				{
					Title:             "ULTIMAを埋める",
					AchievementType:   "play_count",
					AchievementParams: json.RawMessage(`{"count":10}`),
					Attributes:        json.RawMessage(`{"diff":["ULTIMA"]}`),
					SortOrder:         1,
					CreatedAt:         time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC),
				},
				{
					Title:             "高難度を攻略する",
					AchievementType:   "rank_count",
					AchievementParams: json.RawMessage(`{"count":3}`),
					Attributes:        json.RawMessage(`{"diff":["MASTER","ULTIMA"]}`),
					InvertValue:       true,
					GaugeStart:        10.5,
					SortOrder:         2,
					CreatedAt:         time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		RecordFilters: []entity.UserDataTransferRecordFilter{
			{
				Name:          "高難度",
				FilterType:    "standard",
				SchemaVersion: 1,
				Filter:        json.RawMessage(`{"difficulties":["MASTER"]}`),
				CreatedAt:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:     time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
			},
			{
				Name:          "WORLD'S END一覧",
				FilterType:    "worldsend",
				SchemaVersion: 2,
				Filter:        json.RawMessage(`{}`),
				CreatedAt:     time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
				UpdatedAt:     time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
			},
		},
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
	if value[0] == replacement {
		replacement = 'B'
	}
	return string(replacement) + value[1:]
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
