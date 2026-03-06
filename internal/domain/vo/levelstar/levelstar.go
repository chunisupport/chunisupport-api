package levelstar

import (
	"database/sql/driver"
	"fmt"
	"strconv"
)

const (
	minLevelStar = 1
	maxLevelStar = 5
)

// LevelStar は WORLD'S END のレベル星数を表す値オブジェクトです。
type LevelStar int

// NewLevelStar は 1-5 の範囲で LevelStar を生成します。
func NewLevelStar(value int) (LevelStar, error) {
	if value < minLevelStar || value > maxLevelStar {
		return 0, fmt.Errorf("level star must be between %d and %d: %d", minLevelStar, maxLevelStar, value)
	}

	return LevelStar(value), nil
}

// Value は driver.Valuer インターフェースを実装します。
func (l *LevelStar) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}

	return int64(*l), nil
}

// Scan は sql.Scanner インターフェースを実装します。
func (l *LevelStar) Scan(value any) error {
	if value == nil {
		*l = 0
		return nil
	}

	var parsed int

	switch v := value.(type) {
	case int64:
		parsed = int(v)
	case []byte:
		n, err := strconv.Atoi(string(v))
		if err != nil {
			return fmt.Errorf("failed to convert []byte to int: %w", err)
		}
		parsed = n
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("failed to convert string to int: %w", err)
		}
		parsed = n
	default:
		return fmt.Errorf("unsupported type %T", v)
	}

	normalized, err := NewLevelStar(parsed)
	if err != nil {
		return err
	}

	*l = normalized
	return nil
}

// Int は int 値に変換します。
func (l LevelStar) Int() int {
	return int(l)
}
