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
)

var ErrInvalidVersion = errors.New("invalid version")

// Version はCHUNITHMの稼働バージョンを表します。
type Version struct {
	ID         int
	Name       string
	ReleasedAt time.Time
}

// NewVersion は保存可能なバージョンを生成します。
func NewVersion(name string, releasedAt time.Time) (*Version, error) {
	trimmedName := strings.TrimSpace(name)
	if err := validateVersionName(trimmedName); err != nil {
		return nil, err
	}
	if releasedAt.IsZero() {
		return nil, ErrInvalidVersion
	}

	return &Version{
		Name:       trimmedName,
		ReleasedAt: normalizeVersionDate(releasedAt),
	}, nil
}

// Rename はバージョン名を検証して変更します。
func (v *Version) Rename(name string) error {
	trimmedName := strings.TrimSpace(name)
	if err := validateVersionName(trimmedName); err != nil {
		return err
	}
	v.Name = trimmedName
	return nil
}

func validateVersionName(name string) error {
	if !strings.HasPrefix(name, VersionNamePrefix) || utf8.RuneCountInString(name) > VersionNameMaxLength {
		return ErrInvalidVersion
	}
	return nil
}

func normalizeVersionDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
