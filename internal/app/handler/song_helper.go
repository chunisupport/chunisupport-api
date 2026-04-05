package handler

import (
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v4"
)

var normalDifficultyNameSet = map[string]struct{}{
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
	// Initialize map with zero value for all normal difficulty levels in master data.
	chartsMap := make(map[string]T)
	for _, diffName := range difficultyNames {
		normalizedName := strings.ToUpper(diffName)
		if _, ok := normalDifficultyNameSet[normalizedName]; !ok {
			continue
		}
		var zero T
		chartsMap[normalizedName] = zero
	}

	// Populate map with actual chart data.
	for _, chart := range charts {
		diffName, ok := difficultyNames[chart.DifficultyID]
		if !ok {
			continue
		}
		normalizedName := strings.ToUpper(diffName)
		if _, ok := normalDifficultyNameSet[normalizedName]; !ok {
			continue
		}
		chartsMap[normalizedName] = converter(chart)
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
