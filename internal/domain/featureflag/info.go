package featureflag

import "slices"

// 機能名の定数を追加するときは、この登録一覧と feature-flag.yml にも必ず追加します。
// 定数だけを追加した場合の登録漏れは自動検出しません。
var registeredNames = []Name{}

// RegisteredNames は呼び出し元から登録一覧を変更できないようにコピーを返します。
func RegisteredNames() []Name {
	return slices.Clone(registeredNames)
}
