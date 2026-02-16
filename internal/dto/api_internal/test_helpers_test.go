package api_internal

// containsString はstrがsubstrを含むかどうかを判定します。
func containsString(str, substr string) bool {
	return indexOfString(str, substr) != -1
}

// indexOfString はstrの中でsubstrが最初に現れる位置を返します。見つからない場合は-1を返します。
func indexOfString(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
