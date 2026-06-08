package dto

import "github.com/chunisupport/chunisupport-api/internal/domain/service"

// OpTargetDifficultyPtr は難易度IDからOP対象難易度名のポインタを返します。
// 譜面が存在しない（ID=0）または未知の難易度IDの場合は nil を返します。
func OpTargetDifficultyPtr(difficultyID int) *string {
	if difficultyID == 0 {
		return nil
	}

	name, ok := service.DifficultyNameByID(difficultyID)
	if !ok {
		return nil
	}

	return &name
}
