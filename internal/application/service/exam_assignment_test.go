package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

func TestExamAssignmentCreateRequiresClassWriteRole(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, "class-1", 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = class.SpaceID
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, newStubPracticeRepo()).(*examAssignmentService)

	_, err := svc.CreateAssignment(ctx, 10000, "student-1", "class-1", &types.CreateExamAssignmentRequest{
		GroupID: "group-1",
		Title:   "Reading homework",
	})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("CreateAssignment error = %v, want ErrExamPermissionDenied", err)
	}
	if len(assignRepo.assignments) != 0 {
		t.Fatalf("student created %d assignments, want 0", len(assignRepo.assignments))
	}
}

func TestExamAssignmentCreateRejectsGroupOutsideClassSpace(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, "class-1", 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = "other-space"
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, newStubPracticeRepo()).(*examAssignmentService)

	_, err := svc.CreateAssignment(ctx, 10000, "teacher-1", "class-1", &types.CreateExamAssignmentRequest{
		GroupID: "group-1",
		Title:   "Reading homework",
	})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("CreateAssignment error = %v, want ErrExamPermissionDenied", err)
	}
}

func TestExamAssignmentAttemptStoresAssignmentID(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, "class-1", 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, "class-1", 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = class.SpaceID
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	practiceRepo := newStubPracticeRepo()
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, practiceRepo).(*examAssignmentService)
	assignment, err := svc.CreateAssignment(ctx, 10000, "teacher-1", class.ID, &types.CreateExamAssignmentRequest{
		GroupID:      "group-1",
		Title:        "Reading homework",
		Instructions: "Finish before Friday.",
	})
	if err != nil {
		t.Fatalf("CreateAssignment returned error: %v", err)
	}

	result, err := svc.CreateAssignmentAttempt(ctx, 10000, "student-1", assignment.Assignment.ID)

	if err != nil {
		t.Fatalf("CreateAssignmentAttempt returned error: %v", err)
	}
	if result.Attempt.AssignmentID == nil || *result.Attempt.AssignmentID != assignment.Assignment.ID {
		t.Fatalf("attempt assignment_id = %#v, want %s", result.Attempt.AssignmentID, assignment.Assignment.ID)
	}
	if result.Attempt.SpaceID != class.SpaceID || result.Attempt.GroupID != "group-1" {
		t.Fatalf("attempt scope = space %s group %s, want %s/group-1", result.Attempt.SpaceID, result.Attempt.GroupID, class.SpaceID)
	}
}

func TestExamAssignmentListMineRequiresActiveClassMember(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, "class-1", 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, "class-1", 10000, "student-active", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, "class-1", 10000, "student-pending", types.ExamClassRoleStudent, types.ExamClassMemberStatusPending)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = class.SpaceID
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, newStubPracticeRepo()).(*examAssignmentService)
	if _, err := svc.CreateAssignment(ctx, 10000, "teacher-1", class.ID, &types.CreateExamAssignmentRequest{
		GroupID: "group-1",
		Title:   "Reading homework",
	}); err != nil {
		t.Fatalf("CreateAssignment returned error: %v", err)
	}

	activeItems, err := svc.ListMyAssignments(ctx, 10000, "student-active", types.ListExamAssignmentsFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListMyAssignments(active) returned error: %v", err)
	}
	if len(activeItems) != 1 {
		t.Fatalf("active student assignment count = %d, want 1", len(activeItems))
	}
	pendingItems, err := svc.ListMyAssignments(ctx, 10000, "student-pending", types.ListExamAssignmentsFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListMyAssignments(pending) returned error: %v", err)
	}
	if len(pendingItems) != 0 {
		t.Fatalf("pending student assignment count = %d, want 0", len(pendingItems))
	}
}

type stubExamAssignmentRepo struct {
	assignments []*types.ExamClassAssignment
	classRepo   *fakeExamClassRepo
}

func newStubExamAssignmentRepo(classRepo *fakeExamClassRepo) *stubExamAssignmentRepo {
	return &stubExamAssignmentRepo{classRepo: classRepo}
}

func (r *stubExamAssignmentRepo) CreateAssignment(_ context.Context, assignment *types.ExamClassAssignment) error {
	r.assignments = append(r.assignments, cloneExamClassAssignment(assignment))
	return nil
}

func (r *stubExamAssignmentRepo) GetAssignmentByIDAndTenant(_ context.Context, tenantID uint64, assignmentID string) (*types.ExamClassAssignment, error) {
	for _, assignment := range r.assignments {
		if assignment.ID == assignmentID && assignment.TenantID == tenantID {
			return cloneExamClassAssignment(assignment), nil
		}
	}
	return nil, repository.ErrExamClassAssignmentNotFound
}

func (r *stubExamAssignmentRepo) ListAssignmentsByClass(_ context.Context, tenantID uint64, classID string, limit int) ([]*types.ExamClassAssignment, error) {
	out := []*types.ExamClassAssignment{}
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.ClassID == classID && assignment.Status == types.ExamAssignmentStatusPublished {
			out = append(out, cloneExamClassAssignment(assignment))
		}
	}
	return limitExamAssignments(out, limit), nil
}

func (r *stubExamAssignmentRepo) ListAssignmentsByUserClasses(ctx context.Context, tenantID uint64, userID string, limit int) ([]*types.ExamClassAssignment, error) {
	out := []*types.ExamClassAssignment{}
	for _, assignment := range r.assignments {
		if assignment.TenantID != tenantID || assignment.Status != types.ExamAssignmentStatusPublished {
			continue
		}
		member, err := r.classRepo.GetMember(ctx, assignment.ClassID, tenantID, userID)
		if err == nil && member.Status == types.ExamClassMemberStatusActive {
			out = append(out, cloneExamClassAssignment(assignment))
		}
	}
	return limitExamAssignments(out, limit), nil
}

func limitExamAssignments(items []*types.ExamClassAssignment, limit int) []*types.ExamClassAssignment {
	if limit <= 0 || limit >= len(items) {
		return items
	}
	return items[:limit]
}

func cloneExamClassAssignment(assignment *types.ExamClassAssignment) *types.ExamClassAssignment {
	if assignment == nil {
		return nil
	}
	cp := *assignment
	if assignment.DueAt != nil {
		dueAt := *assignment.DueAt
		cp.DueAt = &dueAt
	}
	return &cp
}

func assignmentDueAt() *time.Time {
	due := time.Now().Add(48 * time.Hour)
	return &due
}
