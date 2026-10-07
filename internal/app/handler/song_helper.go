package handler

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainservice "github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
)

const (
	MinDifficultyID = domainservice.DifficultyIDBasic
	MaxDifficultyID = domainservice.DifficultyIDUltima
)

// ParseDifficultyPath はパスパラメータを内部難易度名に変換します。
// 無効なパラメータの場合は空文字とfalseを返します。
// info.ParseDifficultyPathのラッパー関数です。
func ParseDifficultyPath(path string) (difficultyName string, ok bool) {
	return info.ParseDifficultyPath(path)
}

// BuildChartsMap は難易度名をキーとする譜面マップを生成します。
// Tには譜面DTOの型を指定します。存在しない難易度の値はTのゼロ値です。
func BuildChartsMap[T any](
	charts []*entity.Chart,
	difficultyNames map[int]string,
	converter func(*entity.Chart) T,
) map[string]T {
	chartsMap := make(map[string]T)
	for diffID, diffName := range difficultyNames {
		if diffID >= MinDifficultyID && diffID <= MaxDifficultyID {
			var zero T
			chartsMap[diffName] = zero
		}
	}

	for _, chart := range charts {
		if diffName, ok := difficultyNames[chart.DifficultyID]; ok {
			chartsMap[diffName] = converter(chart)
		}
	}

	return chartsMap
}

// GetRequesterAccountTypeID はコンテキストからログインユーザーのAccountTypeIDを取得します。
// ユーザーがログインしていない場合はnilを返します。
func GetRequesterAccountTypeID(c *echo.Context) *int {
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
