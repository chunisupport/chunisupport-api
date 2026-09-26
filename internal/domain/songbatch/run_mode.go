package songbatch

import (
	"errors"
	"slices"
)

// RunMode はバッチの実行モードです。
type RunMode string

const (
	// RunModeNormal は通常実行です。
	RunModeNormal RunMode = "NORMAL"
	// RunModeMajorUpdate は大型更新です。
	RunModeMajorUpdate RunMode = "MAJOR_UPDATE"
)

// ErrInvalidRunMode は未定義の実行モードが指定されたことを表します。
var ErrInvalidRunMode = errors.New("invalid song batch run mode")

// ParseRunMode は文字列から実行モードを生成します。
func ParseRunMode(value string) (RunMode, error) {
	mode := RunMode(value)
	if mode != RunModeNormal && mode != RunModeMajorUpdate {
		return "", ErrInvalidRunMode
	}
	return mode, nil
}

// TargetSources は実行モードで取得対象とするデータソースを返します。
// 大型更新では公式データと追加楽曲だけを正とし、他ソースの古い定数で上書きしないよう最初から対象外にします。
func (m RunMode) TargetSources() []DataSourceType {
	if m == RunModeMajorUpdate {
		return []DataSourceType{DataSourceOfficial, DataSourceAdditionalSongs}
	}
	return SupportedDataSources()
}

// RequiredSources は実行モードで欠けてはならないデータソースを返します。
// 必須ソースが1つでも利用できない場合、部分的なデータでMySQLを更新しないようバッチ全体を失敗させます。
func (m RunMode) RequiredSources() []DataSourceType {
	required := []DataSourceType{DataSourceOfficial, DataSourceAdditionalSongs}
	if m != RunModeMajorUpdate {
		required = append(required, DataSourceMainframe)
	}
	return required
}

// IsRequired は指定データソースが実行モードで必須かどうかを返します。
func (m RunMode) IsRequired(sourceType DataSourceType) bool {
	return slices.Contains(m.RequiredSources(), sourceType)
}

// RunRequest はバッチ1回分の実行条件です。flag 名や環境変数名は含みません。
type RunRequest struct {
	Mode                   RunMode
	FillMissingReleaseDate bool
}

// NewRunRequest はフラグ値から実行リクエストを組み立てます。
func NewRunRequest(majorUpdate, fillMissingReleaseDate bool) RunRequest {
	mode := RunModeNormal
	if majorUpdate {
		mode = RunModeMajorUpdate
	}
	return RunRequest{
		Mode:                   mode,
		FillMissingReleaseDate: fillMissingReleaseDate,
	}
}

// LockConflictIsError はロック競合時にエラー終了すべきかを返します。
// 大型更新は運用者が明示的に起動するため、実行されなかったことを失敗として知らせます。
func (r RunRequest) LockConflictIsError() bool {
	return r.Mode == RunModeMajorUpdate
}
