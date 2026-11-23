package usecase

import (
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
)

// playerTeamObservation はプレイヤーデータから読み取った所属チームの状態です。
type playerTeamObservation struct {
	// resolved は所属状態を確定できたかを表します。false の場合は保存済みの所属を維持します。
	resolved bool
	// team は観測した所属チームです。resolved が true で nil の場合はチーム未所属を表します。
	team *entity.PlayerTeam
}

// newPlayerTeamObservation はプレイヤーデータのチーム情報を検証し、所属チームの状態へ変換します。
// チーム名は変更できるため、チームコードを取得できなかった場合は所属先を特定できないものとして保存済みの所属を維持します。
// チームコードがある場合は、取得処理の不具合を早期に検知するため形式不正を検証エラーにします。
func newPlayerTeamObservation(payload PlayerDataTeamPayload) (playerTeamObservation, error) {
	if payload.Code == "" {
		return playerTeamObservation{resolved: payload.Name == ""}, nil
	}
	code, err := teamcode.NewTeamCode(payload.Code)
	if err != nil {
		return playerTeamObservation{}, &PlayerDataValidationError{Field: "team.code", Message: err.Error(), Err: err}
	}
	team, err := entity.NewPlayerTeam(code, payload.Name, payload.Color)
	if err != nil {
		field := "team.name"
		if errors.Is(err, entity.ErrTeamColorInvalid) {
			field = "team.color"
		}
		return playerTeamObservation{}, &PlayerDataValidationError{Field: field, Message: err.Error(), Err: err}
	}
	return playerTeamObservation{resolved: true, team: team}, nil
}
