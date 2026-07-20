package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
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
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, newStubPracticeRepo(), newStubExamAssignmentSpaceService()).(*examAssignmentService)

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
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, newStubPracticeRepo(), newStubExamAssignmentSpaceService()).(*examAssignmentService)

	_, err := svc.CreateAssignment(ctx, 10000, "teacher-1", "class-1", &types.CreateExamAssignmentRequest{
		GroupID: "group-1",
		Title:   "Reading homework",
	})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("CreateAssignment error = %v, want ErrExamPermissionDenied", err)
	}
}

func TestExamAssignmentCreateImportsReadableGroupOutsideClassSpace(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, "class-1", 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = "personal-space"
	group.Assets = []*types.QuestionGroupAsset{{
		ID:            "asset-1",
		TenantID:      10000,
		GroupID:       group.Group.ID,
		AssetType:     "image",
		StorageURI:    "oss://exam/asset-1.png",
		AltText:       "diagram",
		SourceChunkID: "chunk-asset",
		BBox:          types.JSONMap{},
		Metadata:      types.JSONMap{},
		SortOrder:     1,
		CreatedAt:     time.Now(),
	}}
	questionRepo := &stubQuestionGroupWriter{
		banks: []*types.QuestionBank{{
			ID:              "bank-1",
			TenantID:        10000,
			SpaceID:         "personal-space",
			DomainID:        "gaokao",
			Name:            "2026高考英语",
			SourceType:      "material_structuring",
			ReviewStatus:    types.ExamReviewStatusPrivate,
			Status:          "active",
			CreatedByUserID: "teacher-1",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}},
		created: []*types.QuestionGroupDetail{group},
	}
	svc := NewExamAssignmentService(
		assignRepo,
		classRepo,
		questionRepo,
		newStubPracticeRepo(),
		newStubExamAssignmentSpaceService("personal-space"),
	).(*examAssignmentService)

	summary, err := svc.CreateAssignment(ctx, 10000, "teacher-1", "class-1", &types.CreateExamAssignmentRequest{
		GroupID: group.Group.ID,
		Title:   "Reading homework",
	})

	if err != nil {
		t.Fatalf("CreateAssignment returned error: %v", err)
	}
	if summary.Assignment.SpaceID != class.SpaceID {
		t.Fatalf("assignment space_id = %s, want %s", summary.Assignment.SpaceID, class.SpaceID)
	}
	if summary.Assignment.GroupID == group.Group.ID {
		t.Fatalf("assignment reused source group %s, want imported class-space group", group.Group.ID)
	}
	if len(questionRepo.created) != 2 {
		t.Fatalf("created group count = %d, want source plus imported group", len(questionRepo.created))
	}
	imported := questionRepo.created[1]
	if imported.Group.SpaceID != class.SpaceID {
		t.Fatalf("imported group space_id = %s, want %s", imported.Group.SpaceID, class.SpaceID)
	}
	if imported.Group.QuestionBankID == group.Group.QuestionBankID {
		t.Fatalf("imported group reused source bank %s", group.Group.QuestionBankID)
	}
	if len(imported.Assets) != 1 || imported.Assets[0].GroupID != imported.Group.ID || imported.Assets[0].ID == "asset-1" {
		t.Fatalf("imported assets were not cloned correctly: %#v", imported.Assets)
	}
	if len(imported.Questions) != 1 {
		t.Fatalf("imported question count = %d, want 1", len(imported.Questions))
	}
	importedQuestion := imported.Questions[0]
	if importedQuestion.Question.ID == "question-1" ||
		importedQuestion.Question.QuestionBankID != imported.Group.QuestionBankID ||
		importedQuestion.Question.GroupID == nil ||
		*importedQuestion.Question.GroupID != imported.Group.ID {
		t.Fatalf("imported question scope/id = %#v, want cloned class-space question", importedQuestion.Question)
	}
	if len(importedQuestion.Options) != 2 || importedQuestion.Options[0].QuestionID != importedQuestion.Question.ID {
		t.Fatalf("imported options were not cloned correctly: %#v", importedQuestion.Options)
	}
	if len(importedQuestion.Answers) != 1 || importedQuestion.Answers[0].QuestionID != importedQuestion.Question.ID {
		t.Fatalf("imported answers were not cloned correctly: %#v", importedQuestion.Answers)
	}
	if len(importedQuestion.Explanations) != 1 || importedQuestion.Explanations[0].QuestionID != importedQuestion.Question.ID {
		t.Fatalf("imported explanations were not cloned correctly: %#v", importedQuestion.Explanations)
	}
	if len(importedQuestion.ChunkRefs) != 1 ||
		importedQuestion.ChunkRefs[0].QuestionID != importedQuestion.Question.ID ||
		importedQuestion.ChunkRefs[0].ChunkID != "chunk-1" {
		t.Fatalf("imported chunk refs were not cloned correctly: %#v", importedQuestion.ChunkRefs)
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
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, practiceRepo, newStubExamAssignmentSpaceService()).(*examAssignmentService)
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

func TestExamAssignmentAttemptMapsAtomicClosureToStateConflict(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	fixture.practiceRepo.assignmentAttemptErr = repository.ErrExamAssignmentAttemptClosed

	result, err := fixture.svc.CreateAssignmentAttempt(
		context.Background(), 10000, "student-1", "assignment-1",
	)

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrExamStateConflict)
	require.Empty(t, fixture.practiceRepo.attempts)
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
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, newStubPracticeRepo(), newStubExamAssignmentSpaceService()).(*examAssignmentService)
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

