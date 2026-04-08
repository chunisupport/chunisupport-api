package usecase

import "encoding/json"

func marshalPlayerDataPayload(payload *PlayerDataPayload) ([]byte, error) {
	return json.Marshal(payload)
}

func unmarshalPlayerDataPayload(data []byte, payload *PlayerDataPayload) error {
	return json.Unmarshal(data, payload)
}
