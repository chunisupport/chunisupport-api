package app

import (
	"bytes"
	"encoding/json"
	"reflect"
	"time"

	"github.com/labstack/echo/v5"
)

var (
	timeType        = reflect.TypeFor[time.Time]()
	timePointerType = reflect.TypeFor[*time.Time]()
	jsonMarshaler   = reflect.TypeFor[json.Marshaler]()
)

// TimezoneJSONSerializer はtime.TimeだけをAPI出力用タイムゾーンへ変換します。
// 元の値は変更しないため、ドメイン・ユースケース・DBではUTCを維持できます。
type TimezoneJSONSerializer struct {
	location *time.Location
}

// NewTimezoneJSONSerializer は指定ロケーションを使用するJSONシリアライザを生成します。
func NewTimezoneJSONSerializer(location *time.Location) *TimezoneJSONSerializer {
	if location == nil {
		location = time.UTC
	}
	return &TimezoneJSONSerializer{location: location}
}

// Serialize はレスポンス値をコピーしてから時刻のLocationだけを変換します。
func (s *TimezoneJSONSerializer) Serialize(c *echo.Context, target any, indent string) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	if indent != "" {
		encoder.SetIndent("", indent)
	}
	converted := target
	if target != nil {
		converted = s.convert(reflect.ValueOf(target)).Interface()
	}
	if err := encoder.Encode(converted); err != nil {
		return err
	}
	_, err := c.Response().Write(buffer.Bytes())
	return err
}

// Deserialize は入力値を変換せず、Echo標準のデシリアライズを使用します。
func (s *TimezoneJSONSerializer) Deserialize(c *echo.Context, target any) error {
	return (echo.DefaultJSONSerializer{}).Deserialize(c, target)
}

func (s *TimezoneJSONSerializer) convert(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	if value.Type() == timeType {
		converted := value.Interface().(time.Time).In(s.location)
		return reflect.ValueOf(converted)
	}
	if value.Type() == timePointerType {
		if value.IsNil() {
			return value
		}
		converted := value.Interface().(*time.Time).In(s.location)
		return reflect.ValueOf(&converted)
	}
	if value.Type().Implements(jsonMarshaler) && value.Kind() != reflect.Map {
		return value
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return value
		}
		converted := s.convert(value.Elem())
		result := reflect.New(value.Type()).Elem()
		result.Set(converted)
		return result
	case reflect.Pointer:
		if value.IsNil() {
			return value
		}
		result := reflect.New(value.Type().Elem())
		result.Elem().Set(s.convert(value.Elem()))
		return result
	case reflect.Struct:
		result := reflect.New(value.Type()).Elem()
		result.Set(value)
		for i := range value.NumField() {
			if result.Field(i).CanSet() && value.Type().Field(i).IsExported() {
				result.Field(i).Set(s.convert(value.Field(i)))
			}
		}
		return result
	case reflect.Slice:
		if value.IsNil() {
			return value
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := range value.Len() {
			result.Index(i).Set(s.convert(value.Index(i)))
		}
		return result
	case reflect.Array:
		result := reflect.New(value.Type()).Elem()
		for i := range value.Len() {
			result.Index(i).Set(s.convert(value.Index(i)))
		}
		return result
	case reflect.Map:
		if value.IsNil() {
			return value
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			result.SetMapIndex(iterator.Key(), s.convert(iterator.Value()))
		}
		return result
	default:
		return value
	}
}
