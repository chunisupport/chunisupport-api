package songbatch

import (
	"math"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
)

// DisplayLevelBandContains は chartConst が officialConst の属する公式表示レベルに収まるかを返します。
// 10以上は0.5幅で、公式の "13" は 13.0〜13.4、"13+" は 13.5〜13.9 です。
// 10未満は細かい定数がなく、"9" は 9.0、"9+" は 9.5 だけが譜面定数です。
// ワークスペースの const は表示レベルの下限（n.0 / n.5）として入ることが多いですが、
// 10以上で帯の途中の定数が残っていても、同じ表示レベル帯として比較します。
func DisplayLevelBandContains(officialConst, chartConst float64) bool {
	officialLevel := displaylevel.FromFloat64(officialConst)
	bandStart := officialLevel.LowerTenths()
	chartTenths := roundedConstTenths(chartConst)
	return chartTenths >= bandStart && chartTenths <= officialLevel.UpperTenths()
}

// FormatOfficialDisplayLevel は officialConst が属する公式表示レベルを "14" または "14+" で返します。
func FormatOfficialDisplayLevel(officialConst float64) string {
	return displaylevel.FromFloat64(officialConst).String()
}

func roundedConstTenths(constValue float64) int {
	return int(math.Round(constValue * 10))
}
