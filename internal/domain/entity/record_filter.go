package entity

import (
	"errors"
	"time"
)

// RecordFilter はユーザーが保存する譜面フィルタを表します。
type RecordFilter struct {
	id              []byte
	userID          int
	name            string
	filterValueGzip []byte
	isWorldsend     bool
	updatedAt       time.Time
}

var (
	ErrRecordFilterIDRequired              = errors.New("record filter id is required")
	ErrRecordFilterUserIDInvalid           = errors.New("record filter user_id is invalid")
	ErrRecordFilterNameRequired            = errors.New("record filter name is required")
	ErrRecordFilterFilterValueGzipRequired = errors.New("record filter filter_value_gzip is required")
)

// NewRecordFilter は必須項目を検証した RecordFilter を生成します。
func NewRecordFilter(id []byte, userID int, name string, filterValueGzip []byte, isWorldsend bool) (*RecordFilter, error) {
	return RestoreRecordFilter(id, userID, name, filterValueGzip, isWorldsend, time.Time{})
}

// RestoreRecordFilter は永続化済みデータから RecordFilter を復元します。
func RestoreRecordFilter(id []byte, userID int, name string, filterValueGzip []byte, isWorldsend bool, updatedAt time.Time) (*RecordFilter, error) {
	if len(id) == 0 {
		return nil, ErrRecordFilterIDRequired
	}
	if userID <= 0 {
		return nil, ErrRecordFilterUserIDInvalid
	}
	if name == "" {
		return nil, ErrRecordFilterNameRequired
	}
	if len(filterValueGzip) == 0 {
		return nil, ErrRecordFilterFilterValueGzipRequired
	}

	return &RecordFilter{
		id:              append([]byte(nil), id...),
		userID:          userID,
		name:            name,
		filterValueGzip: append([]byte(nil), filterValueGzip...),
		isWorldsend:     isWorldsend,
		updatedAt:       updatedAt,
	}, nil
}

// ID はレコードフィルタIDを返します。
func (r *RecordFilter) ID() []byte {
	return append([]byte(nil), r.id...)
}

// UserID は所有ユーザーIDを返します。
func (r *RecordFilter) UserID() int {
	return r.userID
}

// Name はフィルタ名を返します。
func (r *RecordFilter) Name() string {
	return r.name
}

// FilterValueGzip は gzip 圧縮済みフィルタ条件を返します。
func (r *RecordFilter) FilterValueGzip() []byte {
	return append([]byte(nil), r.filterValueGzip...)
}

// IsWorldsend はワールズエンド用フィルタかどうかを返します。
func (r *RecordFilter) IsWorldsend() bool {
	return r.isWorldsend
}

// UpdatedAt は最終更新日時を返します。
func (r *RecordFilter) UpdatedAt() time.Time {
	return r.updatedAt
}

// ChangeName は保存済みフィルタ名を変更します。
func (r *RecordFilter) ChangeName(name string) error {
	if name == "" {
		return ErrRecordFilterNameRequired
	}
	r.name = name
	return nil
}

// ChangeFilterValueGzip は gzip 圧縮済みフィルタ条件を差し替えます。
func (r *RecordFilter) ChangeFilterValueGzip(filterValueGzip []byte) error {
	if len(filterValueGzip) == 0 {
		return ErrRecordFilterFilterValueGzipRequired
	}
	r.filterValueGzip = append([]byte(nil), filterValueGzip...)
	return nil
}

// ChangeWorldsend はワールズエンド用フィルタかどうかを変更します。
func (r *RecordFilter) ChangeWorldsend(isWorldsend bool) {
	r.isWorldsend = isWorldsend
}
