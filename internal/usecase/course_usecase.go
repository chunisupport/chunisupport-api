package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrInvalidCourseInput = errors.New("invalid course input")
)

type CreateCourseInput struct{ Idx, Name, Class string }
type UpdateCourseInput struct{ Name, Class string }
type CourseOutput struct {
	ID               int
	DisplayID        string
	Idx, Name, Class string
	IsDeleted        bool
	UpdatedAt        *time.Time
}
type CourseRecordOutput struct {
	DisplayID        string
	Idx, Name, Class string
	IsPlayed         bool
	Score            uint32
	IsClear          bool
	ComboLamp        *string
	UpdatedAt        *time.Time
}
type CourseRecordResult struct {
	Records   []*CourseRecordOutput
	UpdatedAt *time.Time
}

type CourseUsecase interface {
	List(ctx context.Context, includeDeleted bool) ([]*CourseOutput, error)
	Get(ctx context.Context, displayID string, includeDeleted bool) (*CourseOutput, error)
	Create(ctx context.Context, input CreateCourseInput) (*CourseOutput, error)
	Update(ctx context.Context, displayID string, input UpdateCourseInput) (*CourseOutput, error)
	Delete(ctx context.Context, displayID string) error
	Restore(ctx context.Context, displayID string) error
	GetUserRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*CourseRecordResult, error)
	GetUserRecord(ctx context.Context, username string, requester *entity.User, displayID string) (*CourseRecordOutput, error)
}

type courseUsecase struct {
	db             repository.Executor
	repo           repository.CourseRepository
	userRepo       repository.UserRepository
	friendshipRepo repository.FriendshipRepository
}

func NewCourseUsecase(db repository.Executor, repo repository.CourseRepository, userRepo repository.UserRepository, friendshipRepo repository.FriendshipRepository) CourseUsecase {
	return &courseUsecase{db: db, repo: repo, userRepo: userRepo, friendshipRepo: friendshipRepo}
}

