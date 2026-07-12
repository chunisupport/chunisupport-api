package coursescore

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
)

const Max uint32 = 3030000

// CourseScore は3曲分のコーススコアを表す不変の値オブジェクトです。
type CourseScore uint32

// New は有効範囲を検証してコーススコアを生成します。
func New(value uint32) (CourseScore, error) {
	if value > Max {
		return 0, fmt.Errorf("course score cannot exceed %d", Max)
	}
	return CourseScore(value), nil
}

// Uint32 はスコアを符号なし整数で返します。
func (s CourseScore) Uint32() uint32 { return uint32(s) }

// Value はdriver.Valuerを実装します。
func (s CourseScore) Value() (driver.Value, error) { return int64(s), nil }

// Scan はsql.Scannerを実装します。
func (s *CourseScore) Scan(value any) error {
	if value == nil {
		*s = 0
		return nil
	}
	var n int64
	switch v := value.(type) {
	case int64:
		n = v
	case []byte:
		parsed, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse course score: %w", err)
		}
		n = parsed
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse course score: %w", err)
		}
		n = parsed
	default:
		return fmt.Errorf("unsupported course score type %T", value)
	}
	if n < 0 {
		return errors.New("course score cannot be negative")
	}
	parsed, err := New(uint32(n))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
