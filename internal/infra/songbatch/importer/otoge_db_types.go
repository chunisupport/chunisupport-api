package importer

// OtogeDbSong はotoge-dbデータソースの単一の楽曲エントリを表します。
// JSON のキーとの対応は jsonFields で定義します。フィールドを追加したら jsonFields にも追加してください。
type OtogeDbSong struct {
	ID            string // 楽曲ID (文字列)
	Title         string // 楽曲タイトル
	DateAdded     string // 追加日 (YYYYMMDD形式)
	Version       string // バージョン情報
	Catname       string // カテゴリ名
	Artist        string // アーティスト
	Image         string // 画像ファイル名
	BPM           string // BPM
	WeKanji       string // WORLD'S ENDカテゴリ漢字
	WeStar        string // WORLD'S END星数
	LevWENotes    string // WORLD'S END ノーツ数
	LevWEDesigner string // WORLD'S END 譜面製作者
	WikiwikiURL   string // CHUNITHM攻略wiki(wikiwiki)の楽曲ページURL
}

// otogeDbField は otoge-db の JSON キーと読み込み先のフィールドの対応です。
type otogeDbField struct {
	key string
	dst *string
}

// jsonFields は JSON キーと各フィールドの対応を返します。
// otoge-db はフィールド単位で型が崩れることがあるため、構造体タグで一括デコードせず、
// この対応表に沿ってフィールドごとに読み込みます。
func (s *OtogeDbSong) jsonFields() []otogeDbField {
	return []otogeDbField{
		{key: "id", dst: &s.ID},
		{key: "title", dst: &s.Title},
		{key: "date_added", dst: &s.DateAdded},
		{key: "version", dst: &s.Version},
		{key: "catname", dst: &s.Catname},
		{key: "artist", dst: &s.Artist},
		{key: "image", dst: &s.Image},
		{key: "bpm", dst: &s.BPM},
		{key: "we_kanji", dst: &s.WeKanji},
		{key: "we_star", dst: &s.WeStar},
		{key: "lev_we_notes", dst: &s.LevWENotes},
		{key: "lev_we_designer", dst: &s.LevWEDesigner},
		{key: "wikiwiki_url", dst: &s.WikiwikiURL},
	}
}

// OtogeDbData はotoge-dbデータソースの楽曲データのコレクションです
type OtogeDbData []OtogeDbSong
