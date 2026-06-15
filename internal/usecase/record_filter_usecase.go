package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/google/uuid"
)

const (
	RecordFilterTypeStandard  = "standard"
	RecordFilterTypeWorldsend = "worldsend"
)

// RecordFilterUsecase は保存済み譜面フィルタを扱うユースケースです。
type RecordFilterUsecase interface {
	List(ctx context.Context, userID int, filterType string) ([]*RecordFilterOutput, error)
	Create(ctx context.Context, userID int, input *RecordFilterInput) (*RecordFilterOutput, error)
	Update(ctx context.Context, userID int, id string, input *RecordFilterInput) (*RecordFilterOutput, error)
	Delete(ctx context.Context, userID int, id string) error
}

// RecordFilterInput は保存済み譜面フィルタの作成・更新入力です。
type RecordFilterInput struct {
	Name          string
	FilterType    string
	SchemaVersion int
	Filter        json.RawMessage
}

// RecordFilterOutput は保存済み譜面フィルタAPI向けの出力です。
type RecordFilterOutput struct {
	ID            string
	Name          string
	FilterType    string
	SchemaVersion int
	Filter        json.RawMessage
	CreatedAt     string
	UpdatedAt     string
}

type recordFilterUsecase struct {
	repo repository.RecordFilterRepository
}

type recordFilterPayload struct {
	SchemaVersion int             `json:"schema_version"`
	Filter        json.RawMessage `json:"filter"`
}

type validatedRecordFilterInput struct {
	name            string
	isWorldsend     bool
	filterValueGzip []byte
}

// NewRecordFilterUsecase は RecordFilterUsecase を生成します。
func NewRecordFilterUsecase(repo repository.RecordFilterRepository) RecordFilterUsecase {
	return &recordFilterUsecase{repo: repo}
}

func (u *recordFilterUsecase) List(ctx context.Context, userID int, filterType string) ([]*RecordFilterOutput, error) {
	var wantWorldsend *bool
	if filterType != "" {
		isWorldsend, err := recordFilterTypeToWorldsend(filterType)
		if err != nil {
			return nil, err
		}
		wantWorldsend = &isWorldsend
	}

	filters, err := u.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	outs := make([]*RecordFilterOutput, 0, len(filters))
	for _, filter := range filters {
		if wantWorldsend != nil && filter.IsWorldsend() != *wantWorldsend {
			continue
		}
		out, err := toRecordFilterOutput(filter)
		if err != nil {
			return nil, err
		}
		outs = append(outs, out)
	}
	return outs, nil
}

func (u *recordFilterUsecase) Create(ctx context.Context, userID int, input *RecordFilterInput) (*RecordFilterOutput, error) {
	validated, err := validateRecordFilterInput(input)
	if err != nil {
		return nil, err
	}

	count, err := u.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= info.RecordFilterMaxPerUser {
		return nil, ErrRecordFilterLimitExceeded
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, ErrInternalError
	}
	idBytes, err := id.MarshalBinary()
	if err != nil {
		return nil, ErrInternalError
	}

	filter, err := entity.NewRecordFilter(idBytes, userID, validated.name, validated.filterValueGzip, validated.isWorldsend)
	if err != nil {
		return nil, err
	}
	if err := u.repo.Save(ctx, filter); err != nil {
		return nil, err
	}

	saved, err := u.repo.FindByIDAndUserID(ctx, idBytes, userID)
	if err != nil {
		return nil, err
	}
	return toRecordFilterOutput(saved)
}

func (u *recordFilterUsecase) Update(ctx context.Context, userID int, id string, input *RecordFilterInput) (*RecordFilterOutput, error) {
	idBytes, err := parseRecordFilterID(id)
	if err != nil {
		return nil, err
	}
	validated, err := validateRecordFilterInput(input)
	if err != nil {
		return nil, err
	}

	filter, err := u.repo.FindByIDAndUserID(ctx, idBytes, userID)
	if err != nil {
		if errors.Is(err, repository.ErrRecordFilterNotFound) {
			return nil, ErrRecordFilterNotFound
		}
		return nil, err
	}
	if err := filter.ChangeName(validated.name); err != nil {
		return nil, err
	}
	if err := filter.ChangeFilterValueGzip(validated.filterValueGzip); err != nil {
		return nil, err
	}
	filter.ChangeWorldsend(validated.isWorldsend)

	if err := u.repo.Save(ctx, filter); err != nil {
		if errors.Is(err, repository.ErrRecordFilterNotFound) {
			return nil, ErrRecordFilterNotFound
		}
		return nil, err
	}

	saved, err := u.repo.FindByIDAndUserID(ctx, idBytes, userID)
	if err != nil {
		return nil, err
	}
	return toRecordFilterOutput(saved)
}

