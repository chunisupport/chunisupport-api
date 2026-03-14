package chartconstant

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// ChartConstant は譜面定数の値オブジェクトです。
// float64を基底型として使用し、精度の問題を回避します。
type ChartConstant float64

// NewChartConstant は新しい ChartConstant を生成します。
// 譜面定数は0以上である必要があります。
func NewChartConstant(value float64) (ChartConstant, error) {
	if value < 0 {
		return 0, errors.New("chart constant must be 0 or greater")
	}
	return ChartConstant(value), nil
}

// String は ChartConstant の文字列表現を返します。
func (c ChartConstant) String() string {
	return fmt.Sprintf("%.1f", c)
}

// Value は driver.Valuer インターフェースを実装します。
// データベースに値を保存する際に呼び出されます。
func (c ChartConstant) Value() (driver.Value, error) {
	// DECIMAL型との互換性のため、文字列として保存します。
	return c.String(), nil
}

// Scan は sql.Scanner インターフェースを実装します。
// データベースから値を読み取る際に呼び出されます。
func (c *ChartConstant) Scan(value any) error {
	if value == nil {
		chartConst, err := NewChartConstant(0)
		if err != nil {
			return err
		}
		*c = chartConst
		return nil
	}

	switch v := value.(type) {
	case []byte:
		// DECIMAL型は[]byteで返されることがあります。
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return fmt.Errorf("failed to convert []byte to float64: %w", err)
		}
		chartConst, err := NewChartConstant(f)
		if err != nil {
			return err
		}
		*c = chartConst
		return nil
	case float64:
		// 数値型として返される場合。
		chartConst, err := NewChartConstant(v)
		if err != nil {
			return err
		}
		*c = chartConst
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// MarshalJSON は json.Marshaler インターフェースを実装します。
// JSON出力時に小数点以下1桁の形式で出力されます。
func (c ChartConstant) MarshalJSON() ([]byte, error) {
	// 小数点以下1桁の数値としてマーシャルします。
	// String()メソッドの結果を使用して統一的な表現を保証します。
	return json.Marshal(float64(c))
}

// UnmarshalJSON は json.Unmarshaler インターフェースを実装します。
// JSON入力から ChartConstant を復元します。
func (c *ChartConstant) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	chartConst, err := NewChartConstant(f)
	if err != nil {
		return err
	}
	*c = chartConst
	return nil
}
