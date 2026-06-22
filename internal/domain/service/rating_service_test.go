package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// floatEquals は浮動小数点数の比較を行います（許容誤差: 1e-9）
func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestCalcSingleRating(t *testing.T) {
	tests := []struct {
		name       string
		score      uint32
		chartConst float64
		want       float64
	}{
		// SSS+ ボーダー
		{"SSS+ 理論値", 1010000, 15.0, 17.15},
		{"SSS+ ボーダー", 1009000, 15.0, 17.15},
		{"SSS 上限", 1008999, 15.0, 17.14},

		// SSS ボーダー
		{"SSS ボーダー", 1007500, 15.0, 17.0},
		{"SS+ 上限", 1007499, 15.0, 16.99},

		// SS+ ボーダー
		{"SS+ ボーダー", 1005000, 15.0, 16.5},
		{"SS 上限", 1004999, 15.0, 16.49},

		// SS ボーダー
		{"SS ボーダー", 1000000, 15.0, 16.0},
		{"S+ 上限", 999999, 15.0, 15.99},

		// S+ ボーダー
		{"S+ ボーダー", 990000, 15.0, 15.6},
		{"S 上限", 989999, 15.0, 15.59},

		// S ボーダー
		{"S ボーダー", 975000, 15.0, 15.0},
		{"AAA 上限", 974999, 15.0, 14.99},

		// AAA ボーダー
		{"AAA ボーダー", 950000, 15.0, 13.33},
		{"AA 上限", 949999, 15.0, 13.32},

		// AA ボーダー
		{"AA ボーダー", 925000, 15.0, 11.66},
		{"A 上限", 924999, 15.0, 11.66},

		// A ボーダー
		{"A ボーダー", 900000, 15.0, 10.0},

		// BBB ボーダー
		{"BBB ボーダー", 800000, 15.0, 5.0},

		// C ボーダー
		{"C ボーダー", 500000, 15.0, 0.0},

		// D（0未満は0）
		{"D ランク", 400000, 15.0, 0.0},

		// 低定数での確認
		{"低定数 SSS+", 1009000, 10.0, 12.15},
		{"低定数 S", 975000, 10.0, 10.0},
		{"低定数 A", 900000, 10.0, 5.0},

		// 定数5以下の場合（BBB以下で特殊処理）
		{"定数5 A", 900000, 5.0, 0.0},
		{"定数4 BBB", 800000, 4.0, 0.0},

		// その他
		{"その他1", 1009067, 13.4, 15.55},
		{"その他2", 1009690, 14.2, 16.35},
		{"その他3", 1009255, 14.4, 16.55},
		{"その他4", 1009944, 14.4, 16.55},
		{"その他5", 1006800, 15.2, 17.06},

		// 実データから計算された検証データ
		{"実データ1", 1008280, 15.7, 17.77},
		{"実データ2", 1009020, 15.6, 17.75},
		{"実データ3", 1007906, 15.7, 17.74},
		{"実データ4", 1007862, 15.7, 17.73},
		{"実データ5", 1007685, 15.7, 17.71},
		{"実データ6", 1008596, 15.6, 17.7},
		{"実データ7", 1008136, 15.6, 17.66},
		{"実データ8", 1009247, 15.5, 17.65},
		{"実データ9", 1008031, 15.6, 17.65},
		{"実データ10", 1007845, 15.6, 17.63},
		{"実データ11", 1007415, 15.6, 17.58},
		{"実データ12", 1008103, 15.5, 17.56},
		{"実データ13", 1008149, 15.5, 17.56},
		{"実データ14", 1008084, 15.5, 17.55},
		{"実データ15", 1008966, 15.4, 17.54},
		{"実データ16", 1007907, 15.5, 17.54},
		{"実データ17", 1007831, 15.5, 17.53},
		{"実データ18", 1007610, 15.5, 17.51},
		{"実データ19", 1007542, 15.7, 17.70},
		{"実データ20", 1007522, 15.6, 17.60},
		{"実データ21", 1008472, 15.4, 17.49},
		{"実データ22", 1008415, 15.4, 17.49},
		{"実データ23", 1008063, 15.4, 17.45},
		{"実データ24", 1009011, 15.3, 17.45},
		{"実データ25", 1009495, 15.3, 17.45},
		{"実データ26", 1009133, 15.3, 17.45},
		{"実データ27", 1009065, 15.3, 17.45},
		{"実データ28", 1009339, 15.3, 17.45},
		{"実データ29", 1008080, 15.4, 17.45},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcSingleRating(tt.score, tt.chartConst)
			if !floatEquals(got, tt.want) {
				assert.Failf(t, "アサーション失敗", "CalcSingleRating(%d, %.1f) = %.6f, want %.6f", tt.score, tt.chartConst, got, tt.want)
			}
		})
	}
}

