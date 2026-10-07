package importer

// removeBOM はByte Order Mark (BOM)をデータから除去します。
// UTF-8 BOM (EF BB BF) を検出して削除します。
func removeBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return data[2:]
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return data[2:]
	}
	return data
}
