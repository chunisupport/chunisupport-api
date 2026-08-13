package usecase

import "errors"

var (
	ErrDataTransferPlayerNotFound      = errors.New("data transfer player not found")
	ErrDataTransferInvalidFile         = errors.New("invalid data transfer file")
	ErrDataTransferInvalidSignature    = errors.New("invalid data transfer signature")
	ErrDataTransferUnsupportedSchema   = errors.New("unsupported data transfer schema")
	ErrDataTransferInvalidData         = errors.New("invalid data transfer data")
	ErrDataTransferUnresolvedReference = errors.New("unresolved data transfer reference")
	ErrDataTransferDestinationNotEmpty = errors.New("data transfer destination is not empty")
	ErrDataTransferPayloadTooLarge     = errors.New("data transfer payload is too large")
)
