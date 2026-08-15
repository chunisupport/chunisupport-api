package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
)

// GoalGroup はユーザーが目標を分類するグループです。
type GoalGroup struct {
	ID        uint32
	UserID    int
	Name      goalgroupname.GoalGroupName
	SortOrder uint16
	CreatedAt time.Time
}

// NewGoalGroup は検証済みの名前を持つ目標グループを生成します。
func NewGoalGroup(userID int, name string) (*GoalGroup, error) {
	validatedName, err := goalgroupname.NewGoalGroupName(name)
	if err != nil {
		return nil, err
	}
	return &GoalGroup{UserID: userID, Name: validatedName}, nil
}

// Rename は検証済みの名前へ変更します。
func (g *GoalGroup) Rename(name string) error {
	validatedName, err := goalgroupname.NewGoalGroupName(name)
	if err != nil {
		return err
	}
	g.Name = validatedName
	return nil
}
