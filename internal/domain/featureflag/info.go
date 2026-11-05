package featureflag

import "slices"

// FeatureFlagProbe は feature-flag の配備設定が各環境で意図どおりに効いているかを、
// 管理画面から確認するための機能です。確認が済んだら定数・登録一覧・YAMLの定義を削除します。
const FeatureFlagProbe Name = "feature_flag_probe"

// 機能名の定数を追加するときは、この登録一覧と feature-flag.yml にも必ず追加します。
// 定数だけを追加した場合の登録漏れは自動検出しません。
var registeredNames = []Name{FeatureFlagProbe}

// RegisteredNames は呼び出し元から登録一覧を変更できないようにコピーを返します。
func RegisteredNames() []Name {
	return slices.Clone(registeredNames)
}
