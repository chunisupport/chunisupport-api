// Package goalgroupname は目標グループ名の値オブジェクトを提供します。
package goalgroupname

import (
	"errors"
	"strings"
	"unicode"
)

const MaxLength = 30

var ErrInvalidGoalGroupName = errors.New("invalid goal group name")

// GoalGroupName は検証済みの目標グループ名です。
type GoalGroupName struct {
	value string
}

// NewGoalGroupName は前後の空白を除去し、表示可能な1〜30文字の名前を生成します。
func NewGoalGroupName(value string) (GoalGroupName, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len([]rune(trimmed)) > MaxLength {
		return GoalGroupName{}, ErrInvalidGoalGroupName
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return GoalGroupName{}, ErrInvalidGoalGroupName
		}
	}
	return GoalGroupName{value: trimmed}, nil
}

// String はグループ名を返します。
func (n GoalGroupName) String() string {
	return n.value
}
