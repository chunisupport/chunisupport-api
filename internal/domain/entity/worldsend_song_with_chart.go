package entity

// WorldsendSongWithChart は WORLD'S END 楽曲とその譜面情報を保持する構造体です。
// WORLD'S END は1曲1譜面が保証されています。
type WorldsendSongWithChart struct {
	Song  *Song
	Chart *WorldsendChart
}
