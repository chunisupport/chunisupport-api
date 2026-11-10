// Package gamedate はCHUNITHMの「今日」と現行バージョンの切り替わりを一か所で定義します。
// 店舗の日替わりとバージョン稼働開始に合わせ、暦日の0時ではなく日本時間7時を境目にします。
// APIとフロントエンドは同じ定義を使います。ここ以外に時刻をハードコードしないでください。
package gamedate

import "time"

// BoundaryHour は日付が切り替わる日本時間の時です。
const BoundaryHour = 7

// jst は夏時間のない日本標準時です。
// time.LoadLocation は tzdata が無い環境で失敗し、日付判定が実行環境に依存するため使いません。
var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// Location は日付判定に使う日本標準時を返します。
func Location() *time.Location {
	return jst
}

// Today は now 時点のCHUNITHMの「今日」（再計算バッチの運用日）を、日本時間0時の時刻として返します。
// 日本時間7時未満は前日、7時以降は当日です。
func Today(now time.Time) time.Time {
	shifted := now.In(jst).Add(-BoundaryHour * time.Hour)
	return time.Date(shifted.Year(), shifted.Month(), shifted.Day(), 0, 0, 0, 0, jst)
}

// StartsAt は稼働日の日本時間7時を返します。この時刻からそのバージョンが現行になります。
// releasedAt は日付だけの値です。年月日は値のロケーションの壁時計から読みます。
// UTCへ変換してから日付を取ると、日本時間0時で保持した稼働日が前日になり、稼働開始が1日ずれます。
func StartsAt(releasedAt time.Time) time.Time {
	year, month, day := releasedAt.Date()
	return time.Date(year, month, day, BoundaryHour, 0, 0, 0, jst)
}

// IsReleased は releasedAt のバージョンが now 時点で稼働開始済みかを返します。
// 稼働日の日本時間7時ちょうどから稼働開始済みです。7時未満は未公開です。
func IsReleased(releasedAt, now time.Time) bool {
	return !now.Before(StartsAt(releasedAt))
}
