package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

var (
	// ErrUsernameForbidden はユーザー名が登録禁止語に該当する場合に返されます。
	ErrUsernameForbidden = errors.New("username is forbidden")
)

// UsernamePolicy は新規登録に利用可能なユーザー名かを判定します。
type UsernamePolicy interface {
	Validate(userName username.UserName) error
}

type forbiddenUsernamePolicy struct {
	exact    map[string]struct{}
	contains []string
}

// NewForbiddenUsernamePolicy は完全一致と部分一致の禁止語からポリシーを生成します。
func NewForbiddenUsernamePolicy(exactWords, containsWords []string) (UsernamePolicy, error) {
	exact, err := validatedForbiddenWords(exactWords)
	if err != nil {
		return nil, fmt.Errorf("invalid exact forbidden word: %w", err)
	}
	containsSet, err := validatedForbiddenWords(containsWords)
	if err != nil {
		return nil, fmt.Errorf("invalid contains forbidden word: %w", err)
	}

	contains := make([]string, 0, len(containsSet))
	for word := range containsSet {
		contains = append(contains, word)
	}

	return &forbiddenUsernamePolicy{
		exact:    exact,
		contains: contains,
	}, nil
}

// Validate は禁止語に該当するユーザー名を拒否します。
func (p *forbiddenUsernamePolicy) Validate(userName username.UserName) error {
	value := userName.String()
	if _, forbidden := p.exact[value]; forbidden {
		return ErrUsernameForbidden
	}
	for _, word := range p.contains {
		if strings.Contains(value, word) {
			return ErrUsernameForbidden
		}
	}
	return nil
}

func validatedForbiddenWords(words []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(words))
	for _, word := range words {
		if word == "" {
			return nil, errors.New("forbidden word must not be empty")
		}
		result[word] = struct{}{}
	}
	return result, nil
}

var _ UsernamePolicy = (*forbiddenUsernamePolicy)(nil)
