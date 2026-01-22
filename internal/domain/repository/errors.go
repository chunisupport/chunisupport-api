package repository

import "errors"

// リポジトリ層の共通エラー定義
// infraからusecaseへの依存を避けるため、ドメイン関連エラーはここで定義します。
var (
	// ErrSongNotFound は楽曲が見つからなかった場合に返されるエラーです。
	ErrSongNotFound = errors.New("song not found")
)
