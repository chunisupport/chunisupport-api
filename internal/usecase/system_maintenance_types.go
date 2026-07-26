package usecase

import "time"

// MaintenanceState はリクエスト処理が参照するメンテナンス状態の不変スナップショットです。
type MaintenanceState struct {
	Enabled   bool
	Comment   string
	UpdatedAt time.Time
}

// MaintenanceStateProvider はメンテナンス状態の読み取り専用アクセスを提供します。
// 実装はリクエストごとのDBアクセスや読み取りロックを行わないことを前提とします。
type MaintenanceStateProvider interface {
	Current() MaintenanceState
}
