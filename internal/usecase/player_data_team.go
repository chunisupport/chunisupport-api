package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
)

// playerTeamObservation はプレイヤーデータから読み取った所属チームの状態です。
type playerTeamObservation struct {
	// resolved は所属状態を確定できたかを表します。false の場合は保存済みの所属を維持します。
	resolved bool
	// team は観測した所属チームです。resolved が true で nil の場合はチーム未所属を表します。
	team *entity.Team
}

// newPlayerTeamObservation はプレイヤーデータのチーム情報を検証し、所属チームの状態へ変換します。
// チーム名は変更できるため、チームコードを取得できなかった場合は所属先を特定できないものとして保存済みの所属を維持します。
// チームコードがある場合は、取得処理の不具合を早期に検知するため形式不正を検証エラーにします。
func newPlayerTeamObservation(payload PlayerDataTeamPayload, collectedAt time.Time) (playerTeamObservation, error) {
	if payload.Code == "" {
		return playerTeamObservation{resolved: payload.Name == ""}, nil
	}
	code, err := teamcode.NewTeamCode(payload.Code)
	if err != nil {
		return playerTeamObservation{}, &PlayerDataValidationError{Field: "team.code", Message: err.Error(), Err: err}
	}
	team, err := entity.NewTeam(code, payload.Name, payload.Color, collectedAt)
	if err != nil {
		field := "team.name"
		if errors.Is(err, entity.ErrTeamColorInvalid) {
			field = "team.color"
		}
		return playerTeamObservation{}, &PlayerDataValidationError{Field: field, Message: err.Error(), Err: err}
	}
	return playerTeamObservation{resolved: true, team: team}, nil
}

// resolvePlayerTeamID は観測した所属チームを保存し、プレイヤーに設定するチームIDを返します。
func (us *playerDataUsecase) resolvePlayerTeamID(ctx context.Context, tx repository.Executor, observation playerTeamObservation) (*int, error) {
	if observation.team == nil {
		return nil, nil
	}
	teamID, err := us.ensureTeam(ctx, tx, observation.team)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure team: %w", err)
	}
	return &teamID, nil
}

// ensureTeam は観測したチームを登録、または登録済みのチームへ反映し、チームIDを返します。
// 存在しない行を FOR UPDATE で検索すると、同じチームを並行登録したトランザクション同士がギャップロックでデッドロックするため、
// 先にロックなしで存在を確認し、未登録ならINSERTします。並行登録で一意制約に違反した場合だけ、登録済みの行をロックして更新します。
func (us *playerDataUsecase) ensureTeam(ctx context.Context, tx repository.Executor, observed *entity.Team) (int, error) {
	_, err := us.teamRepo.FindByCode(ctx, tx, observed.Code)
	if err == nil {
		return us.applyTeamObservation(ctx, tx, observed)
	}
	if !errors.Is(err, repository.ErrTeamNotFound) {
		return 0, err
	}
	err = us.teamRepo.Save(ctx, tx, observed)
	if err == nil {
		return observed.ID, nil
	}
	if !errors.Is(err, repository.ErrTeamAlreadyExists) {
		return 0, err
	}
	return us.applyTeamObservation(ctx, tx, observed)
}

// applyTeamObservation は登録済みのチームを行ロックし、より新しい観測であれば名前とカラーを更新します。
func (us *playerDataUsecase) applyTeamObservation(ctx context.Context, tx repository.Executor, observed *entity.Team) (int, error) {
	team, err := us.teamRepo.FindByCodeForUpdate(ctx, tx, observed.Code)
	if err != nil {
		return 0, err
	}
	if team.ApplyObservation(observed) {
		if err := us.teamRepo.Save(ctx, tx, team); err != nil {
			return 0, err
		}
	}
	return team.ID, nil
}
