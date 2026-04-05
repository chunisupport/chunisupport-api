package handler

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v4"
)

var standardDifficultyNames = map[string]struct{}{
	"BASIC":    {},
	"ADVANCED": {},
	"EXPERT":   {},
	"MASTER":   {},
	"ULTIMA":   {},
}

// ParseDifficultyPath はパスパラメータを内部難易度名に変換します。
// 無効なパラメータの場合は空文字とfalseを返します。
// info.ParseDifficultyPathのラッパー関数です。
func ParseDifficultyPath(path string) (difficultyName string, ok bool) {
	return info.ParseDifficultyPath(path)
}

// BuildChartsMap creates a map of charts keyed by difficulty name.
// T is the type of the Chart DTO (e.g., *dto.ChartDTO or *dto.V1ChartDTO).
func BuildChartsMap[T any](
	charts []*entity.Chart,
	difficultyNames map[int]string,
	converter func(*entity.Chart) T,
) map[string]T {
	// Initialize map with nil for all difficulty levels
	chartsMap := make(map[string]T)
	for _, diffName := range difficultyNames {
		if _, ok := standardDifficultyNames[diffName]; !ok {
			continue
		}
		var zero T
		chartsMap[diffName] = zero
	}

	// Populate map with actual chart data
	for _, chart := range charts {
		if diffName, ok := difficultyNames[chart.DifficultyID]; ok {
			if _, isStandard := standardDifficultyNames[diffName]; !isStandard {
				continue
			}
			chartsMap[diffName] = converter(chart)
		}
	}

	return chartsMap
}

// GetRequesterAccountTypeID はコンテキストからログインユーザーのAccountTypeIDを取得します。
// ユーザーがログインしていない場合はnilを返します。
func GetRequesterAccountTypeID(c echo.Context) *int {
	userObj := c.Get("userEntity")
	if userObj == nil {
		return nil
	}

	user, ok := userObj.(*entity.User)
	if !ok {
		return nil
	}

	return &user.AccountTypeID
}
