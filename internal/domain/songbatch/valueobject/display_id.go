package valueobject

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// DisplayID はマスタデータの表示用IDを表す値オブジェクト
type DisplayID string

// NewDisplayID は新しいDisplayIDを生成します（crypto/rand使用）
func NewDisplayID() (DisplayID, error) {
	b := make([]byte, 8) // 16文字の16進数
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate display ID: %w", err)
	}
	return DisplayID(hex.EncodeToString(b)), nil
}

// String はDisplayIDを文字列として返します
func (d DisplayID) String() string {
	return string(d)
}

// IsEmpty はDisplayIDが空かどうかを返します
func (d DisplayID) IsEmpty() bool {
	return d == ""
}
