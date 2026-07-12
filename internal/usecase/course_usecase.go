package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto"
)

var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrInvalidCourseInput = errors.New("invalid course input")
)

type CreateCourseInput struct{ Idx, Name, Class string }
type UpdateCourseInput struct{ Name, Class string }
type CourseRecordResult struct {
	Records   []*dto.CourseRecordDTO
	UpdatedAt *time.Time
}

type CourseUsecase interface {
	List(ctx context.Context, includeDeleted bool) ([]*dto.CourseDTO, error)
	Get(ctx context.Context, idx string, includeDeleted bool) (*dto.CourseDTO, error)
	Create(ctx context.Context, input CreateCourseInput) (*dto.CourseDTO, error)
	Update(ctx context.Context, idx string, input UpdateCourseInput) (*dto.CourseDTO, error)
	Delete(ctx context.Context, idx string) error
	Restore(ctx context.Context, idx string) error
	GetUserRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*CourseRecordResult, error)
	GetUserRecord(ctx context.Context, username string, requester *entity.User, idx string) (*dto.CourseRecordDTO, error)
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

func (u *courseUsecase) List(ctx context.Context, includeDeleted bool) ([]*dto.CourseDTO, error) {
	items, err := u.repo.FindAll(ctx, u.db, includeDeleted)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.CourseDTO, 0, len(items))
	for _, v := range items {
		result = append(result, dto.ToCourseDTO(v, includeDeleted))
	}
	return result, nil
}
func (u *courseUsecase) Get(ctx context.Context, idx string, includeDeleted bool) (*dto.CourseDTO, error) {
	item, err := u.repo.FindByOfficialIdx(ctx, u.db, strings.TrimSpace(idx), includeDeleted)
	if errors.Is(err, repository.ErrCourseNotFound) {
		return nil, ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	return dto.ToCourseDTO(item, includeDeleted), nil
}
func (u *courseUsecase) Create(ctx context.Context, input CreateCourseInput) (*dto.CourseDTO, error) {
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
	course := &entity.Course{OfficialIdx: idx, Name: name, CourseClassID: class.ID, CourseClass: class}
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
	return dto.ToCourseDTO(created, true), nil
}
func (u *courseUsecase) Update(ctx context.Context, idx string, input UpdateCourseInput) (*dto.CourseDTO, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, ErrInvalidCourseInput
	}
	course, err := u.repo.FindByOfficialIdx(ctx, u.db, strings.TrimSpace(idx), true)
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
	return dto.ToCourseDTO(updated, true), nil
}
func (u *courseUsecase) Delete(ctx context.Context, idx string) error {
	return u.setDeleted(ctx, idx, true)
}
func (u *courseUsecase) Restore(ctx context.Context, idx string) error {
	return u.setDeleted(ctx, idx, false)
}
func (u *courseUsecase) setDeleted(ctx context.Context, idx string, deleted bool) error {
	course, err := u.repo.FindByOfficialIdx(ctx, u.db, strings.TrimSpace(idx), true)
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
		return &CourseRecordResult{Records: []*dto.CourseRecordDTO{}}, nil
	}
	records, err := u.repo.FindRecordsByPlayerID(ctx, u.db, *user.PlayerID, false, includeNoPlay)
	if err != nil {
		return nil, err
	}
	result := &CourseRecordResult{Records: make([]*dto.CourseRecordDTO, 0, len(records))}
	for _, record := range records {
		item := dto.ToCourseRecordDTO(record)
		result.Records = append(result.Records, item)
		if item.UpdatedAt != nil && (result.UpdatedAt == nil || item.UpdatedAt.After(*result.UpdatedAt)) {
			result.UpdatedAt = item.UpdatedAt
		}
	}
	return result, nil
}

func (u *courseUsecase) GetUserRecord(ctx context.Context, username string, requester *entity.User, idx string) (*dto.CourseRecordDTO, error) {
	result, err := u.GetUserRecords(ctx, username, requester, true)
	if err != nil {
		return nil, err
	}
	idx = strings.TrimSpace(idx)
	for _, record := range result.Records {
		if record.Idx == idx {
			return record, nil
		}
	}
	return nil, ErrCourseNotFound
}
func normalizeCourseClass(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "6" {
		return "inf"
	}
	return value
}
