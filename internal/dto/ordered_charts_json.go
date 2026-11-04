package dto

import (
	"bytes"
	"encoding/json"
)

// chartJSONKeyOrder は譜面オブジェクトのキー順です。
// encoding/json の map はキー順が不定なため、API出力ではこの順に固定します。
var chartJSONKeyOrder = [...]string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}

// MarshalOrderedCharts は譜面マップを難易度順の JSON オブジェクトへ変換します。
// 順序に含まれるキーは常に出力し、マップに無いキーと nil 値は null にします。
// 順序に無いキーは出力しません。値本体は encoding/json.Marshal のバイト列をそのまま連結し、
// エスケープや数値表現を個別の手動連結実装と揃えます。
func MarshalOrderedCharts[T any](charts map[string]*T) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range chartJSONKeyOrder {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('"')
		buf.WriteString(key)
		buf.WriteString(`":`)

		chart, exists := charts[key]
		if !exists || chart == nil {
			buf.WriteString("null")
			continue
		}

		chartJSON, err := json.Marshal(chart)
		if err != nil {
			return nil, err
		}
		buf.Write(chartJSON)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
