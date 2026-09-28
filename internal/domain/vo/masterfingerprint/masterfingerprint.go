package masterfingerprint

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"regexp"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo"
)

// Fingerprint は再計算に使ったマスタと計算ロジックを識別するSHA-256値です。
// 値が同じであれば、同じプレイヤーデータから同じ計算結果が得られることを表します。
type Fingerprint struct {
	value string
}

var (
	fingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

	ErrInvalidFingerprint = errors.New("fingerprint must be exactly 64 hexadecimal characters (lowercase)")
)

// NewFingerprint は16進数小文字64文字の文字列からFingerprintを生成します。
func NewFingerprint(value string) (Fingerprint, error) {
	if !fingerprintPattern.MatchString(value) {
		return Fingerprint{}, ErrInvalidFingerprint
	}
	return Fingerprint{value: value}, nil
}

// Compute はデータのSHA-256値からFingerprintを生成します。
func Compute(data []byte) Fingerprint {
	sum := sha256.Sum256(data)
	return Fingerprint{value: hex.EncodeToString(sum[:])}
}

// String はFingerprintの16進数表現を返します。
func (f Fingerprint) String() string {
	return f.value
}

// Value は database/sql の driver.Valuer インターフェースを実装します。
// コンストラクタを通していないゼロ値は、空文字として保存されないようエラーにします。
func (f Fingerprint) Value() (driver.Value, error) {
	if f.value == "" {
		return nil, ErrInvalidFingerprint
	}
	return f.value, nil
}

// Scan はDBから取得した値を検証し、Fingerprintを復元します。
// NULLはポインタ型のフィールドで表現するため、ここでは不正値として扱います。
func (f *Fingerprint) Scan(src any) error {
	if src == nil {
		return ErrInvalidFingerprint
	}
	s, err := vo.ToString(src)
	if err != nil {
		return err
	}
	fingerprint, err := NewFingerprint(s)
	if err != nil {
		return err
	}
	*f = fingerprint
	return nil
}