func TestExamAssignmentProgressSummarizesActiveStudents(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "assistant-1", types.ExamClassRoleAssistant, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-done", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-new", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-pending", types.ExamClassRoleStudent, types.ExamClassMemberStatusPending)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = class.SpaceID
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	practiceRepo := newStubPracticeRepo()
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, practiceRepo, newStubExamAssignmentSpaceService()).(*examAssignmentService)
	assignment, err := svc.CreateAssignment(ctx, 10000, "teacher-1", class.ID, &types.CreateExamAssignmentRequest{
		GroupID: group.Group.ID,
		Title:   "Reading homework",
	})
	if err != nil {
		t.Fatalf("CreateAssignment returned error: %v", err)
	}
	completedAt := time.Now()
	assignmentID := assignment.Assignment.ID
	practiceRepo.attempts = append(practiceRepo.attempts, &types.ExamPracticeAttempt{
		ID:             "attempt-done",
		TenantID:       10000,
		UserID:         "student-done",
		SpaceID:        class.SpaceID,
		QuestionBankID: group.Group.QuestionBankID,
		GroupID:        group.Group.ID,
		AssignmentID:   &assignmentID,
		Status:         types.ExamPracticeAttemptStatusCompleted,
		QuestionCount:  5,
		AnsweredCount:  5,
		CorrectCount:   4,
		StartedAt:      completedAt.Add(-15 * time.Minute),
		CompletedAt:    &completedAt,
		CreatedAt:      completedAt.Add(-15 * time.Minute),
		UpdatedAt:      completedAt,
	})
	practiceRepo.attempts = append(practiceRepo.attempts, &types.ExamPracticeAttempt{
		ID:             "attempt-pending-member",
		TenantID:       10000,
		UserID:         "student-pending",
		SpaceID:        class.SpaceID,
		QuestionBankID: group.Group.QuestionBankID,
		GroupID:        group.Group.ID,
		AssignmentID:   &assignmentID,
		Status:         types.ExamPracticeAttemptStatusCompleted,
		QuestionCount:  5,
		AnsweredCount:  5,
		CorrectCount:   5,
		StartedAt:      completedAt,
		CompletedAt:    &completedAt,
		CreatedAt:      completedAt,
		UpdatedAt:      completedAt,
	})

	progress, err := svc.GetAssignmentProgress(ctx, 10000, "teacher-1", class.ID, assignment.Assignment.ID)

	if err != nil {
		t.Fatalf("GetAssignmentProgress returned error: %v", err)
	}
	if progress.TotalStudents != 2 || progress.StartedCount != 1 || progress.CompletedCount != 1 {
		t.Fatalf("progress counts = total %d started %d completed %d, want 2/1/1", progress.TotalStudents, progress.StartedCount, progress.CompletedCount)
	}
	if progress.AverageCorrectRate != 0.8 {
		t.Fatalf("average correct rate = %.2f, want 0.80", progress.AverageCorrectRate)
	}
	if len(progress.Members) != 2 {
		t.Fatalf("member progress count = %d, want 2", len(progress.Members))
	}
	done := findAssignmentProgress(progress.Members, "student-done")
	if done == nil || done.Status != types.ExamAssignmentProgressStatusCompleted || done.CorrectRate != 0.8 {
		t.Fatalf("student-done progress = %#v, want completed with 0.8 rate", done)
	}
	notStarted := findAssignmentProgress(progress.Members, "student-new")
	if notStarted == nil || notStarted.Status != types.ExamAssignmentProgressStatusNotStarted || notStarted.Attempt != nil {
		t.Fatalf("student-new progress = %#v, want not started without attempt", notStarted)
	}
	if findAssignmentProgress(progress.Members, "student-pending") != nil {
		t.Fatalf("pending student should not be included in assignment progress")
	}
	if findAssignmentProgress(progress.Members, "assistant-1") != nil {
		t.Fatalf("assistant should not be included as a student progress row")
	}
}

