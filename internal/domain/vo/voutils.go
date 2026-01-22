package vo

import "fmt"

func ToString(v any) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}
