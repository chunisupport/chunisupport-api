package api_internal

import "time"

// GoalGroupRequest は目標グループ作成・更新リクエストです。
type GoalGroupRequest struct {
	Name string `json:"name"`
}

// GoalGroupResponse は目標グループレスポンスです。
type GoalGroupResponse struct {
	ID        uint32    `json:"id"`
	Name      string    `json:"name"`
	SortOrder uint16    `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// GoalGroupsResponse は目標グループ一覧レスポンスです。
type GoalGroupsResponse struct {
	Groups []*GoalGroupResponse `json:"groups"`
}

// GoalGroupOrderRequest は目標グループ並び順更新リクエストです。
type GoalGroupOrderRequest struct {
	GroupIDs []uint32 `json:"group_ids"`
}
