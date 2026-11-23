package entity

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
)

var (
	// ErrTeamNameInvalid はチーム名が空、または上限を超える場合のエラーです。
	ErrTeamNameInvalid = errors.New("team name is invalid")
	// ErrTeamColorInvalid はチームカラーが小文字英数字・アンダースコア・ハイフン以外を含むか、上限を超える場合のエラーです。
	ErrTeamColorInvalid = errors.New("team color is invalid")

	// チームカラーはCHUNITHM-NETのCSSクラス名（normal, gold など）由来で、players.team_color はASCII列のため英小文字等に限ります。
	teamColorPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

// PlayerTeam はプレイヤーデータで観測した所属チームです。
// チームはChuniSupport上で独立して管理せず、同じチームコードを持つプレイヤーの集まりとして扱います。
// そのためメンバーがいなくなればチームも自然に消え、チーム名やカラーもプレイヤーごとに観測した値を保持します。
type PlayerTeam struct {
	code  teamcode.TeamCode
	name  string
	color *string
}

// NewPlayerTeam は観測したチーム情報を検証して生成します。
// CHUNITHM-NETのテキストには前後の空白や改行が含まれることがあるため、チーム名の前後の空白は除去します。
// カラーが空の場合は取得できなかったものとして扱います。
func NewPlayerTeam(code teamcode.TeamCode, name, color string) (*PlayerTeam, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > TeamNameMaxLength {
		return nil, ErrTeamNameInvalid
	}
	var colorPtr *string
	if color != "" {
		if len(color) > TeamColorMaxLength || !teamColorPattern.MatchString(color) {
			return nil, ErrTeamColorInvalid
		}
		colorPtr = &color
	}
	return &PlayerTeam{code: code, name: name, color: colorPtr}, nil
}

// Code はチームを一意に識別するチームコードを返します。
func (t *PlayerTeam) Code() teamcode.TeamCode {
	return t.code
}

// Name は観測したチーム名を返します。
func (t *PlayerTeam) Name() string {
	return t.name
}

// Color はチームエンブレムのカラーを返します。取得できなかった場合はnilです。
func (t *PlayerTeam) Color() *string {
	return t.color
}
