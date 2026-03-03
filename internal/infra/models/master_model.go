package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
)

// MasterModel は全てのマスタテーブルで共通のモデル構造です。
type MasterModel struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// ToAccountType はMasterModelをmaster.AccountTypeに変換します。
func (m *MasterModel) ToAccountType() *master.AccountType {
	return &master.AccountType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToClearLampType はMasterModelをmaster.ClearLampTypeに変換します。
func (m *MasterModel) ToClearLampType() *master.ClearLampType {
	return &master.ClearLampType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToComboLampType はMasterModelをmaster.ComboLampTypeに変換します。
func (m *MasterModel) ToComboLampType() *master.ComboLampType {
	return &master.ComboLampType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToFullChainType はMasterModelをmaster.FullChainTypeに変換します。
func (m *MasterModel) ToFullChainType() *master.FullChainType {
	return &master.FullChainType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToHonorType はMasterModelをmaster.HonorTypeに変換します。
func (m *MasterModel) ToHonorType() *master.HonorType {
	return &master.HonorType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToChartDifficulty はMasterModelをmaster.ChartDifficultyに変換します。
func (m *MasterModel) ToChartDifficulty() *master.ChartDifficulty {
	return &master.ChartDifficulty{
		ID:   m.ID,
		Name: m.Name,
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

// ToClassEmblem はMasterModelをmaster.ClassEmblemに変換します。
func (m *MasterModel) ToClassEmblem() *master.ClassEmblem {
	return &master.ClassEmblem{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToClassEmblemBase はMasterModelをmaster.ClassEmblemBaseに変換します。
func (m *MasterModel) ToClassEmblemBase() *master.ClassEmblemBase {
	return &master.ClassEmblemBase{
		ID:   m.ID,
		Name: m.Name,
	}
}

// FromMasterEntity は任意のマスタエンティティをMasterModelに変換します。
func FromMasterEntity(id int, name string) *MasterModel {
	return &MasterModel{
		ID:   id,
		Name: name,
	}
}
