package utils

import (
	"github.com/matthewhartstonge/argon2"
)

// HashPasswordWithPepper はパスワードとペッパーを組み合わせてArgon2ハッシュ（PHC形式）を生成します。
// ソルトはライブラリによって自動的に生成されます。
func HashPasswordWithPepper(password, pepper string) (string, error) {
	// パスワード + ペッパー を結合
	combined := []byte(password + pepper)

	// デフォルト設定でハッシュ化
	// DefaultConfigはRFC9106で推奨されているメモリ制約のある設定 (64 MiB) を使用します
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded(combined)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

// CheckPasswordHashWithPepper はパスワード、ペッパーとPHC形式のハッシュを比較します。
func CheckPasswordHashWithPepper(password, pepper, encodedHash string) bool {
	// パスワード + ペッパー を結合
	combined := []byte(password + pepper)

	// ハッシュを検証
	ok, err := argon2.VerifyEncoded(combined, []byte(encodedHash))
	if err != nil {
		return false
	}
	return ok
}
