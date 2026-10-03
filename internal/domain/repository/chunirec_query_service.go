package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

type ChunirecProfile struct {
	Name              string
	Level             int
	Rating            *float64
	ClassEmblemID     *int
	ClassEmblemBaseID *int
	Title             *string
	TitleRarity       *string
	UpdatedAt         time.Time
}

type ChunirecRecord struct {
	ID             string
	Title          string
	Difficulty     string
	Genre          string
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
	Score          uint32
	ClearLamp      *string
	ComboLamp      *string
	FullChain      *string
	UpdatedAt      time.Time
}

type ChunirecQueryService interface {
	FindProfileByPlayerID(ctx context.Context, playerID int) (*ChunirecProfile, error)
	ListRecordsByPlayerID(ctx context.Context, playerID int) ([]*ChunirecRecord, error)
}
