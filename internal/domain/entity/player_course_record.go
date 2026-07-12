package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
)

// PlayerCourseRecord はプレイヤーのコース記録です。
type PlayerCourseRecord struct {
	PlayerID   int
	CourseID   int
	Score      coursescore.CourseScore
	IsClear    bool
	ComboLampID int
	UpdatedAt  time.Time
	Course     *Course
	ComboLamp  *ComboLampType
}
