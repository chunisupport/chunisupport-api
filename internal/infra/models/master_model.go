package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
)

// MasterModel は sort_order を持たないマスタテーブルで共通のモデル構造です。
type MasterModel struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// SortedMasterModel は sort_order カラムを持つマスタテーブルで共通のモデル構造です。
type SortedMasterModel struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	SortOrder int    `db:"sort_order"`
}

// ToAccountType はMasterModelをmaster.AccountTypeに変換します。
func (m *MasterModel) ToAccountType() *master.AccountType {
	return &master.AccountType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToClearLampType はSortedMasterModelをmaster.ClearLampTypeに変換します。
func (m *SortedMasterModel) ToClearLampType() *master.ClearLampType {
	return &master.ClearLampType{
		ID:        m.ID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
	}
}

// ToComboLampType はSortedMasterModelをmaster.ComboLampTypeに変換します。
func (m *SortedMasterModel) ToComboLampType() *master.ComboLampType {
	return &master.ComboLampType{
		ID:        m.ID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
	}
}

// ToFullChainType はSortedMasterModelをmaster.FullChainTypeに変換します。
func (m *SortedMasterModel) ToFullChainType() *master.FullChainType {
	return &master.FullChainType{
		ID:        m.ID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
	}
}

// ToHonorType はMasterModelをmaster.HonorTypeに変換します。
func (m *MasterModel) ToHonorType() *master.HonorType {
	return &master.HonorType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToChartDifficulty はSortedMasterModelをmaster.ChartDifficultyに変換します。
func (m *SortedMasterModel) ToChartDifficulty() *master.ChartDifficulty {
	return &master.ChartDifficulty{
		ID:        m.ID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
	}
}

// ToGenre はMasterModelをmaster.Genreに変換します。
func (m *MasterModel) ToGenre() *master.Genre {
	return &master.Genre{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToSlot はMasterModelをmaster.Slotに変換します。
func (m *MasterModel) ToSlot() *master.Slot {
	return &master.Slot{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToClassEmblem はSortedMasterModelをmaster.ClassEmblemに変換します。
func (m *SortedMasterModel) ToClassEmblem() *master.ClassEmblem {
	return &master.ClassEmblem{
		ID:        m.ID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
	}
}

// ToClassEmblemBase はSortedMasterModelをmaster.ClassEmblemBaseに変換します。
func (m *SortedMasterModel) ToClassEmblemBase() *master.ClassEmblemBase {
	return &master.ClassEmblemBase{
		ID:        m.ID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
	}
}

// FromMasterEntity は任意のマスタエンティティをMasterModelに変換します。
func FromMasterEntity(id int, name string) *MasterModel {
	return &MasterModel{
		ID:   id,
		Name: name,
	}
}
