package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

func (u *courseUsecase) ReplaceTracks(ctx context.Context, displayID string, tracks []entity.CourseTrack) error {
	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		course, err := u.repo.FindByDisplayIDForUpdate(ctx, tx, displayID)
		if errors.Is(err, repository.ErrCourseNotFound) {
			return ErrCourseNotFound
		}
		if err != nil {
			return err
		}
		if course.IsDeleted {
			return ErrCourseNotFound
		}
		if err := course.ReplaceTracks(tracks); err != nil {
			return errors.Join(ErrInvalidCourseInput, err)
		}
		err = u.repo.ReplaceTracks(ctx, tx, course)
		if errors.Is(err, repository.ErrInvalidCourseTracks) {
			return errors.Join(ErrInvalidCourseInput, err)
		}
		return err
	})
}

func toCourseTrackOutputs(tracks []entity.CourseTrack, candidates map[int][]entity.CourseChartCandidate, detail bool) []CourseTrackOutput {
	outputs := make([]CourseTrackOutput, 0, len(tracks))
	for _, track := range tracks {
		output := CourseTrackOutput{CourseTrack: track}
		if detail {
			output.Candidates = candidates[track.Number]
			if output.Candidates == nil {
				output.Candidates = []entity.CourseChartCandidate{}
			}
		}
		outputs = append(outputs, output)
	}
	return outputs
}