func TestCalcRatingStats(t *testing.T) {
	tests := []struct {
		name        string
		records     []RatingRecord
		wantPlayer  float64
		wantBestAvg float64
		wantNewAvg  float64
	}{
		{
			name:        "レコードなし",
			records:     []RatingRecord{},
			wantPlayer:  0.0,
			wantBestAvg: 0.0,
			wantNewAvg:  0.0,
		},
		{
			name: "ベスト枠のみ30曲",
			records: func() []RatingRecord {
				records := make([]RatingRecord, 30)
				for i := 0; i < 30; i++ {
					records[i] = RatingRecord{
						Score:      1009000,
						ChartConst: 15.0 - float64(i)*0.1,
						IsNew:      false,
					}
				}
				return records
			}(),
			// ベスト枠30曲の合計: 17.15 + 17.05 + ... + 14.25 = 471.0
			// ベスト平均: 471.0 / 30 = 15.7
			// 新曲枠0曲の合計: 0
			// 新曲平均: 0
			// プレイヤーレーティング: (471.0 + 0) / 50 = 9.42
			wantPlayer:  9.42,
			wantBestAvg: 15.7,
			wantNewAvg:  0.0,
		},
		{
			name: "新曲枠のみ20曲",
			records: func() []RatingRecord {
				records := make([]RatingRecord, 20)
				for i := 0; i < 20; i++ {
					records[i] = RatingRecord{
						Score:      1009000,
						ChartConst: 15.0 - float64(i)*0.1,
						IsNew:      true,
					}
				}
				return records
			}(),
			// ベスト枠: 新曲20曲すべてが含まれる (count=20)
			// ベスト枠合計: 17.15 + 17.05 + ... + 15.25 = 324.0
			// ベスト平均: 324.0 / 20 = 16.2
			// 新曲枠20曲合計: 324.0
			// 新曲平均: 16.2
			// プレイヤーレーティング: (324.0 + 324.0) / 50 = 648.0 / 50 = 12.96
			wantPlayer:  12.96,
			wantBestAvg: 16.2,
			wantNewAvg:  16.2,
		},
		{
			name: "1曲のみ",
			records: []RatingRecord{
				{Score: 1009000, ChartConst: 15.0, IsNew: false},
			},
			wantPlayer:  0.343,
			wantBestAvg: 17.15,
			wantNewAvg:  0.0,
		},
		{
			name: "ベスト枠30曲未満かつ新曲10曲",
			records: []RatingRecord{
				{Score: 1009000, ChartConst: 15.0, IsNew: true},
				{Score: 1009000, ChartConst: 14.9, IsNew: true},
				{Score: 1009000, ChartConst: 14.8, IsNew: true},
				{Score: 1009000, ChartConst: 14.7, IsNew: true},
				{Score: 1009000, ChartConst: 14.6, IsNew: true},
				{Score: 1009000, ChartConst: 14.5, IsNew: true},
				{Score: 1009000, ChartConst: 14.4, IsNew: true},
				{Score: 1009000, ChartConst: 14.3, IsNew: true},
				{Score: 1009000, ChartConst: 14.2, IsNew: true},
				{Score: 1009000, ChartConst: 14.1, IsNew: true},
				{Score: 1009000, ChartConst: 10.0, IsNew: false},
				{Score: 1009000, ChartConst: 10.0, IsNew: false},
			},
			wantPlayer:  7.166,
			wantBestAvg: 15.9416,
			wantNewAvg:  16.7,
		},
		{
			name: "ベスト枠30曲と新曲枠20曲が混在",
			records: func() []RatingRecord {
				records := make([]RatingRecord, 50)
				for i := 0; i < 30; i++ {
					records[i] = RatingRecord{
						Score:      1009000,
						ChartConst: 15.0 - float64(i)*0.1,
						IsNew:      false,
					}
				}
				for i := 0; i < 20; i++ {
					records[30+i] = RatingRecord{
						Score:      1009000,
						ChartConst: 14.0 - float64(i)*0.1,
						IsNew:      true,
					}
				}
				return records
			}(),
			wantPlayer:  15.7,
			wantBestAvg: 16.0333,
			wantNewAvg:  15.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcRatingStats(tt.records)
			if !floatEquals(got.PlayerRating, tt.wantPlayer) {
				assert.Failf(t, "アサーション失敗", "PlayerRating = %.6f, want %.6f", got.PlayerRating, tt.wantPlayer)
			}
			if !floatEquals(got.BestAverage, tt.wantBestAvg) {
				assert.Failf(t, "アサーション失敗", "BestAverage = %.6f, want %.6f", got.BestAverage, tt.wantBestAvg)
			}
			if !floatEquals(got.NewAverage, tt.wantNewAvg) {
				assert.Failf(t, "アサーション失敗", "NewAverage = %.6f, want %.6f", got.NewAverage, tt.wantNewAvg)
			}
		})
	}
}

