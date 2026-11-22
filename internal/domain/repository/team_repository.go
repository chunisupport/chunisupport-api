package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
)

// TeamRepository はプレイヤーデータから収集したチームの永続化を扱うリポジトリです。
type TeamRepository interface {
	// FindByCode はチームコードでチームを検索します。対象が存在しない場合は ErrTeamNotFound を返します。
	// 存在しない行を FOR UPDATE で検索するとギャップロックにより並行登録がデッドロックするため、新規登録前の存在確認にはこちらを使います。
	FindByCode(ctx context.Context, exec Executor, code teamcode.TeamCode) (*entity.Team, error)
	// FindByCodeForUpdate はチームコードでチームを検索し、更新用に行ロックします。対象が存在しない場合は ErrTeamNotFound を返します。
	FindByCodeForUpdate(ctx context.Context, exec Executor, code teamcode.TeamCode) (*entity.Team, error)
	// Save はチームを保存します。ID=0 の場合はINSERTして team.ID を設定し、それ以外は名前・カラー・取得日時を更新します。
	// 同じチームコードが並行して登録済みの場合は ErrTeamAlreadyExists を返します。
	Save(ctx context.Context, exec Executor, team *entity.Team) error
}