func (u *recordFilterUsecase) Delete(ctx context.Context, userID int, id string) error {
	idBytes, err := parseRecordFilterID(id)
	if err != nil {
		return err
	}
	err = u.repo.DeleteByIDAndUserID(ctx, idBytes, userID)
	if errors.Is(err, repository.ErrRecordFilterNotFound) {
		return ErrRecordFilterNotFound
	}
	return err
}

func validateRecordFilterInput(input *RecordFilterInput) (*validatedRecordFilterInput, error) {
	if input == nil {
		return nil, ErrInvalidRecordFilterInput
	}

	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > info.RecordFilterNameMaxLength || hasControlCharacterForRecordFilter(name) {
		return nil, ErrInvalidRecordFilterInput
	}

	isWorldsend, err := recordFilterTypeToWorldsend(input.FilterType)
	if err != nil {
		return nil, err
	}
	if input.SchemaVersion <= 0 {
		return nil, ErrInvalidRecordFilterInput
	}

	canonicalFilter, err := compactRecordFilterObject(input.Filter)
	if err != nil {
		return nil, ErrInvalidRecordFilterInput
	}

	payload, err := json.Marshal(recordFilterPayload{
		SchemaVersion: input.SchemaVersion,
		Filter:        canonicalFilter,
	})
	if err != nil {
		return nil, ErrInvalidRecordFilterInput
	}
	if len(payload) > info.RecordFilterMaxPayloadBytes {
		return nil, ErrInvalidRecordFilterInput
	}

	filterValueGzip, err := gzipBytes(payload)
	if err != nil {
		return nil, err
	}
	return &validatedRecordFilterInput{name: name, isWorldsend: isWorldsend, filterValueGzip: filterValueGzip}, nil
}

func compactRecordFilterObject(value json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' || !json.Valid(trimmed) {
		return nil, ErrInvalidRecordFilterInput
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, trimmed); err != nil {
		return nil, ErrInvalidRecordFilterInput
	}
	return buf.Bytes(), nil
}

func recordFilterTypeToWorldsend(filterType string) (bool, error) {
	switch filterType {
	case RecordFilterTypeStandard:
		return false, nil
	case RecordFilterTypeWorldsend:
		return true, nil
	default:
		return false, ErrInvalidRecordFilterInput
	}
}

func recordFilterTypeFromWorldsend(isWorldsend bool) string {
	if isWorldsend {
		return RecordFilterTypeWorldsend
	}
	return RecordFilterTypeStandard
}

func parseRecordFilterID(id string) ([]byte, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidRecordFilterID
	}
	idBytes, err := parsed.MarshalBinary()
	if err != nil {
		return nil, ErrInvalidRecordFilterID
	}
	return idBytes, nil
}

func toRecordFilterOutput(filter *entity.RecordFilter) (*RecordFilterOutput, error) {
	id, err := uuid.FromBytes(filter.ID())
	if err != nil {
		return nil, ErrInvalidRecordFilterID
	}
	payloadBytes, err := gunzipBytes(filter.FilterValueGzip(), info.RecordFilterMaxPayloadBytes)
	if err != nil {
		return nil, err
	}

	var payload recordFilterPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}
	if payload.SchemaVersion <= 0 || len(payload.Filter) == 0 {
		return nil, ErrInvalidRecordFilterInput
	}

	return &RecordFilterOutput{
		ID:            id.String(),
		Name:          filter.Name(),
		FilterType:    recordFilterTypeFromWorldsend(filter.IsWorldsend()),
		SchemaVersion: payload.SchemaVersion,
		Filter:        payload.Filter,
		CreatedAt:     formatRecordFilterTime(filter.CreatedAt()),
		UpdatedAt:     formatRecordFilterTime(filter.UpdatedAt()),
	}, nil
}

func gzipBytes(value []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(value); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gunzipBytes(value []byte, maxBytes int) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(value))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	raw, err := io.ReadAll(io.LimitReader(reader, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBytes {
		return nil, ErrInvalidRecordFilterInput
	}
	return raw, nil
}

func formatRecordFilterTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func hasControlCharacterForRecordFilter(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
