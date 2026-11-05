// Package featureflag は環境や設定ファイル形式に依存せず、機能の利用可否を表します。
package featureflag

// Name は機能を識別する名前です。
type Name string

// Flags は有効な機能の読み取り専用集合です。ゼロ値は全機能が無効です。
// 生成後に内部の集合を変更しないため、値を共有して並行して参照できます。
type Flags struct {
	enabled map[Name]struct{}
}

// New は呼び出し元のスライスを保持せず、有効な機能の集合を生成します。
// 本番の登録一覧との突き合わせは設定読み込み側で行い、テストでは任意の名前を渡せます。
func New(enabled ...Name) Flags {
	flags := Flags{enabled: make(map[Name]struct{}, len(enabled))}
	for _, name := range enabled {
		flags.enabled[name] = struct{}{}
	}
	return flags
}

// Enabled は集合に登録された機能かを判定し、未登録の名前には false を返します。
func (f Flags) Enabled(name Name) bool {
	_, ok := f.enabled[name]
	return ok
}
