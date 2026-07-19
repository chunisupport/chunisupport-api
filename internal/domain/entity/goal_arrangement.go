package entity

import "errors"

var (
	ErrInvalidGoalArrangement = errors.New("invalid goal arrangement")
	ErrGoalArrangementMissing = errors.New("goal is not included in goal arrangement")
)

// GoalArrangement はユーザーが所有する全目標のグループ所属とグループ内表示順を管理します。
type GoalArrangement struct {
	userID int
	goals  []*Goal
}

// NewGoalArrangement は永続化済みのグループ内順序を保ったまま集約を生成します。
func NewGoalArrangement(userID int, goals []*Goal) (*GoalArrangement, error) {
	owned := append([]*Goal(nil), goals...)
	seen := make(map[uint32]struct{}, len(owned))
	for _, goal := range owned {
		if goal == nil || goal.UserID != userID {
			return nil, ErrInvalidGoalArrangement
		}
		if _, exists := seen[goal.ID]; exists {
			return nil, ErrInvalidGoalArrangement
		}
		seen[goal.ID] = struct{}{}
	}
	arrangement := &GoalArrangement{userID: userID, goals: owned}
	arrangement.normalizeAll()
	return arrangement, nil
}

// UserID は配置を所有するユーザーIDを返します。
func (a *GoalArrangement) UserID() int { return a.userID }

// Goals は集約に含まれる全目標を返します。
func (a *GoalArrangement) Goals() []*Goal { return append([]*Goal(nil), a.goals...) }

// GoalsInGroup は指定グループの目標を表示順で返します。
func (a *GoalArrangement) GoalsInGroup(groupID *uint32) []*Goal {
	goals := make([]*Goal, 0)
	for _, goal := range a.goals {
		if sameOptionalID(goal.GroupID, groupID) {
			goals = append(goals, goal)
		}
	}
	return goals
}

// Reorder は指定グループに現在所属する全目標を指定順へ並び替えます。
func (a *GoalArrangement) Reorder(groupID *uint32, orderedGoalIDs []uint32) error {
	order, err := NewGoalOrder(a.userID, groupID, a.GoalsInGroup(groupID))
	if err != nil {
		return ErrInvalidGoalArrangement
	}
	if err := order.Reorder(orderedGoalIDs); err != nil {
		return err
	}
	ordered := order.Goals()
	next := 0
	for i, goal := range a.goals {
		if sameOptionalID(goal.GroupID, groupID) {
			a.goals[i] = ordered[next]
			next++
		}
	}
	return nil
}

// Move は目標を移動先グループの末尾へ追加し、移動元の順番を詰めます。
func (a *GoalArrangement) Move(goalID uint32, destinationGroupID *uint32) error {
	var target *Goal
	for _, goal := range a.goals {
		if goal.ID == goalID {
			target = goal
			break
		}
	}
	if target == nil {
		return ErrGoalArrangementMissing
	}
	if sameOptionalID(target.GroupID, destinationGroupID) {
		return nil
	}
	sourceGroupID := cloneOptionalID(target.GroupID)
	remaining := make([]*Goal, 0, len(a.goals))
	for _, goal := range a.goals {
		if goal.ID != goalID {
			remaining = append(remaining, goal)
		}
	}
	target.GroupID = cloneOptionalID(destinationGroupID)
	a.goals = append(remaining, target)
	a.normalizeGroup(sourceGroupID)
	a.normalizeGroup(destinationGroupID)
	return nil
}

// Remove は目標を取り除き、同じグループの順番を詰めます。
func (a *GoalArrangement) Remove(goalID uint32) error {
	remaining := make([]*Goal, 0, max(len(a.goals)-1, 0))
	var groupID *uint32
	found := false
	for _, goal := range a.goals {
		if goal.ID == goalID {
			groupID = cloneOptionalID(goal.GroupID)
			found = true
			continue
		}
		remaining = append(remaining, goal)
	}
	if !found {
		return ErrGoalArrangementMissing
	}
	a.goals = remaining
	a.normalizeGroup(groupID)
	return nil
}

// RemoveGroup は所属目標を未分類の末尾へ移動します。
func (a *GoalArrangement) RemoveGroup(groupID uint32) {
	moved := a.GoalsInGroup(&groupID)
	if len(moved) == 0 {
		return
	}
	remaining := make([]*Goal, 0, len(a.goals))
	for _, goal := range a.goals {
		if goal.GroupID == nil || *goal.GroupID != groupID {
			remaining = append(remaining, goal)
		}
	}
	for _, goal := range moved {
		goal.GroupID = nil
	}
	a.goals = append(remaining, moved...)
	a.normalizeGroup(nil)
}

func (a *GoalArrangement) normalizeAll() {
	seenGroups := make(map[uint32]struct{})
	a.normalizeGroup(nil)
	for _, goal := range a.goals {
		if goal.GroupID == nil {
			continue
		}
		if _, exists := seenGroups[*goal.GroupID]; exists {
			continue
		}
		seenGroups[*goal.GroupID] = struct{}{}
		a.normalizeGroup(goal.GroupID)
	}
}

func (a *GoalArrangement) normalizeGroup(groupID *uint32) {
	goals := a.GoalsInGroup(groupID)
	for i, goal := range goals {
		goal.SortOrder = uint16(i + 1)
	}
}
