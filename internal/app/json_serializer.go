package app

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"time"

	"github.com/labstack/echo/v5"
)

// TimezoneJSONSerializer はtime.TimeだけをAPI出力用タイムゾーンへ変換します。
// ドメイン・ユースケース・DBのUTC時刻を変更しないよう、値は書き換えずにエンコード時だけ変換します。
type TimezoneJSONSerializer struct {
	options jsonv2.Options
}

// NewTimezoneJSONSerializer は指定ロケーションを使用するJSONシリアライザを生成します。
// レスポンス全体を複製せずに時刻だけを変換するため、encoding/json/v2 の型別マーシャラを使います。
// タグ解釈（omitempty など）やエスケープを encoding/json と揃えるため、v1 互換オプションと組み合わせます。
func NewTimezoneJSONSerializer(location *time.Location) *TimezoneJSONSerializer {
	if location == nil {
		location = time.UTC
	}
	marshalTime := jsonv2.MarshalToFunc(func(enc *jsontext.Encoder, value time.Time) error {
		// time.Time.MarshalJSON と同じRFC3339表現を、エンコーダのバッファへ直接書き込んで確保を避けます。
		encoded, err := value.In(location).AppendText(append(enc.AvailableBuffer(), '"'))
		if err != nil {
			return err
		}
		return enc.WriteValue(append(encoded, '"'))
	})
	return &TimezoneJSONSerializer{
		options: jsonv2.JoinOptions(json.DefaultOptionsV1(), jsonv2.WithMarshalers(marshalTime)),
	}
}

// Serialize は時刻を出力タイムゾーンへ変換しながらレスポンス値をエンコードします。
// 独自の MarshalJSON は内部で別のエンコードを始めるため、その内側の時刻には変換が及びません。
// 時刻を含む値を独自にエンコードする型は、渡されたエンコーダへ書き込む MarshalJSONTo を実装してください。
func (s *TimezoneJSONSerializer) Serialize(c *echo.Context, target any, indent string) error {
	body, err := jsonv2.Marshal(target, s.options)
	if err != nil {
		return err
	}
	if indent != "" {
		// json.Encoder.SetIndent と同じ整形結果にするため、エンコード後に json.Indent で整形します。
		var buffer bytes.Buffer
		if err := json.Indent(&buffer, body, "", indent); err != nil {
			return err
		}
		body = buffer.Bytes()
	}
	// json.Encoder.Encode と同じく末尾に改行を付けます。
	_, err = c.Response().Write(append(body, '\n'))
	return err
}

// Deserialize は入力値を変換せず、Echo標準のデシリアライズを使用します。
func (s *TimezoneJSONSerializer) Deserialize(c *echo.Context, target any) error {
	return (echo.DefaultJSONSerializer{}).Deserialize(c, target)
}
