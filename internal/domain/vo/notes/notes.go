package notes

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
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
	if n == nil {
		return fmt.Errorf("cannot scan into nil Notes receiver")
	}

	if value == nil {
		*n = 0
		return nil
	}

	var parsed int

	switch v := value.(type) {
	case int64:
		parsed = int(v)
	case []byte:
		parsedValue, err := strconv.Atoi(string(v))
		if err != nil {
			return fmt.Errorf("failed to convert []byte to int: %w", err)
		}
		parsed = parsedValue
	case string:
		parsedValue, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("failed to convert string to int: %w", err)
		}
		parsed = parsedValue
	default:
		return fmt.Errorf("unsupported type %T", v)
	}

	normalized, err := NewNotes(parsed)
	if err != nil {
		return err
	}

	*n = normalized
	return nil
}
