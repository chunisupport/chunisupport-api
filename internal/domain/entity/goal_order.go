package entity

import "errors"

var (
	ErrInvalidGoalOrder = errors.New("invalid goal order")
	ErrGoalOrderMissing = errors.New("goal is not included in goal order")
)

// GoalOrder はユーザーが所有する目標全体の表示順を表す集約です。
type GoalOrder struct {
	userID int
	goals  []*Goal
}

// NewGoalOrder は同一ユーザーの目標から表示順集約を生成します。
// 呼び出し元が一覧取得順で渡した目標を正規の表示順として採用し、既存のSortOrder値は連番へ正規化します。
func NewGoalOrder(userID int, goals []*Goal) (*GoalOrder, error) {
	ordered := append([]*Goal(nil), goals...)
	seenIDs := make(map[uint32]struct{}, len(ordered))
	for _, goal := range ordered {
		if goal == nil || goal.UserID != userID {
			return nil, ErrInvalidGoalOrder
		}
		if _, exists := seenIDs[goal.ID]; exists {
			return nil, ErrInvalidGoalOrder
		}
		seenIDs[goal.ID] = struct{}{}
	}
	order := &GoalOrder{userID: userID, goals: ordered}
	order.assignSortOrders()
	return order, nil
}

// UserID は表示順を所有するユーザーIDを返します。
func (o *GoalOrder) UserID() int {
	return o.userID
}

// Goals は表示順どおりの目標を返します。
func (o *GoalOrder) Goals() []*Goal {
	return append([]*Goal(nil), o.goals...)
}

// Reorder は現在含まれるすべての目標IDを指定順へ並び替えます。
func (o *GoalOrder) Reorder(orderedGoalIDs []uint32) error {
	if len(o.goals) != len(orderedGoalIDs) {
		return ErrInvalidGoalOrder
	}
	goalsByID := make(map[uint32]*Goal, len(o.goals))
	for _, goal := range o.goals {
		goalsByID[goal.ID] = goal
	}
	ordered := make([]*Goal, 0, len(o.goals))
	for _, id := range orderedGoalIDs {
		goal, ok := goalsByID[id]
		if !ok {
			return ErrInvalidGoalOrder
		}
		ordered = append(ordered, goal)
		delete(goalsByID, id)
	}
	if len(goalsByID) != 0 {
		return ErrInvalidGoalOrder
	}
	o.goals = ordered
	o.assignSortOrders()
	return nil
}

// Remove は指定目標を表示順集約から取り除き、残りを連番へ詰め直します。
func (o *GoalOrder) Remove(id uint32) error {
	remaining := make([]*Goal, 0, max(len(o.goals)-1, 0))
	found := false
	for _, goal := range o.goals {
		if goal.ID == id {
			found = true
			continue
		}
		remaining = append(remaining, goal)
	}
	if !found {
		return ErrGoalOrderMissing
	}
	o.goals = remaining
	o.assignSortOrders()
	return nil
}

func (o *GoalOrder) assignSortOrders() {
	for i, goal := range o.goals {
		goal.SortOrder = uint16(i + 1)
	}
}
