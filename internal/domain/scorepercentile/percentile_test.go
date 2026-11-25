package scorepercentile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildThresholdsAndLookup(t *testing.T) {
	t.Run("空のスコア配列はnilを返す", func(t *testing.T) {
		assert.Nil(t, BuildThresholds(nil))
		assert.Nil(t, BuildThresholds([]int{}))
	})

	t.Run("同点・理論値・中間スコア・最低点未満の新規スコアを正しく逆引きできる", func(t *testing.T) {
		// Given: 10人中 1,010,000 が2人、1,009,000 が3人、1,000,000 が5人
		scores := []int{
			1000000, 1009000, 1010000, 1000000, 1009000,
			1000000, 1010000, 1000000, 1009000, 1000000,
		}
		thresholds := BuildThresholds(scores)
		snapshot := NewSnapshot(1, []Entry{{TargetID: 10, Thresholds: thresholds}}, nil, nil)

		// When / Then
		tests := []struct {
			name     string
			score    uint32
			expected float64
		}{
			{name: "理論値（1位タイ: 1/10）", score: 1010000, expected: 10.00},
			{name: "スナップショットにない中間スコア（1,009,500: 上に2人→3位/10）", score: 1009500, expected: 30.00},
			{name: "1,009,000（3位タイ: 3/10）", score: 1009000, expected: 30.00},
			{name: "1,008,999（上に5人→6位/10）", score: 1008999, expected: 60.00},
			{name: "1,000,000（6位タイ: 6/10）", score: 1000000, expected: 60.00},
			{name: "最低点未満の新規スコアは100.00にクランプ", score: 950000, expected: 100.00},
			{name: "0点も100.00にクランプ", score: 0, expected: 100.00},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := snapshot.LookupChart(10, tt.score)
				require.NotNil(t, got)
				assert.Equal(t, tt.expected, *got)
			})
		}
		assert.Nil(t, snapshot.LookupChart(999, 1010000))
	})

	t.Run("母集団1人の譜面では全スコアが100.00になる", func(t *testing.T) {
		thresholds := BuildThresholds([]int{1005000})
		assert.Equal(t, []Threshold{{MinScore: 0, TopPercentBP: 10000}}, thresholds)
		snapshot := NewSnapshot(1, []Entry{{TargetID: 1, Thresholds: thresholds}}, nil, nil)
		got := snapshot.LookupChart(1, 1005000)
		require.NotNil(t, got)
		assert.Equal(t, 100.00, *got)
	})

	t.Run("母集団が多く1位が0.01%未満になっても0.01にクランプする", func(t *testing.T) {
		// Given: 25,000人（1位 = 1/25000 * 100 = 0.004% → 0.01% にクランプ）
		scores := make([]int, 25000)
		scores[0] = 1010000
		for i := 1; i < len(scores); i++ {
			scores[i] = 1000000
		}
		thresholds := BuildThresholds(scores)
		snapshot := NewSnapshot(1, nil, []Entry{{TargetID: 5, Thresholds: thresholds}}, nil)

		// When / Then
		top := snapshot.LookupWorldsendChart(5, 1010000)
		require.NotNil(t, top)
		assert.Equal(t, 0.01, *top)

		second := snapshot.LookupWorldsendChart(5, 1000000)
		require.NotNil(t, second)
		assert.Equal(t, 0.01, *second) // 2/25000 = 0.008% → 四捨五入で 0.01%
	})

	t.Run("コースの3030000点満点と0点記録を含む分布を正しく扱える", func(t *testing.T) {
		scores := []int{3030000, 2900000, 0}
		thresholds := BuildThresholds(scores)
		snapshot := NewSnapshot(1, nil, nil, []Entry{{TargetID: 7, Thresholds: thresholds}})

		top := snapshot.LookupCourse(7, 3030000)
		require.NotNil(t, top)
		assert.Equal(t, 33.33, *top)

		mid := snapshot.LookupCourse(7, 2900000)
		require.NotNil(t, mid)
		assert.Equal(t, 66.67, *mid)

		zero := snapshot.LookupCourse(7, 0)
		require.NotNil(t, zero)
		assert.Equal(t, 100.00, *zero)
	})
}

func TestEncodeDecodeThresholds(t *testing.T) {
	t.Run("正常な境界列をエンコード・デコードして復元できる", func(t *testing.T) {
		original := []Threshold{
			{MinScore: 0, TopPercentBP: 10000},
			{MinScore: 1000000, TopPercentBP: 6000},
			{MinScore: 1010000, TopPercentBP: 1000},
		}
		encoded := EncodeThresholds(original)
		decoded, err := DecodeThresholds(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("不正なバイナリはエラーになる", func(t *testing.T) {
		tests := []struct {
			name string
			data []byte
		}{
			{name: "空バイト列", data: nil},
			{name: "バージョン不一致", data: []byte{2, 0, 0, 0, 0, 1, 0}},
			{name: "長さが6の倍数でない", data: []byte{1, 0, 0, 0, 0, 1}},
			{name: "先頭のmin_scoreが0でない", data: []byte{1, 1, 0, 0, 0, 10, 0}},
			{name: "bpが0で範囲外", data: []byte{1, 0, 0, 0, 0, 0, 0}},
			{name: "bpが10001で範囲外", data: []byte{1, 0, 0, 0, 0, 0x11, 0x27}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := DecodeThresholds(tt.data)
				assert.ErrorIs(t, err, ErrInvalidThresholdData)
			})
		}
	})
}