func TestExamAssignmentProgressRequiresClassWriteRole(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = class.SpaceID
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	svc := NewExamAssignmentService(assignRepo, classRepo, questionRepo, newStubPracticeRepo(), newStubExamAssignmentSpaceService()).(*examAssignmentService)
	assignment, err := svc.CreateAssignment(ctx, 10000, "teacher-1", class.ID, &types.CreateExamAssignmentRequest{
		GroupID: group.Group.ID,
		Title:   "Reading homework",
	})
	if err != nil {
		t.Fatalf("CreateAssignment returned error: %v", err)
	}

	_, err = svc.GetAssignmentProgress(ctx, 10000, "student-1", class.ID, assignment.Assignment.ID)

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("GetAssignmentProgress error = %v, want ErrExamPermissionDenied", err)
	}
}

type assignmentLifecycleFixture struct {
	svc            *examAssignmentService
	assignmentRepo *stubExamAssignmentRepo
	practiceRepo   *stubPracticeRepo
	now            time.Time
}

func newAssignmentLifecycleFixture(t *testing.T) *assignmentLifecycleFixture {
	t.Helper()
	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	classRepo := newFakeExamClassRepo()
	assignmentRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "pending-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusPending)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = class.SpaceID
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	practiceRepo := newStubPracticeRepo()
	svc := NewExamAssignmentService(
		assignmentRepo,
		classRepo,
		questionRepo,
		practiceRepo,
		newStubExamAssignmentSpaceService(),
	).(*examAssignmentService)
	svc.now = func() time.Time { return now }
	dueAt := now.Add(24 * time.Hour)
	assignmentRepo.assignments = append(assignmentRepo.assignments, &types.ExamClassAssignment{
		ID:              "assignment-1",
		TenantID:        10000,
		ClassID:         class.ID,
		SpaceID:         class.SpaceID,
		QuestionBankID:  group.Group.QuestionBankID,
		GroupID:         group.Group.ID,
		Title:           "Reading homework",
		Status:          types.ExamAssignmentStatusPublished,
		DueAt:           &dueAt,
		CreatedByUserID: "teacher-1",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	return &assignmentLifecycleFixture{
		svc:            svc,
		assignmentRepo: assignmentRepo,
		practiceRepo:   practiceRepo,
		now:            now,
	}
}

func TestExamAssignmentLifecycleWithdrawEditRepublish(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	ctx := context.Background()
	assignmentID := "assignment-1"
	fixture.practiceRepo.attempts = append(fixture.practiceRepo.attempts, &types.ExamPracticeAttempt{
		ID: "attempt-1", TenantID: 10000, UserID: "student-1", AssignmentID: &assignmentID,
		Status: types.ExamPracticeAttemptStatusInProgress,
	})

	withdrawn, err := fixture.svc.WithdrawAssignment(ctx, 10000, "teacher-1", "class-1", assignmentID)
	require.NoError(t, err)
	require.Equal(t, types.ExamAssignmentStatusWithdrawn, withdrawn.Assignment.Status)

	dueAt := fixture.now.Add(48 * time.Hour)
	updated, err := fixture.svc.UpdateAssignment(ctx, 10000, "teacher-1", "class-1", assignmentID, &types.UpdateExamAssignmentRequest{
		Title: "Extended homework", Instructions: "Finish every question", DueAt: &dueAt,
	})
	require.NoError(t, err)
	require.Equal(t, "Extended homework", updated.Assignment.Title)
	require.Equal(t, withdrawn.Assignment.GroupID, updated.Assignment.GroupID)

	republished, err := fixture.svc.RepublishAssignment(ctx, 10000, "teacher-1", "class-1", assignmentID)
	require.NoError(t, err)
	require.Equal(t, assignmentID, republished.Assignment.ID)
	require.Equal(t, types.ExamAssignmentStatusPublished, republished.Assignment.Status)
	require.Len(t, fixture.practiceRepo.attempts, 1)
	require.Equal(t, assignmentID, *fixture.practiceRepo.attempts[0].AssignmentID)
}

func TestExamAssignmentLifecycleRejectsExpiredPublishedUntilWithdrawn(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	ctx := context.Background()
	pastDueAt := fixture.now.Add(-time.Minute)
	fixture.assignmentRepo.assignments[0].DueAt = &pastDueAt

	_, err := fixture.svc.UpdateAssignment(ctx, 10000, "teacher-1", "class-1", "assignment-1", &types.UpdateExamAssignmentRequest{
		Title: "Late edit",
	})
	require.ErrorIs(t, err, ErrExamStateConflict)
	_, err = fixture.svc.CreateAssignmentAttempt(ctx, 10000, "student-1", "assignment-1")
	require.ErrorIs(t, err, ErrExamStateConflict)
	require.Empty(t, fixture.practiceRepo.attempts)

	_, err = fixture.svc.WithdrawAssignment(ctx, 10000, "teacher-1", "class-1", "assignment-1")
	require.NoError(t, err)
	futureDueAt := fixture.now.Add(24 * time.Hour)
	_, err = fixture.svc.UpdateAssignment(ctx, 10000, "teacher-1", "class-1", "assignment-1", &types.UpdateExamAssignmentRequest{
		Title: "Reopened homework", DueAt: &futureDueAt,
	})
	require.NoError(t, err)
	_, err = fixture.svc.RepublishAssignment(ctx, 10000, "teacher-1", "class-1", "assignment-1")
	require.NoError(t, err)
}

func TestExamAssignmentLifecycleEnforcesRoleAndActiveMembership(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	ctx := context.Background()
	request := &types.UpdateExamAssignmentRequest{Title: "Student edit"}

	_, err := fixture.svc.UpdateAssignment(ctx, 10000, "student-1", "class-1", "assignment-1", request)
	require.ErrorIs(t, err, ErrExamPermissionDenied)
	_, err = fixture.svc.WithdrawAssignment(ctx, 10000, "student-1", "class-1", "assignment-1")
	require.ErrorIs(t, err, ErrExamPermissionDenied)
	_, err = fixture.svc.RepublishAssignment(ctx, 10000, "student-1", "class-1", "assignment-1")
	require.ErrorIs(t, err, ErrExamPermissionDenied)
	_, err = fixture.svc.ListClassAssignments(ctx, 10000, "pending-1", "class-1", types.ListExamAssignmentsFilter{})
	require.ErrorIs(t, err, ErrExamPermissionDenied)
}

func TestExamAssignmentListAndProgressRespectWithdrawnVisibility(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	ctx := context.Background()
	withdrawn := cloneExamClassAssignment(fixture.assignmentRepo.assignments[0])
	withdrawn.ID = "assignment-withdrawn"
	withdrawn.Status = types.ExamAssignmentStatusWithdrawn
	withdrawn.CreatedAt = withdrawn.CreatedAt.Add(time.Minute)
	fixture.assignmentRepo.assignments = append(fixture.assignmentRepo.assignments, withdrawn)

	teacherItems, err := fixture.svc.ListClassAssignments(ctx, 10000, "teacher-1", "class-1", types.ListExamAssignmentsFilter{})
	require.NoError(t, err)
	require.Len(t, teacherItems, 2)
	studentItems, err := fixture.svc.ListClassAssignments(ctx, 10000, "student-1", "class-1", types.ListExamAssignmentsFilter{})
	require.NoError(t, err)
	require.Len(t, studentItems, 1)
	require.Equal(t, types.ExamAssignmentStatusPublished, studentItems[0].Assignment.Status)

	progress, err := fixture.svc.GetAssignmentProgress(ctx, 10000, "teacher-1", "class-1", withdrawn.ID)
	require.NoError(t, err)
	require.Equal(t, types.ExamAssignmentStatusWithdrawn, progress.Assignment.Status)
}

func findAssignmentProgress(items []*types.ExamAssignmentMemberProgress, userID string) *types.ExamAssignmentMemberProgress {
	for _, item := range items {
		if item != nil && item.Member != nil && item.Member.UserID == userID {
			return item
		}
	}
	return nil
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

func (r *stubExamAssignmentRepo) ListAssignmentsByClass(
	_ context.Context,
	tenantID uint64,
	classID string,
	statuses []types.ExamAssignmentStatus,
	limit int,
) ([]*types.ExamClassAssignment, error) {
	out := []*types.ExamClassAssignment{}
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.ClassID == classID && assignmentStatusIncluded(assignment.Status, statuses) {
			out = append(out, cloneExamClassAssignment(assignment))
		}
	}
	return limitExamAssignments(out, limit), nil
}

func (r *stubExamAssignmentRepo) UpdateAssignmentMetadata(
	_ context.Context,
	tenantID uint64,
	classID string,
	assignmentID string,
	allowed []types.ExamAssignmentStatus,
	title string,
	instructions string,
	dueAt *time.Time,
	updatedAt time.Time,
) error {
	for _, assignment := range r.assignments {
		if assignment.ID != assignmentID || assignment.TenantID != tenantID || assignment.ClassID != classID {
			continue
		}
		if !assignmentStatusIncluded(assignment.Status, allowed) {
			return repository.ErrExamClassAssignmentStateConflict
		}
		assignment.Title = title
		assignment.Instructions = instructions
		assignment.DueAt = dueAt
		assignment.UpdatedAt = updatedAt
		return nil
	}
	return repository.ErrExamClassAssignmentStateConflict
}

func (r *stubExamAssignmentRepo) TransitionAssignmentStatus(
	_ context.Context,
	tenantID uint64,
	classID string,
	assignmentID string,
	expected types.ExamAssignmentStatus,
	next types.ExamAssignmentStatus,
	updatedAt time.Time,
) error {
	for _, assignment := range r.assignments {
		if assignment.ID != assignmentID || assignment.TenantID != tenantID || assignment.ClassID != classID || assignment.Status != expected {
			continue
		}
		assignment.Status = next
		assignment.UpdatedAt = updatedAt
		return nil
	}
	return repository.ErrExamClassAssignmentStateConflict
}

func assignmentStatusIncluded(status types.ExamAssignmentStatus, statuses []types.ExamAssignmentStatus) bool {
	for _, candidate := range statuses {
		if candidate == status {
			return true
		}
	}
	return false
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

type stubExamAssignmentSpaceService struct {
	readable map[string]bool
}

func newStubExamAssignmentSpaceService(readableSpaces ...string) *stubExamAssignmentSpaceService {
	readable := map[string]bool{}
	for _, spaceID := range readableSpaces {
		readable[spaceID] = true
	}
	return &stubExamAssignmentSpaceService{readable: readable}
}

func (s *stubExamAssignmentSpaceService) ListSpaces(context.Context, uint64, string) ([]*types.ExamSpace, error) {
	return nil, nil
}

func (s *stubExamAssignmentSpaceService) EnsurePersonalSpace(context.Context, uint64, string) (*types.ExamSpace, error) {
	return nil, nil
}

func (s *stubExamAssignmentSpaceService) CanReadSpace(_ context.Context, _ uint64, _ string, spaceID string) (bool, error) {
	return s.readable[spaceID], nil
}

func (s *stubExamAssignmentSpaceService) CanWriteSpace(context.Context, uint64, string, string) (bool, error) {
	return false, nil
}

func (s *stubExamAssignmentSpaceService) GetSpace(context.Context, uint64, string, string) (*types.ExamSpace, error) {
	return nil, ErrExamNotFound
}
