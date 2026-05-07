package entity

// PlayerLockedSong はプレイヤーごとの未解禁楽曲状態を表します。
type PlayerLockedSong struct {
	PlayerID int
	SongID   int
	IsUltima bool
}
