package entity

import "errors"

var (
	ErrInvalidGoalGroupOrder = errors.New("invalid goal group order")
	ErrGoalGroupOrderMissing = errors.New("goal group is not included in goal group order")
)

// GoalGroupOrder はユーザーが所有する目標グループの表示順を表す集約です。
type GoalGroupOrder struct {
	userID int
	groups []*GoalGroup
}

// NewGoalGroupOrder は一覧取得順を正としてグループ順を正規化します。
func NewGoalGroupOrder(userID int, groups []*GoalGroup) (*GoalGroupOrder, error) {
	ordered := append([]*GoalGroup(nil), groups...)
	seenIDs := make(map[uint32]struct{}, len(ordered))
	for _, group := range ordered {
		if group == nil || group.UserID != userID {
			return nil, ErrInvalidGoalGroupOrder
		}
		if _, exists := seenIDs[group.ID]; exists {
			return nil, ErrInvalidGoalGroupOrder
		}
		seenIDs[group.ID] = struct{}{}
	}
	order := &GoalGroupOrder{userID: userID, groups: ordered}
	order.assignSortOrders()
	return order, nil
}

// UserID は表示順を所有するユーザーIDを返します。
func (o *GoalGroupOrder) UserID() int { return o.userID }

// Groups は表示順どおりのグループを返します。
func (o *GoalGroupOrder) Groups() []*GoalGroup {
	return append([]*GoalGroup(nil), o.groups...)
}

// Reorder は現在含まれるすべてのグループIDを指定順へ並び替えます。
func (o *GoalGroupOrder) Reorder(orderedGroupIDs []uint32) error {
	if len(o.groups) != len(orderedGroupIDs) {
		return ErrInvalidGoalGroupOrder
	}
	groupsByID := make(map[uint32]*GoalGroup, len(o.groups))
	for _, group := range o.groups {
		groupsByID[group.ID] = group
	}
	ordered := make([]*GoalGroup, 0, len(o.groups))
	for _, id := range orderedGroupIDs {
		group, ok := groupsByID[id]
		if !ok {
			return ErrInvalidGoalGroupOrder
		}
		ordered = append(ordered, group)
		delete(groupsByID, id)
	}
	o.groups = ordered
	o.assignSortOrders()
	return nil
}

// Remove は指定グループを取り除き、残りの順番を詰めます。
func (o *GoalGroupOrder) Remove(id uint32) error {
	remaining := make([]*GoalGroup, 0, max(len(o.groups)-1, 0))
	found := false
	for _, group := range o.groups {
		if group.ID == id {
			found = true
			continue
		}
		remaining = append(remaining, group)
	}
	if !found {
		return ErrGoalGroupOrderMissing
	}
	o.groups = remaining
	o.assignSortOrders()
	return nil
}

func (o *GoalGroupOrder) assignSortOrders() {
	for i, group := range o.groups {
		group.SortOrder = uint16(i + 1)
	}
}
