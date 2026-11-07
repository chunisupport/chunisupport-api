package dto

import (
	"encoding/json"
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type orderedChartSample struct {
	Name string `json:"name"`
}

type orderedChartMixed struct {
	Label string   `json:"label"`
	Bad   chan int `json:"bad,omitempty"`
}

// orderedChartsForTest は MarshalOrderedChartsTo を json.Marshal から呼び出すためのテスト用マップです。
type orderedChartsForTest[T any] map[string]*T

func (o orderedChartsForTest[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return MarshalOrderedChartsTo(enc, o)
}

func TestMarshalOrderedChartsTo(t *testing.T) {
	tests := []struct {
		name string
		// Given: 順不同で入れた譜面マップ
		charts map[string]*orderedChartSample
		// Then: 難易度順のJSON
		want string
	}{
		{
			name: "欠損とnilはnullになり未知キーは出力しない",
			charts: map[string]*orderedChartSample{
				"ULTIMA":  {Name: "u"},
				"BASIC":   nil,
				"UNKNOWN": {Name: "x"},
			},
			want: `{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":{"name":"u"}}`,
		},
		{
			name:   "nilマップは全難易度をnullにする",
			charts: nil,
			want:   `{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":null}`,
		},
		{
			name:   "空マップは全難易度をnullにする",
			charts: map[string]*orderedChartSample{},
			want:   `{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := json.Marshal(orderedChartsForTest[orderedChartSample](tt.charts))

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestMarshalOrderedChartsTo_エスケープはjsonMarshalと一致する(t *testing.T) {
	// Given: encoding/json がエスケープする文字を含む値と、順序外のキー
	value := &orderedChartSample{Name: "A \"B\" <C> & D\\\n\u2028譜面"}
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	charts := map[string]*orderedChartSample{
		"EXPERT": value,
		"NOPE":   value,
	}

	// When
	got, err := json.Marshal(orderedChartsForTest[orderedChartSample](charts))

	// Then
	require.NoError(t, err)
	want := `{"BASIC":null,"ADVANCED":null,"EXPERT":` + string(encoded) + `,"MASTER":null,"ULTIMA":null}`
	assert.Equal(t, want, string(got))
	assert.NotContains(t, string(got), "NOPE")
}

func TestMarshalOrderedChartsTo_値のエンコード失敗は部分結果を返さない(t *testing.T) {
	// Given: 先頭の難易度はエンコードでき、後続で失敗する
	charts := map[string]*orderedChartMixed{
		"BASIC":  {Label: "ok"},
		"MASTER": {Label: "ng", Bad: make(chan int)},
	}

	// When
	got, err := json.Marshal(orderedChartsForTest[orderedChartMixed](charts))

	// Then
	assert.Nil(t, got)
	assert.Error(t, err)
}