func TestCalcSingleOverpower(t *testing.T) {
	tests := []struct {
		name        string
		score       uint32
		chartConst  float64
		comboLampID int
		want        float64
	}{
		// AJC（理論値）: (15+2)*5 + (1010000-1007500)/2500*3.75 + 1.25 = 85 + 3.75 + 1.25 = 90.0
		// スコア==1010000の場合は+1.25が加算される（仕様: (譜面定数+3)×5）
		{"AJC 理論値", 1010000, 15.0, 3, 90.0},

		// SSS以上 + AJ: (15+2)*5 + (1009000-1007500)/2500*3.75 + 1.0 = 85 + 2.25 + 1.0 = 88.25
		{"SSS+ AJ", 1009000, 15.0, 3, 88.25},
		{"SSS AJ", 1007500, 15.0, 3, 86.0}, // 85 + 0 + 1.0 = 86

		// SSS以上 + FC
		{"SSS+ FC", 1009000, 15.0, 2, 87.75}, // 85 + 2.25 + 0.5 = 87.75
		{"SSS FC", 1007500, 15.0, 2, 85.5},   // 85 + 0 + 0.5 = 85.5

		// SSS以上 + ノーコンボ
		{"SSS+ NONE", 1009000, 15.0, 1, 87.25}, // 85 + 2.25 + 0 = 87.25
		{"SSS NONE", 1007500, 15.0, 1, 85.0},   // 85 + 0 + 0 = 85

		// S以上 + 各種コンボ
		{"S AJ", 975000, 15.0, 3, 76.0},   // 15*5 + 1.0 = 76
		{"S FC", 975000, 15.0, 2, 75.5},   // 15*5 + 0.5 = 75.5
		{"S NONE", 975000, 15.0, 1, 75.0}, // 15*5 = 75

		// SS
		{"SS AJ", 1000000, 15.0, 3, 81.0},   // (15+1)*5 + 1.0 = 81
		{"SS FC", 1000000, 15.0, 2, 80.5},   // (15+1)*5 + 0.5 = 80.5
		{"SS NONE", 1000000, 15.0, 1, 80.0}, // (15+1)*5 = 80

		// A～AAA（S未満なので0.05単位）
		// AAA: (15-5)*5 + (950000-900000)/75000*25 = 50 + 16.666... = 66.666... → 0.05単位で66.65
		{"AAA NONE", 950000, 15.0, 1, 66.65},
		{"A NONE", 900000, 15.0, 1, 50.0}, // (15-5)*5 = 50

		// 低スコア
		{"C NONE", 500000, 15.0, 1, 0.0},
		{"D NONE", 400000, 15.0, 1, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcSingleOverpower(tt.score, tt.chartConst, tt.comboLampID)
			if !floatEquals(got, tt.want) {
				assert.Failf(t, "アサーション失敗", "CalcSingleOverpower(%d, %.1f, %d) = %.6f, want %.6f", tt.score, tt.chartConst, tt.comboLampID, got, tt.want)
			}
		})
	}
}

func TestCalcSingleOverpower_WikiExample(t *testing.T) {
	// Wikiの例: 譜面定数15.4、スコア1,009,540、AJ
	// (15.4+2)×5+1.0+(1,009,540-1,007,500)×0.0015 = 87 + 1.0 + 3.06 = 91.06
	score := uint32(1009540)
	chartConst := 15.4
	comboLampID := 3 // AJ

	got := CalcSingleOverpower(score, chartConst, comboLampID)
	want := 91.06

	if !floatEquals(got, want) {
		assert.Failf(t, "アサーション失敗", "Wiki example: CalcSingleOverpower(%d, %.1f, %d) = %.6f, want %.6f", score, chartConst, comboLampID, got, want)
	}
}

func TestCalcSongMaxOP(t *testing.T) {
	tests := []struct {
		name          string
		maxChartConst float64
		want          float64
	}{
		{
			name:          "定数0の場合",
			maxChartConst: 0,
			want:          0,
		},
		{
			name:          "最大定数15.0で理論値を計算",
			maxChartConst: 15.0,
			want:          90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcSongMaxOP(tt.maxChartConst)
			if !floatEquals(got, tt.want) {
				assert.Failf(t, "アサーション失敗", "CalcSongMaxOP() = %.6f, want %.6f", got, tt.want)
			}
		})
	}
}

func TestCalcSingleOverpowerPercent(t *testing.T) {
	tests := []struct {
		name        string
		score       uint32
		chartConst  float64
		comboLampID int
		expected    float64
	}{
		{
			name:        "理論値の場合100%になる",
			score:       1010000,
			chartConst:  14.0,
			comboLampID: 3,
			expected:    100.0,
		},
		{
			name:        "譜面別理論値に対する割合を小数点以下4桁で返す",
			score:       1009000,
			chartConst:  14.0,
			comboLampID: 3,
			expected:    97.9412,
		},
		{
			name:        "譜面定数が0の場合0%になる",
			score:       1009000,
			chartConst:  0,
			comboLampID: 3,
			expected:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			actual := CalcSingleOverpowerPercent(tt.score, tt.chartConst, tt.comboLampID)

			// Then
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestRoundN(t *testing.T) {
	tests := []struct {
		num  float64
		n    int
		want float64
	}{
		{1.234, 2, 1.23},
		{1.235, 2, 1.24},
		{1.2345, 3, 1.235},
		{1.2344, 3, 1.234},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := roundN(tt.num, tt.n)
			if !floatEquals(got, tt.want) {
				assert.Failf(t, "アサーション失敗", "roundN(%.4f, %d) = %.6f, want %.6f", tt.num, tt.n, got, tt.want)
			}
		})
	}
}
