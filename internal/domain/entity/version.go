package entity

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	VersionNamePrefix    = "CHUNITHM "
	VersionNameMaxLength = 50
	// VersionShortNameMaxLength は表示幅を抑えるための超ショート名の最大文字数です。
	VersionShortNameMaxLength = 10
)

var ErrInvalidVersion = errors.New("invalid version")

// Version はCHUNITHMの稼働バージョンを表します。
type Version struct {
	ID   int
	Name string
	// ShortName は一覧などで表示幅を最小限にするための超ショート名です（例: CRY+）。
	ShortName  string
	ReleasedAt time.Time
}

// NewVersion は保存可能なバージョンを生成します。
func NewVersion(name, shortName string, releasedAt time.Time) (*Version, error) {
	trimmedName := strings.TrimSpace(name)
	if err := validateVersionName(trimmedName); err != nil {
		return nil, err
	}
	trimmedShortName := strings.TrimSpace(shortName)
	if err := validateVersionShortName(trimmedShortName); err != nil {
		return nil, err
	}
	if releasedAt.IsZero() {
		return nil, ErrInvalidVersion
	}

	return &Version{
		Name:       trimmedName,
		ShortName:  trimmedShortName,
		ReleasedAt: normalizeVersionDate(releasedAt),
	}, nil
}

// Rename はバージョン名と超ショート名を検証して変更します。
// 片方だけが変更された中途半端な状態を作らないよう、両方の検証後にまとめて反映します。
func (v *Version) Rename(name, shortName string) error {
	trimmedName := strings.TrimSpace(name)
	if err := validateVersionName(trimmedName); err != nil {
		return err
	}
	trimmedShortName := strings.TrimSpace(shortName)
	if err := validateVersionShortName(trimmedShortName); err != nil {
		return err
	}
	v.Name = trimmedName
	v.ShortName = trimmedShortName
	return nil
}

func validateVersionName(name string) error {
	if !strings.HasPrefix(name, VersionNamePrefix) || utf8.RuneCountInString(name) > VersionNameMaxLength {
		return ErrInvalidVersion
	}
	return nil
}

func validateVersionShortName(shortName string) error {
	if shortName == "" || utf8.RuneCountInString(shortName) > VersionShortNameMaxLength {
		return ErrInvalidVersion
	}
	return nil
}

func normalizeVersionDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
