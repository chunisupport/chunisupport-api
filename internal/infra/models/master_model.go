package models

import "github.com/Qman110101/chunisupport-api/internal/domain/entity"

// MasterModel は全てのマスタテーブルで共通のモデル構造です。
type MasterModel struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// ToAccountType はMasterModelをentity.AccountTypeに変換します。
func (m *MasterModel) ToAccountType() *entity.AccountType {
	return &entity.AccountType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToClearLampType はMasterModelをentity.ClearLampTypeに変換します。
func (m *MasterModel) ToClearLampType() *entity.ClearLampType {
	return &entity.ClearLampType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToComboLampType はMasterModelをentity.ComboLampTypeに変換します。
func (m *MasterModel) ToComboLampType() *entity.ComboLampType {
	return &entity.ComboLampType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToFullChainType はMasterModelをentity.FullChainTypeに変換します。
func (m *MasterModel) ToFullChainType() *entity.FullChainType {
	return &entity.FullChainType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToHonorType はMasterModelをentity.HonorTypeに変換します。
func (m *MasterModel) ToHonorType() *entity.HonorType {
	return &entity.HonorType{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToChartDifficulty はMasterModelをentity.ChartDifficultyに変換します。
func (m *MasterModel) ToChartDifficulty() *entity.ChartDifficulty {
	return &entity.ChartDifficulty{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToGenre はMasterModelをentity.Genreに変換します。
func (m *MasterModel) ToGenre() *entity.Genre {
	return &entity.Genre{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToSlot はMasterModelをentity.Slotに変換します。
func (m *MasterModel) ToSlot() *entity.Slot {
	return &entity.Slot{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToClassEmblem はMasterModelをentity.ClassEmblemに変換します。
func (m *MasterModel) ToClassEmblem() *entity.ClassEmblem {
	return &entity.ClassEmblem{
		ID:   m.ID,
		Name: m.Name,
	}
}

// ToClassEmblemBase はMasterModelをentity.ClassEmblemBaseに変換します。
func (m *MasterModel) ToClassEmblemBase() *entity.ClassEmblemBase {
	return &entity.ClassEmblemBase{
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
