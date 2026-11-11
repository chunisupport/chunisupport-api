package featureflag

import "slices"

const (
	// ProfileComment はプロフィールコメントの取得・保存と、保存時の内容判定を有効にします。
	// 無効な環境ではエンドポイントを登録せず、OpenAI Moderation API も呼びません。
	ProfileComment Name = "profile_comment"
)

// 機能名の定数を追加するときは、この登録一覧と feature-flag.yml にも必ず追加します。
// 定数だけを追加した場合の登録漏れは自動検出しません。
var registeredNames = []Name{
	ProfileComment,
}

// RegisteredNames は呼び出し元から登録一覧を変更できないようにコピーを返します。
func RegisteredNames() []Name {
	return slices.Clone(registeredNames)
}
