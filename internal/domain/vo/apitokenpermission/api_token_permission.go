// Package apitokenpermission はAPIトークン権限の値オブジェクトを提供します。
package apitokenpermission

import "errors"

const (
	// Read は参照系APIだけを利用できる権限です。
	Read APITokenPermission = "read"
	// ReadWrite は参照系・更新系APIを利用できる権限です。
	ReadWrite APITokenPermission = "read_write"
)

var ErrInvalidAPITokenPermission = errors.New("invalid API token permission")

// APITokenPermission は検証済みのAPIトークン権限です。
type APITokenPermission string

// NewAPITokenPermission は許可されたAPIトークン権限を生成します。
func NewAPITokenPermission(value string) (APITokenPermission, error) {
	switch APITokenPermission(value) {
	case Read, ReadWrite:
		return APITokenPermission(value), nil
	default:
		return "", ErrInvalidAPITokenPermission
	}
}

// String はAPIトークン権限の外部表現を返します。
func (p APITokenPermission) String() string {
	return string(p)
}

// CanWrite は更新系APIを利用できる権限かを返します。
func (p APITokenPermission) CanWrite() bool {
	return p == ReadWrite
}
