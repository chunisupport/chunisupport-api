package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

// playerRepository は PlayerRepository の実装です。
type playerRepository struct {
	db *sqlx.DB
}

// NewPlayerRepository は PlayerRepository の実装を生成します。
func NewPlayerRepository(db *sqlx.DB) repository.PlayerRepository {
	return &playerRepository{db: db}
}

// FindByID はIDでプレイヤーを検索します。関連する全てのフィールドを含むエンティティを返します。
func (r *playerRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	query := `
		SELECT
			id, user_id, player_name, player_level,
			official_player_rating, calculated_player_rating, new_average_rating, best_average_rating,
			class_emblem_id, class_emblem_base_id, last_played_at,
			overpower_value,
			created_at, updated_at
		FROM players
		WHERE id = ?
	`
	var playerModel models.PlayerModel
	if err := exec.GetContext(ctx, &playerModel, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrPlayerNotFound, err)
		}
		return nil, err
	}
	return playerModel.ToEntity()
}

func (r *playerRepository) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	query := `
		SELECT
			p.id, p.user_id, p.player_name, p.player_level,
			p.official_player_rating, p.calculated_player_rating, p.new_average_rating, p.best_average_rating,
			p.class_emblem_id, p.class_emblem_base_id, p.last_played_at,
			p.overpower_value,
			p.created_at, p.updated_at,
			ph.slot AS honor_slot,
			h.name AS honor_name,
			ht.name AS honor_type_name,
			NULLIF(h.image_url, '') AS honor_image_url
		FROM players p
		LEFT JOIN player_honors ph ON ph.player_id = p.id
		LEFT JOIN honors h ON ph.honor_id = h.id
		LEFT JOIN honor_types ht ON h.honor_type_id = ht.id
		WHERE p.id = ?
		ORDER BY ph.slot
	`

	rows, err := exec.QueryxContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type playerWithHonorRow struct {
		models.PlayerModel
		HonorSlot     *int    `db:"honor_slot"`
		HonorName     *string `db:"honor_name"`
		HonorTypeName *string `db:"honor_type_name"`
		HonorImageURL *string `db:"honor_image_url"`
	}

	var result *repository.PlayerWithHonors
	for rows.Next() {
		var row playerWithHonorRow
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}

		if result == nil {
			player, err := row.PlayerModel.ToEntity()
			if err != nil {
				return nil, err
			}
			result = &repository.PlayerWithHonors{
				Player: player,
				Honors: []*entity.PlayerHonor{},
			}
		}

		if row.HonorSlot == nil || row.HonorName == nil || row.HonorTypeName == nil {
			continue
		}

		result.Honors = append(result.Honors, &entity.PlayerHonor{
			Slot:     *row.HonorSlot,
			Name:     *row.HonorName,
			TypeName: *row.HonorTypeName,
			ImageURL: row.HonorImageURL,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		return nil, repository.ErrPlayerNotFound
	}

	return result, nil
}

// FindHonorsByPlayerID はプレイヤーIDで称号情報を取得します。スロット順（1,2,3）でソートされます。
func (r *playerRepository) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerHonor, error) {
	query := `
		SELECT ph.slot, h.name, ht.name AS type_name, NULLIF(h.image_url, '') AS image_url
		FROM player_honors ph
		INNER JOIN honors h ON ph.honor_id = h.id
		INNER JOIN honor_types ht ON h.honor_type_id = ht.id
		WHERE ph.player_id = ?
		ORDER BY ph.slot
	`
	rows, err := exec.QueryxContext(ctx, query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var honors []*entity.PlayerHonor
	for rows.Next() {
		var h struct {
			Slot     int     `db:"slot"`
			Name     string  `db:"name"`
			TypeName string  `db:"type_name"`
			ImageURL *string `db:"image_url"`
		}
		if err := rows.StructScan(&h); err != nil {
			return nil, err
		}
		honors = append(honors, &entity.PlayerHonor{
			Slot:     h.Slot,
			Name:     h.Name,
			TypeName: h.TypeName,
			ImageURL: h.ImageURL,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return honors, nil
}

// UpdateCalculatedRatings はプレイヤーの計算されたレーティング情報を更新します。
func (r *playerRepository) UpdateCalculatedRatings(ctx context.Context, exec repository.Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error {
	query := `
		UPDATE players
		SET calculated_player_rating = ?,
		    best_average_rating = ?,
		    new_average_rating = ?
		WHERE id = ?
	`
	_, err := exec.ExecContext(ctx, query, calculatedRating, bestAverage, newAverage, playerID)
	return err
}

// FindByUserID はユーザーIDでプレイヤーを検索します。見つからない場合は(nil, nil)を返します。
func (r *playerRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	query := `
		SELECT
			id, user_id, player_name, player_level,
			official_player_rating, calculated_player_rating, new_average_rating, best_average_rating,
			class_emblem_id, class_emblem_base_id, last_played_at,
			overpower_value,
			created_at, updated_at
		FROM players
		WHERE user_id = ?
	`
	var playerModel models.PlayerModel
	if err := exec.GetContext(ctx, &playerModel, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return playerModel.ToEntity()
}

// Save はプレイヤー情報を保存します（ID=0の場合はINSERT、それ以外はUPDATE）。
// INSERT時は player が user_id や player_name、player_level など必須カラムを保持している前提です。
// INSERTの場合、playerのIDフィールドが更新されます。
func (r *playerRepository) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	if player.ID == 0 {
		return r.insert(ctx, exec, player)
	}
	return r.update(ctx, exec, player)
}

// insert は新しいプレイヤーをINSERTします。
// Saveからのみ呼び出され、INSERTに必要なカラムが満たされていることを前提にします。
func (r *playerRepository) insert(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	query := `
		INSERT INTO players (
			user_id, player_name, player_level, official_player_rating,
			class_emblem_id, class_emblem_base_id, last_played_at,
			overpower_value, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := exec.ExecContext(ctx, query,
		player.UserID, player.Name.String(), player.Level, player.OfficialRating,
		player.ClassEmblemID, player.ClassEmblemBaseID, player.LastPlayedAt,
		player.OverpowerValue, player.CreatedAt, player.UpdatedAt,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	player.ID = int(id)
	return nil
}

// update は既存のプレイヤーをUPDATEします。
// Saveからのみ呼び出され、既存レコード（player.ID != 0）の更新のみを担当します。
func (r *playerRepository) update(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	query := `
		UPDATE players
		SET player_name = ?,
		    player_level = ?,
		    official_player_rating = ?,
		    class_emblem_id = ?,
		    class_emblem_base_id = ?,
		    last_played_at = ?,
		    overpower_value = ?,
		    updated_at = ?
		WHERE id = ?
	`
	_, err := exec.ExecContext(ctx, query,
		player.Name.String(), player.Level, player.OfficialRating,
		player.ClassEmblemID, player.ClassEmblemBaseID, player.LastPlayedAt,
		player.OverpowerValue, player.UpdatedAt,
		player.ID,
	)
	return err
}

// DeleteByUserID はユーザーに紐づくプレイヤーを削除します。関連データはON DELETE CASCADEで削除されます。
func (r *playerRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	query := `DELETE FROM players WHERE user_id = ?`
	_, err := exec.ExecContext(ctx, query, userID)
	return err
}
