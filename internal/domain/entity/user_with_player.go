package entity

// UserWithPlayer はユーザーとプレイヤー情報を合わせたエンティティです。
// 一覧表示などで使用されます。
type UserWithPlayer struct {
	User   User
	Player *Player // プレイヤー情報は存在しない場合があるためポインタ
}
