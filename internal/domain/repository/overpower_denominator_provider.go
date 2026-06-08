package repository

import "context"

// OverpowerDenominatorSnapshot は最新マスタから作ったOVER POWER割合分母のスナップショットです。
type OverpowerDenominatorSnapshot struct {
	GlobalTotal            float64
	SongMaxOP              map[int]float64
	SongMaxOPWithoutUltima map[int]float64
}

// OverpowerDenominatorProvider はプロフィール返却時のOVER POWER割合分母を提供します。
type OverpowerDenominatorProvider interface {
	Snapshot(ctx context.Context) (*OverpowerDenominatorSnapshot, error)
	Invalidate(ctx context.Context)
}
