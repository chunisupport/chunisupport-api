package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// scanNullableTime は1カラムを返すクエリを実行し、NULL許容のtime.Timeを返します。
// DBドライバごとにtime.Time/[]byte/stringのいずれかで返却されるため、型スイッチで吸収します。
func scanNullableTime(ctx context.Context, exec repository.Executor, query string, args ...any) (*time.Time, error) {
	var raw any
	if err := exec.QueryRowxContext(ctx, query, args...).Scan(&raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	switch value := raw.(type) {
	case time.Time:
		updatedAt := value
		return &updatedAt, nil
	case []byte:
		return parseTimeString(string(value))
	case string:
		return parseTimeString(value)
	default:
		return nil, fmt.Errorf("unsupported updated_at type: %T", raw)
	}
}

// parseTimeString はtime.Localを使って時刻文字列をパースします。
func parseTimeString(value string) (*time.Time, error) {
	return parseTimeStringInLocation(value, time.UTC)
}

// parseTimeStringInLocation は指定したロケーションで時刻文字列をパースします。
// タイムゾーン情報を含むフォーマットを優先し、ない場合はlocationを適用します。
func parseTimeStringInLocation(value string, location *time.Location) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	zonedLayouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
	}
	for _, layout := range zonedLayouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return &parsed, nil
		}
	}

	localLayouts := []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range localLayouts {
		parsed, err := time.ParseInLocation(layout, trimmed, location)
		if err == nil {
			return &parsed, nil
		}
	}

	return nil, fmt.Errorf("failed to parse updated_at: %s", trimmed)
}
