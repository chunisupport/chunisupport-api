package entity

import (
	"errors"
	"time"
)

var (
	ErrPlayerLatestUpdatePlayerIDInvalid      = errors.New("player latest update player_id is invalid")
	ErrPlayerLatestUpdateSchemaVersionInvalid = errors.New("player latest update schema_version is invalid")
	ErrPlayerLatestUpdateResultGzipRequired   = errors.New("player latest update result_gzip is required")
	ErrPlayerLatestUpdateSourceTimeRequired   = errors.New("player latest update source_updated_at is required")
	ErrPlayerLatestUpdateImportedTimeRequired = errors.New("player latest update imported_at is required")
	ErrPlayerLatestUpdateBodyHashRequired     = errors.New("player latest update body_hash is required")
)

// PlayerLatestUpdate はプレイヤーの最新データ登録結果を表します。
type PlayerLatestUpdate struct {
	playerID        int
	schemaVersion   int
	resultGzip      []byte
	sourceUpdatedAt time.Time
	importedAt      time.Time
	bodyHash        string
}

// NewPlayerLatestUpdate は必須項目を検証した最新データ登録結果を生成します。
func NewPlayerLatestUpdate(playerID, schemaVersion int, resultGzip []byte, sourceUpdatedAt, importedAt time.Time, bodyHash string) (*PlayerLatestUpdate, error) {
	if playerID <= 0 {
		return nil, ErrPlayerLatestUpdatePlayerIDInvalid
	}
	if schemaVersion <= 0 {
		return nil, ErrPlayerLatestUpdateSchemaVersionInvalid
	}
	if len(resultGzip) == 0 {
		return nil, ErrPlayerLatestUpdateResultGzipRequired
	}
	if sourceUpdatedAt.IsZero() {
		return nil, ErrPlayerLatestUpdateSourceTimeRequired
	}
	if importedAt.IsZero() {
		return nil, ErrPlayerLatestUpdateImportedTimeRequired
	}
	if bodyHash == "" {
		return nil, ErrPlayerLatestUpdateBodyHashRequired
	}

	return &PlayerLatestUpdate{
		playerID:        playerID,
		schemaVersion:   schemaVersion,
		resultGzip:      append([]byte(nil), resultGzip...),
		sourceUpdatedAt: sourceUpdatedAt,
		importedAt:      importedAt,
		bodyHash:        bodyHash,
	}, nil
}

// PlayerID は対象プレイヤーIDを返します。
func (u *PlayerLatestUpdate) PlayerID() int { return u.playerID }

// SchemaVersion は保存JSONのスキーマバージョンを返します。
func (u *PlayerLatestUpdate) SchemaVersion() int { return u.schemaVersion }

// ResultGzip はgzip圧縮済みの登録結果を返します。
func (u *PlayerLatestUpdate) ResultGzip() []byte { return append([]byte(nil), u.resultGzip...) }

// SourceUpdatedAt は入力データの収集日時を返します。
func (u *PlayerLatestUpdate) SourceUpdatedAt() time.Time { return u.sourceUpdatedAt }

// ImportedAt はサーバーが登録を受け付けた日時を返します。
func (u *PlayerLatestUpdate) ImportedAt() time.Time { return u.importedAt }

// BodyHash は入力本文のSHA-256ハッシュを返します。
func (u *PlayerLatestUpdate) BodyHash() string { return u.bodyHash }
