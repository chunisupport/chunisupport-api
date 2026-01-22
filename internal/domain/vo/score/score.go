package score

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
)

const maxScore = 1010000

// Score はスコアの値オブジェクトです。
type Score uint32

// NewScore は新しい Score を生成します。
// スコアは 0 から 1,010,000 の範囲である必要があります。
func NewScore(value uint32) (Score, error) {
	if value > maxScore {
		return 0, errors.New("score cannot exceed 1,010,000")
	}
	return Score(value), nil
}

// Value は driver.Valuer インターフェースを実装します。
func (s Score) Value() (driver.Value, error) {
	return int64(s), nil
}

// Scan は sql.Scanner インターフェースを実装します。
func (s *Score) Scan(value any) error {
	if value == nil {
		*s = 0
		return nil
	}

	var scoreValue int64

	switch v := value.(type) {
	case int64:
		scoreValue = v
	case []byte:
		parsed, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to convert []byte to int: %w", err)
		}
		scoreValue = parsed
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to convert string to int: %w", err)
		}
		scoreValue = parsed
	default:
		return fmt.Errorf("unsupported type %T", v)
	}

	if scoreValue < 0 {
		return errors.New("score cannot be negative")
	}
	if scoreValue > maxScore {
		return errors.New("score cannot exceed 1,010,000")
	}

	*s = Score(scoreValue)
	return nil
}
