package chartconstant

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
)

// ChartConstant は譜面定数の値オブジェクトです。
// 計算時は Tenths で0.1単位の整数へ変換して使用します。
type ChartConstant float64

// NewChartConstant は新しい ChartConstant を生成します。
// 譜面定数は0以上である必要があります。
// 通常譜面の上限を超える値は許可しません。
func NewChartConstant(value float64) (ChartConstant, error) {
	if value < constants.ChartConstValueMin || value > constants.ChartConstMax {
		return 0, fmt.Errorf("chart constant must be between %.1f and %.1f", constants.ChartConstValueMin, constants.ChartConstMax)
	}

	tenths := math.Round(value * 10)
	if math.Abs(value*10-tenths) > 1e-9 {
		return 0, fmt.Errorf("chart constant must have at most one decimal place: %v", value)
	}

	return ChartConstant(tenths / 10), nil
}

// Float64 はAPI・DB境界で利用する小数表現を返します。
func (c ChartConstant) Float64() float64 {
	return float64(c)
}

// String は ChartConstant の文字列表現を返します。
func (c ChartConstant) String() string {
	return fmt.Sprintf("%.1f", c.Float64())
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
	case int64:
		chartConst, err := NewChartConstant(float64(v))
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
	return json.Marshal(c.Float64())
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
