package service

import (
	"context"
	"errors"
	"sort"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

func (s *examAnalyticsService) ListAnalyzableClasses(
	ctx context.Context,
	tenantID uint64,
	userID string,
) ([]*types.ExamClass, error) {
	classes, err := s.classRepo.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*types.ExamClass, 0, len(classes))
	for _, class := range classes {
		if class == nil {
			continue
		}
		member, err := s.classRepo.GetMember(ctx, class.ID, tenantID, userID)
		if err != nil {
			if errors.Is(err, repository.ErrExamClassMemberNotFound) {
				continue
			}
			return nil, err
		}
		if member.Role.CanWrite() {
			out = append(out, class)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func sortAnalyticsMembers(items []*types.ExamClassAnalyticsMember) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CompletedCount != items[j].CompletedCount {
			return items[i].CompletedCount > items[j].CompletedCount
		}
		if items[i].AverageCorrectRate != items[j].AverageCorrectRate {
			return items[i].AverageCorrectRate > items[j].AverageCorrectRate
		}
		return items[i].Member.UserID < items[j].Member.UserID
	})
}

func sortAnalyticsAssignments(items []*types.ExamClassAnalyticsAssignment) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Assignment.CreatedAt.After(items[j].Assignment.CreatedAt)
	})
}
