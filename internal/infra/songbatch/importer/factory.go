package importer

import (
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
)

// SourceImporter はデータソースの種類に応じたインポーターでファイルを読み込みます。
type SourceImporter struct{}

// NewSourceImporter は SourceImporter を生成します。
func NewSourceImporter() SourceImporter {
	return SourceImporter{}
}

// Import はデータソースの種類に応じたインポーターでファイルを読み込みます。
func (SourceImporter) Import(sourceType songbatch.DataSourceType, filePath string) (*songbatch.ImportedSource, error) {
	dsImporter, err := createImporter(sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to create importer for type %s: %w", sourceType, err)
	}
	return dsImporter.Import(filePath)
}

// createImporter はデータソースの種類に応じたインポーターを生成します
func createImporter(dataSourceType songbatch.DataSourceType) (Importer, error) {
	switch dataSourceType {
	case songbatch.DataSourceOfficial:
		return NewOfficialImporter(), nil
	case songbatch.DataSourceMainframe:
		return NewMainframeImporter(), nil
	case songbatch.DataSourceOtogeDb:
		return NewOtogeDbImporter(), nil
	case songbatch.DataSourceAdditionalSongs:
		return NewAdditionalSongsImporter(), nil
	case songbatch.DataSourceSt1027:
		return NewSt1027Importer(), nil
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", dataSourceType)
	}
}
