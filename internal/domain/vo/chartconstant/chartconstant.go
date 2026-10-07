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
// 定数なしは呼び出し側で nil として保持し、実在する譜面定数だけを受け付けます。
// 通常譜面の上限を超える値は許可しません。
func NewChartConstant(value float64) (ChartConstant, error) {
	if math.IsNaN(value) || value < constants.ChartConstMin || value > constants.ChartConstMax {
		return 0, fmt.Errorf("chart constant must be between %.1f and %.1f", constants.ChartConstMin, constants.ChartConstMax)
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

// Value は driver.Valuer を実装し、DECIMAL型との互換性を保つため文字列を返します。
func (c ChartConstant) Value() (driver.Value, error) {
	if _, err := NewChartConstant(c.Float64()); err != nil {
		return nil, err
	}
	return c.String(), nil
}

// Scan は sql.Scanner を実装し、DB値を検証済みの ChartConstant に復元します。
// DECIMAL値はドライバによって []byte または数値で返るため、両方を受け付けます。
func (c *ChartConstant) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("chart constant cannot be NULL")
	}

	switch v := value.(type) {
	case []byte:
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

// MarshalJSON は json.Marshaler を実装し、ChartConstant をJSON数値として出力します。
func (c ChartConstant) MarshalJSON() ([]byte, error) {
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
