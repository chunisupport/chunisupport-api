package displaylevel

import (
	"fmt"
	"math"
	"strconv"
)

const (
	BandWidthTenths          = 5
	FineConstThresholdTenths = 100
)

// Level は公式の表示レベルを表す値オブジェクトです。
// 整数レベルを n.0、プラスレベルを n.5 として保持します。
type Level struct {
	tenths int
}

// Parse は表示レベル表記（例: 13、13+）を値オブジェクトに変換します。
func Parse(value string) (Level, error) {
	if len(value) == 0 {
		return Level{}, fmt.Errorf("表示レベルの形式が不正です: %q", value)
	}

	isPlus := value[len(value)-1] == '+'
	wholeText := value
	if isPlus {
		wholeText = value[:len(value)-1]
	}
	if wholeText == "" || (len(wholeText) > 1 && wholeText[0] == '0') {
		return Level{}, fmt.Errorf("表示レベルの形式が不正です: %q", value)
	}
	for _, char := range wholeText {
		if char < '0' || char > '9' {
			return Level{}, fmt.Errorf("表示レベルの形式が不正です: %q", value)
		}
	}

	whole, err := strconv.Atoi(wholeText)
	if err != nil {
		return Level{}, fmt.Errorf("表示レベルの整数部が不正です: %q: %w", value, err)
	}
	if whole < 1 || whole > 16 || (isPlus && whole == 16) {
		return Level{}, fmt.Errorf("表示レベルが範囲外です: %q", value)
	}
	if isPlus {
		return Level{tenths: whole*10 + 5}, nil
	}
	return Level{tenths: whole * 10}, nil
}

// String は表示レベルを公式表記に変換します。
func (l Level) String() string {
	tenths := l.LowerTenths()
	whole := tenths / 10
	if tenths%10 == 0 {
		return strconv.Itoa(whole)
	}
	return strconv.Itoa(whole) + "+"
}

// IsValid はLevelがParseで生成できる正規の表示レベルかを返します。
func (l Level) IsValid() bool {
	return l.tenths >= 10 && l.tenths <= 160 && (l.tenths%10 == 0 || l.tenths%10 == 5)
}

// LowerTenths は表示レベル帯の下限を10倍整数で返します。
func (l Level) LowerTenths() int {
	tenths := l.tenths
	return tenths - tenths%BandWidthTenths
}

// UpperTenths は表示レベル帯の包含される上端を10倍整数で返します。
// 10未満のレベルは譜面定数の刻みがないため、下限と同じ値を返します。
func (l Level) UpperTenths() int {
	lower := l.LowerTenths()
	if lower < FineConstThresholdTenths {
		return lower
	}
	return lower + BandWidthTenths - 1
}

// FromFloat64 は既存の譜面定数を表示レベル帯として扱うための変換です。
// 公式表示レベルの入力には Parse を使用し、任意の小数定数を受け取るのは
// 既存の songbatch 帯判定を同じ丸め規則で維持するために限ります。
func FromFloat64(value float64) Level {
	return Level{tenths: int(math.Round(value * 10))}
}
