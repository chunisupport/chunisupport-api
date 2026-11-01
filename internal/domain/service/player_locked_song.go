package service

import "github.com/chunisupport/chunisupport-api/internal/domain/entity"

// PlayerLockedSongSet はプレイヤーの未解禁設定を譜面単位の判定に使える形で保持します。
//
// 未解禁の解禁単位は楽曲の通常譜面群と ULTIMA で異なるため、判定規則を1か所に集約します。
// is_ultima = false は通常譜面群（BASIC / ADVANCED / EXPERT / MASTER）を未解禁にするとともに、
// 同じ楽曲の ULTIMA 譜面も未解禁にします。ULTIMA 譜面だけを未解禁にしたい場合は
// is_ultima = true のレコードで別途表現し、通常譜面群は解禁済みのままにします。
//
// 通常譜面群が未解禁の楽曲は、譜面単位ではなく楽曲ごと OP 計算対象から外れるため、
// 楽曲単位の判定（IsSongLocked）も用意します。
type PlayerLockedSongSet struct {
	normalSongIDs map[int]struct{}
	ultimaSongIDs map[int]struct{}
}

// NewPlayerLockedSongSet は未解禁設定から判定用のセットを生成します。nil 要素は無視します。
func NewPlayerLockedSongSet(lockedSongs []*entity.PlayerLockedSong) PlayerLockedSongSet {
	set := PlayerLockedSongSet{
		normalSongIDs: make(map[int]struct{}, len(lockedSongs)),
		ultimaSongIDs: make(map[int]struct{}, len(lockedSongs)),
	}
	for _, lockedSong := range lockedSongs {
		if lockedSong == nil {
			continue
		}
		set.ultimaSongIDs[lockedSong.SongID] = struct{}{}
		if !lockedSong.IsUltima {
			set.normalSongIDs[lockedSong.SongID] = struct{}{}
		}
	}
	return set
}

// IsLocked は指定した難易度の譜面が未解禁かどうかを返します。
// isUltima が true のときは通常譜面群未解禁・ULTIMA未解禁のどちらでも未解禁と判定します。
func (s PlayerLockedSongSet) IsLocked(songID int, isUltima bool) bool {
	if isUltima {
		return s.IsUltimaLocked(songID)
	}
	return s.IsSongLocked(songID)
}

// IsSongLocked は楽曲の通常譜面群が未解禁かどうかを返します。
func (s PlayerLockedSongSet) IsSongLocked(songID int) bool {
	_, locked := s.normalSongIDs[songID]
	return locked
}

// IsUltimaLocked は楽曲のULTIMA譜面が未解禁かどうかを返します。
// 通常譜面群の未解禁は ULTIMA 譜面の未解禁も兼ねるため、どちらの設定でも true になります。
func (s PlayerLockedSongSet) IsUltimaLocked(songID int) bool {
	_, locked := s.ultimaSongIDs[songID]
	return locked
}

// IsEmpty は未解禁設定が1件もないかを返します。判定処理を省略できる呼び出し元向けです。
func (s PlayerLockedSongSet) IsEmpty() bool {
	return len(s.ultimaSongIDs) == 0
}
