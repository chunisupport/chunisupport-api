package api_internal

import (
	"encoding/json"
	"fmt"
	"time"
)

const dateOnlyLayout = "2006-01-02"

type DateOnly struct {
	time.Time
}

func (d *DateOnly) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parsed, err := time.Parse(dateOnlyLayout, raw)
	if err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD: %w", err)
	}

	d.Time = parsed
	return nil
}

func (d *DateOnly) TimePtr() *time.Time {
	if d == nil {
		return nil
	}

	t := d.Time
	return &t
}