func (u *courseUsecase) List(ctx context.Context, includeDeleted bool) ([]*CourseOutput, error) {
	items, err := u.repo.FindAll(ctx, u.db, includeDeleted)
	if err != nil {
		return nil, err
	}
	result := make([]*CourseOutput, 0, len(items))
	for _, v := range items {
		result = append(result, toCourseOutput(v, includeDeleted))
	}
	return result, nil
}
func (u *courseUsecase) Get(ctx context.Context, displayID string, includeDeleted bool) (*CourseOutput, error) {
	item, err := u.repo.FindByDisplayID(ctx, u.db, displayID, includeDeleted)
	if errors.Is(err, repository.ErrCourseNotFound) {
		return nil, ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	return toCourseOutput(item, includeDeleted), nil
}
func (u *courseUsecase) Create(ctx context.Context, input CreateCourseInput) (*CourseOutput, error) {
	idx := strings.TrimSpace(input.Idx)
	name := strings.TrimSpace(input.Name)
	if idx == "" || name == "" {
		return nil, ErrInvalidCourseInput
	}
	if _, err := u.repo.FindByOfficialIdx(ctx, u.db, idx, true); err == nil {
		return nil, repository.ErrDuplicateOfficialIdx
	} else if !errors.Is(err, repository.ErrCourseNotFound) {
		return nil, err
	}
	class, err := u.repo.FindClassByName(ctx, u.db, normalizeCourseClass(input.Class))
	if err != nil {
		return nil, err
	}
	displayIDValue, err := generateDisplayID()
	if err != nil {
		return nil, err
	}
	displayID, err := displayid.NewDisplayID(displayIDValue)
	if err != nil {
		return nil, errors.Join(ErrInvalidCourseInput, err)
	}
	course := &entity.Course{DisplayID: displayID, OfficialIdx: idx, Name: name, CourseClassID: class.ID, CourseClass: class}
	if err := course.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidCourseInput, err)
	}
	if err := u.repo.Create(ctx, u.db, course); err != nil {
		return nil, err
	}
	created, err := u.repo.FindByOfficialIdx(ctx, u.db, idx, true)
	if err != nil {
		return nil, err
	}
	return toCourseOutput(created, true), nil
}
func (u *courseUsecase) Update(ctx context.Context, displayID string, input UpdateCourseInput) (*CourseOutput, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, ErrInvalidCourseInput
	}
	course, err := u.repo.FindByDisplayID(ctx, u.db, displayID, true)
	if errors.Is(err, repository.ErrCourseNotFound) {
		return nil, ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	class, err := u.repo.FindClassByName(ctx, u.db, normalizeCourseClass(input.Class))
	if err != nil {
		return nil, err
	}
	course.Name = strings.TrimSpace(input.Name)
	course.CourseClassID = class.ID
	course.CourseClass = class
	if err := course.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidCourseInput, err)
	}
	if err := u.repo.Save(ctx, u.db, course); err != nil {
		return nil, err
	}
	updated, err := u.repo.FindByOfficialIdx(ctx, u.db, course.OfficialIdx, true)
	if err != nil {
		return nil, err
	}
	return toCourseOutput(updated, true), nil
}
func (u *courseUsecase) Delete(ctx context.Context, displayID string) error {
	return u.setDeleted(ctx, displayID, true)
}
func (u *courseUsecase) Restore(ctx context.Context, displayID string) error {
	return u.setDeleted(ctx, displayID, false)
}
func (u *courseUsecase) setDeleted(ctx context.Context, displayID string, deleted bool) error {
	course, err := u.repo.FindByDisplayID(ctx, u.db, displayID, true)
	if errors.Is(err, repository.ErrCourseNotFound) {
		return ErrCourseNotFound
	}
	if err != nil {
		return err
	}
	course.IsDeleted = deleted
	return u.repo.Save(ctx, u.db, course)
}
func (u *courseUsecase) GetUserRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*CourseRecordResult, error) {
	user, err := u.userRepo.FindByUsername(ctx, u.db, username)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	ok, err := canAccessPrivateUser(ctx, u.db, u.friendshipRepo, user, requester)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrUserPrivate
	}
	if user.PlayerID == nil {
		return &CourseRecordResult{Records: []*CourseRecordOutput{}}, nil
	}
	records, err := u.repo.FindRecordsByPlayerID(ctx, u.db, *user.PlayerID, false, includeNoPlay)
	if err != nil {
		return nil, err
	}
	result := &CourseRecordResult{Records: make([]*CourseRecordOutput, 0, len(records))}
	for _, record := range records {
		item := toCourseRecordOutput(record)
		result.Records = append(result.Records, item)
		if item.UpdatedAt != nil && (result.UpdatedAt == nil || item.UpdatedAt.After(*result.UpdatedAt)) {
			result.UpdatedAt = item.UpdatedAt
		}
	}
	return result, nil
}

func (u *courseUsecase) GetUserRecord(ctx context.Context, username string, requester *entity.User, displayID string) (*CourseRecordOutput, error) {
	result, err := u.GetUserRecords(ctx, username, requester, true)
	if err != nil {
		return nil, err
	}
	for _, record := range result.Records {
		if record.DisplayID == displayID {
			return record, nil
		}
	}
	return nil, ErrCourseNotFound
}

func toCourseOutput(course *entity.Course, editor bool) *CourseOutput {
	if course == nil {
		return nil
	}
	class := ""
	if course.CourseClass != nil {
		class = course.CourseClass.Name
	}
	result := &CourseOutput{DisplayID: course.DisplayID.String(), Idx: course.OfficialIdx, Name: course.Name, Class: class}
	if editor {
		result.ID, result.IsDeleted = course.ID, course.IsDeleted
		result.UpdatedAt = &course.UpdatedAt
	}
	return result
}

func toCourseRecordOutput(record *entity.PlayerCourseRecord) *CourseRecordOutput {
	if record == nil || record.Course == nil {
		return nil
	}
	class := ""
	if record.Course.CourseClass != nil {
		class = record.Course.CourseClass.Name
	}
	played := !record.UpdatedAt.IsZero()
	var updated *time.Time
	if played {
		value := record.UpdatedAt
		updated = &value
	}
	var lamp *string
	if record.ComboLamp != nil && !strings.EqualFold(record.ComboLamp.Name, "none") {
		value := record.ComboLamp.Name
		lamp = &value
	}
	return &CourseRecordOutput{DisplayID: record.Course.DisplayID.String(), Idx: record.Course.OfficialIdx, Name: record.Course.Name, Class: class, IsPlayed: played, Score: record.Score.Uint32(), IsClear: record.IsClear, ComboLamp: lamp, UpdatedAt: updated}
}
func normalizeCourseClass(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "6" {
		return "inf"
	}
	return value
}
