package notes

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

// Notes はノーツ数の値オブジェクトです。
type Notes int

// NewNotes は新しい Notes を生成します。
// ノーツ数は0以上である必要があります。
func NewNotes(value int) (Notes, error) {
	if value < 0 {
		return 0, errors.New("notes count must be 0 or greater")
	}
	return Notes(value), nil
}

// Value は driver.Valuer インターフェースを実装します。
func (n *Notes) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	return int64(*n), nil
}

// Scan は sql.Scanner インターフェースを実装します。
func (n *Notes) Scan(value any) error {
	if value == nil {
		*n = 0
		return nil
	}

	switch v := value.(type) {
	case int64:
		*n = Notes(v)
		return nil
	default:
		return fmt.Errorf("サポートされていない型(%T)です", v)
	}
}
