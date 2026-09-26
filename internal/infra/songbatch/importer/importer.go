// Package importer は楽曲データソースのJSONを読み込み、検証済みのデータへ変換します。
package importer

import "github.com/chunisupport/chunisupport-api/internal/domain/songbatch"

// Importer はデータソースからデータをインポートするためのインターフェースです
type Importer interface {
	Import(filePath string) (*songbatch.ImportedSource, error)
}
