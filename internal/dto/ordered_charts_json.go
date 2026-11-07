package dto

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
)

// chartJSONKeyOrder は譜面オブジェクトのキー順です。
// encoding/json の map はキー順が不定なため、API出力ではこの順に固定します。
var chartJSONKeyOrder = [...]string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}

// MarshalOrderedChartsTo は譜面マップを難易度順の JSON オブジェクトとして enc へ書き込みます。
// 順序に含まれるキーは常に出力し、マップに無いキーと nil 値は null にします。順序に無いキーは出力しません。
// API出力時の時刻のタイムゾーン変換など呼び出し元のエンコードオプションを各譜面にも適用するため、
// 別のエンコードを始める MarshalJSON ではなく、MarshalJSONTo から渡されたエンコーダへ直接書き込みます。
func MarshalOrderedChartsTo[T any](enc *jsontext.Encoder, charts map[string]*T) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, key := range chartJSONKeyOrder {
		if err := enc.WriteToken(jsontext.String(key)); err != nil {
			return err
		}
		if err := jsonv2.MarshalEncode(enc, charts[key]); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}
