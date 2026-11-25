package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/scorepercentile"
)

// ScorePercentileProvider は通常譜面・WORLD'S END譜面・コースの上位%スナップショットをプロセス内で共有します。
// DBを正とし、DB上の統計変更世代が公開中の世代と異なるときだけ全件を再構築して差し替えます。
type ScorePercentileProvider interface {
	// Snapshot は公開中のスナップショットを返します。
	// 未構築の場合だけ構築を試み、失敗した場合はエラーを返します。構築済みならDBへ問い合わせません。
	Snapshot(ctx context.Context) (*scorepercentile.Snapshot, error)

	// Refresh はDB上の統計変更世代を確認し、公開中の世代と異なる場合に再構築して差し替えます。
	// 同時に呼ばれた場合は直列化し、先行した再構築で最新になっていれば再構築を省略します。
	// 失敗した場合は公開中のスナップショットを維持します。
	Refresh(ctx context.Context) error
}
