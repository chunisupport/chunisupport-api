// Package songbatch は楽曲データ収集バッチのドメインルールを提供します。
package songbatch

import "errors"

// DataSourceType は楽曲データソースの種類を表します。
type DataSourceType string

const (
	// DataSourceOfficial は公式のデータソースです。
	DataSourceOfficial DataSourceType = "official"
	// DataSourceAdditionalSongs は追加楽曲データソースです。
	DataSourceAdditionalSongs DataSourceType = "additional_songs"
	// DataSourceSt1027 はst1027のデータソースです。
	DataSourceSt1027 DataSourceType = "st1027"
	// DataSourceMainframe はMainframeのデータソースです。
	DataSourceMainframe DataSourceType = "mainframe"
	// DataSourceOtogeDb はotoge-dbのデータソースです。
	DataSourceOtogeDb DataSourceType = "otoge_db"
)

// ErrSourceValidation はデータソースの内容が業務上の前提を満たさないことを表します。
// 解析失敗と検証失敗を別の段階として記録するため、インポーターはこのエラーをラップして返します。
var ErrSourceValidation = errors.New("datasource validation failed")

// SupportedDataSources は通常実行で取得するデータソースを返します。
// 後続のソースほど前のソースの値を補完・上書きするため、並び順は統合順を兼ねます。
func SupportedDataSources() []DataSourceType {
	return []DataSourceType{
		DataSourceOfficial,
		DataSourceAdditionalSongs,
		DataSourceSt1027,
		DataSourceMainframe,
		DataSourceOtogeDb,
	}
}

// ImportedSource は取り込み・検証を終えたデータソースです。
// Data の具体的な型はデータソースの形式に依存するため、インフラ層だけが解釈します。
type ImportedSource struct {
	Type DataSourceType
	Data any
}
