package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
)

// teamRepository は TeamRepository のMySQL実装です。
type teamRepository struct{}

// NewTeamRepository は TeamRepository の実装を生成します。
func NewTeamRepository() repository.TeamRepository {
	return &teamRepository{}
}

const selectTeamQuery = `
	SELECT id, code, name, color, data_collected_at, created_at, updated_at
	FROM teams
	WHERE code = ?
`

func (r *teamRepository) FindByCode(ctx context.Context, exec repository.Executor, code teamcode.TeamCode) (*entity.Team, error) {
	return findTeam(ctx, exec, selectTeamQuery, code)
}

func (r *teamRepository) FindByCodeForUpdate(ctx context.Context, exec repository.Executor, code teamcode.TeamCode) (*entity.Team, error) {
	return findTeam(ctx, exec, selectTeamQuery+" FOR UPDATE", code)
}

func findTeam(ctx context.Context, exec repository.Executor, query string, code teamcode.TeamCode) (*entity.Team, error) {
	var model models.TeamModel
	if err := exec.GetContext(ctx, &model, query, code.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrTeamNotFound, err)
		}
		return nil, err
	}
	return model.ToEntity()
}

func (r *teamRepository) Save(ctx context.Context, exec repository.Executor, team *entity.Team) error {
	if team.ID == 0 {
		return insertTeam(ctx, exec, team)
	}
	_, err := exec.ExecContext(ctx, `
		UPDATE teams
		SET name = ?, color = ?, data_collected_at = ?, updated_at = ?
		WHERE id = ?
	`, team.Name, team.Color, team.DataCollectedAt, team.UpdatedAt, team.ID)
	return err
}

// insertTeam は新しいチームをINSERTします。
// 既存チームへのINSERTでAUTO_INCREMENTを浪費しないよう、呼び出し側は事前検索で見つからなかった場合だけ使います。
// 並行登録で一意制約に違反した場合は ErrTeamAlreadyExists を返し、呼び出し側でロック付きの再検索へ切り替えさせます。
func insertTeam(ctx context.Context, exec repository.Executor, team *entity.Team) error {
	result, err := exec.ExecContext(ctx, `
		INSERT INTO teams (code, name, color, data_collected_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, team.Code.String(), team.Name, team.Color, team.DataCollectedAt, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		if isMySQLDuplicateEntryForKey(err, "uq_teams_code") {
			return fmt.Errorf("%w: %w", repository.ErrTeamAlreadyExists, err)
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	team.ID = int(id)
	return nil
}
