package service

import (
	"context"
	"sort"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

func (r *fakeExamClassRepo) ListByUserWithStatuses(
	_ context.Context,
	tenantID uint64,
	userID string,
	statuses []types.ExamClassStatus,
) ([]*types.ExamClass, error) {
	allowed := make(map[types.ExamClassStatus]bool, len(statuses))
	for _, status := range statuses {
		allowed[status] = true
	}
	classes := make([]*types.ExamClass, 0)
	for _, member := range r.members {
		class := r.classes[member.ClassID]
		if member.TenantID == tenantID && member.UserID == userID &&
			member.Status == types.ExamClassMemberStatusActive && class != nil && allowed[class.Status] {
			classes = append(classes, cloneExamClass(class))
		}
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i].ID < classes[j].ID })
	return classes, nil
}

func (r *fakeExamClassRepo) GetByIDAndTenantIncludingArchived(
	_ context.Context,
	id string,
	tenantID uint64,
) (*types.ExamClass, error) {
	class := r.classes[id]
	if class == nil || class.TenantID != tenantID {
		return nil, repository.ErrExamClassNotFound
	}
	return cloneExamClass(class), nil
}

func (r *fakeExamClassRepo) ListByIDsAndTenantIncludingArchived(
	_ context.Context,
	tenantID uint64,
	ids []string,
) ([]*types.ExamClass, error) {
	r.listByIDsCalls++
	classes := make([]*types.ExamClass, 0, len(ids))
	for _, id := range ids {
		if class := r.classes[id]; class != nil && class.TenantID == tenantID {
			classes = append(classes, cloneExamClass(class))
		}
	}
	return classes, nil
}

func (r *fakeExamClassRepo) UpdateClassMetadata(
	_ context.Context,
	tenantID uint64,
	classID string,
	ownerUserID string,
	name string,
	description string,
	memberLimit int,
	updatedAt time.Time,
) error {
	if r.updateClassMetadataErr != nil {
		return r.updateClassMetadataErr
	}
	class := r.classes[classID]
	if class == nil || class.TenantID != tenantID {
		return repository.ErrExamClassNotFound
	}
	if class.OwnerUserID != ownerUserID || class.Status != types.ExamClassStatusActive {
		return repository.ErrExamClassStateConflict
	}
	activeMembers := r.activeMemberCount(classID, tenantID)
	if memberLimit != 0 && activeMembers > memberLimit {
		return repository.ErrExamClassMemberLimitConflict
	}
	class.Name = name
	class.Description = description
	class.MemberLimit = memberLimit
	class.UpdatedAt = updatedAt
	return nil
}

func (r *fakeExamClassRepo) TransitionClassStatus(
	_ context.Context,
	tenantID uint64,
	classID string,
	ownerUserID string,
	expected types.ExamClassStatus,
	next types.ExamClassStatus,
	updatedAt time.Time,
) error {
	if r.transitionClassErr != nil {
		return r.transitionClassErr
	}
	class := r.classes[classID]
	if class == nil || class.TenantID != tenantID {
		return repository.ErrExamClassNotFound
	}
	if class.OwnerUserID != ownerUserID || class.Status != expected {
		return repository.ErrExamClassStateConflict
	}
	class.Status = next
	class.UpdatedAt = updatedAt
	return nil
}

func (r *fakeExamClassRepo) ApproveMemberWithinLimit(
	_ context.Context,
	tenantID uint64,
	classID string,
	userID string,
	joinedAt time.Time,
) (*types.ExamClassMember, error) {
	class := r.classes[classID]
	if class == nil || class.TenantID != tenantID {
		return nil, repository.ErrExamClassNotFound
	}
	if class.Status != types.ExamClassStatusActive {
		return nil, repository.ErrExamClassStateConflict
	}
	member := r.members[classMemberKey(classID, userID)]
	if member == nil || member.TenantID != tenantID {
		return nil, repository.ErrExamClassMemberNotFound
	}
	if member.Status != types.ExamClassMemberStatusPending {
		return nil, repository.ErrExamClassStateConflict
	}
	if class.MemberLimit != 0 && r.activeMemberCount(classID, tenantID) >= class.MemberLimit {
		return nil, repository.ErrExamClassMemberLimitConflict
	}
	member.Status = types.ExamClassMemberStatusActive
	member.JoinedAt = joinedAt
	member.UpdatedAt = joinedAt
	return cloneExamClassMember(member), nil
}

func (r *fakeExamClassRepo) activeMemberCount(classID string, tenantID uint64) int {
	count := 0
	for _, member := range r.members {
		if member.ClassID == classID && member.TenantID == tenantID && member.Status == types.ExamClassMemberStatusActive {
			count++
		}
	}
	return count
}

func seedExamClass(repo *fakeExamClassRepo, classID string, tenantID uint64, ownerID string, inviteCode string) *types.ExamClass {
	class := &types.ExamClass{
		ID:          classID,
		TenantID:    tenantID,
		OwnerUserID: ownerID,
		SpaceID:     "space-" + classID,
		Name:        "IELTS Class",
		InviteCode:  &inviteCode,
		MemberLimit: 50,
		Status:      types.ExamClassStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.classes[class.ID] = class
	return class
}

func seedExamClassMember(repo *fakeExamClassRepo, classID string, tenantID uint64, userID string, role types.ExamClassRole, status types.ExamClassMemberStatus) {
	member := &types.ExamClassMember{
		ID:        "member-" + userID,
		ClassID:   classID,
		UserID:    userID,
		TenantID:  tenantID,
		Role:      role,
		Status:    status,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.members[classMemberKey(classID, userID)] = member
}
