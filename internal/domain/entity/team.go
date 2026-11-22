package entity

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
)

var (
	// ErrTeamNameInvalid はチーム名が空、または上限を超える場合のエラーです。
	ErrTeamNameInvalid = errors.New("team name is invalid")
	// ErrTeamColorInvalid はチームカラーが小文字英数字・アンダースコア・ハイフン以外を含むか、上限を超える場合のエラーです。
	ErrTeamColorInvalid = errors.New("team color is invalid")

	// チームカラーはCHUNITHM-NETのCSSクラス名（normal, gold など）由来で、teams.color はASCII列のため英小文字等に限ります。
	teamColorPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

// Team はプレイヤーデータから収集したCHUNITHMのチームです。
// チームはチームコードで一意に識別し、名前とカラーはプレイヤーデータを取り込むたびに最新の観測値へ更新します。
// 所属メンバーは players.team_id から導出するため、この集約には含めません。
type Team struct {
	ID              int
	Code            teamcode.TeamCode
	Name            string
	Color           *string   // チームエンブレムのカラー。取得できなかった場合はnil
	DataCollectedAt time.Time // 名前とカラーを観測したプレイヤーデータの取得日時
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewTeam はプレイヤーデータで観測したチーム情報を検証して生成します。
// CHUNITHM-NETのテキストには前後の空白や改行が含まれることがあるため、チーム名の前後の空白は除去します。
// teams.data_collected_at は秒精度で、MySQLは小数秒を四捨五入して保存するため、取得日時は秒未満を切り捨てて
// 保存値と同じ精度で新旧を比較します。秒未満で前後する観測は同時刻とみなし、先に保存した観測を優先します。
func NewTeam(code teamcode.TeamCode, name, color string, collectedAt time.Time) (*Team, error) {
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
	now := time.Now().UTC()
	return &Team{
		Code:            code,
		Name:            name,
		Color:           colorPtr,
		DataCollectedAt: collectedAt.Truncate(time.Second),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// ApplyObservation は同じチームを新たに観測した結果を反映し、保存が必要かを返します。
// 複数プレイヤーのデータが前後して届くため、保存済みより新しい取得日時の観測だけを反映し、古いデータで名前を巻き戻さないようにします。
// 名前とカラーが同じでも取得日時は進め、その後に届いた古い観測を確実に退けられるようにします。
func (t *Team) ApplyObservation(observed *Team) bool {
	if !observed.DataCollectedAt.After(t.DataCollectedAt) {
		return false
	}
	t.Name = observed.Name
	t.Color = observed.Color
	t.DataCollectedAt = observed.DataCollectedAt
	t.UpdatedAt = time.Now().UTC()
	return true
}
