package usecase

import "github.com/chunisupport/chunisupport-api/internal/domain/entity"

// UserDataTransferCodec は、移行集約の署名付きファイル表現を永続化層から分離します。
type UserDataTransferCodec interface {
	Encode(snapshot *entity.UserDataTransferSnapshot) ([]byte, error)
	Decode(encoded []byte) (*entity.UserDataTransferSnapshot, error)
}
