package usecase_test

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
)

func TestMasterDataUsecase_NameFolders(t *testing.T) {
	tests := []struct {
		name    string
		masters *masterdata.MasterDataMasters
		want    []masterdata.NameFolder
	}{
		{name: "表示順で並べ内部ID順に依存しない", masters: &masterdata.MasterDataMasters{
			NameFolders: map[string]masterdata.NameFolder{
				"KA":   {ID: 1, Code: "KA", Name: "か行", SortOrder: 8},
				"ABCD": {ID: 9, Code: "ABCD", Name: "ABCD", SortOrder: 1},
			},
		}, want: []masterdata.NameFolder{
			{ID: 9, Code: "ABCD", Name: "ABCD", SortOrder: 1},
			{ID: 1, Code: "KA", Name: "か行", SortOrder: 8},
		}},
		{name: "マスタ未取得でも空配列", want: []masterdata.NameFolder{}},
		{name: "空マスタでも空配列", masters: &masterdata.MasterDataMasters{}, want: []masterdata.NameFolder{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewMasterDataUsecase(&masterDataMasterProviderMock{masters: tt.masters}, &chartStatsMasterProviderMock{})
			assert.Equal(t, tt.want, uc.GetNameFolders(context.Background()))
			assert.Equal(t, tt.want, uc.GetMasterData(context.Background()).NameFolders)
		})
	}
}
