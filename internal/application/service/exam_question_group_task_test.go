package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type questionGroupTaskEnqueuerStub struct {
	task *asynq.Task
	err  error
}

func (s *questionGroupTaskEnqueuerStub) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	s.task = task
	if s.err != nil {
		return nil, s.err
	}
	return &asynq.TaskInfo{ID: "queued-1", Queue: types.QueueQuestion}, nil
}

func TestQuestionGroupDraftServiceStartExtractionOnlyEnqueues(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	extractorCalled := false
	enqueuer := &questionGroupTaskEnqueuerStub{}
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{canRead: true, canWrite: true},
		&stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{{ID: "chunk-1"}}},
		&stubQuestionGroupExtractor{onExtract: func() { extractorCalled = true }},
		nil,
	)
	svc.taskEnqueuer = enqueuer

	result, err := svc.StartExtraction(context.Background(), 10000, "teacher-1", "task-1", false)

	if err != nil {
		t.Fatalf("StartExtraction returned error: %v", err)
	}
	if extractorCalled {
		t.Fatal("StartExtraction must not call the model extractor in the HTTP request")
	}
	if enqueuer.task == nil || enqueuer.task.Type() != types.TypeExamQuestionGroupExtraction {
		t.Fatalf("enqueued task = %#v", enqueuer.task)
	}
	if result.Task.Status != types.ExamStructuringTaskStatusExtracting {
		t.Fatalf("task status = %s", result.Task.Status)
	}
	if result.Task.Progress.Phase != types.ExamStructuringPhaseQueued {
		t.Fatalf("progress phase = %s", result.Task.Progress.Phase)
	}
	var payload types.ExamQuestionGroupExtractionPayload
	if err := json.Unmarshal(enqueuer.task.Payload(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.TaskID != "task-1" || payload.UserID != "teacher-1" || payload.TenantID != 10000 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestQuestionGroupDraftServiceForceReextractsUnapprovedReviewTask(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusReviewing
	repo.drafts = []*types.ExamQuestionGroupDraft{newPendingQuestionGroupDraft("draft-1")}
	enqueuer := &questionGroupTaskEnqueuerStub{}
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{canRead: true, canWrite: true},
		nil,
		nil,
		nil,
	)
	svc.taskEnqueuer = enqueuer

	result, err := svc.StartExtraction(context.Background(), 10000, "teacher-1", "task-1", true)

	if err != nil {
		t.Fatalf("expected force re-extraction before approvals, got %v", err)
	}
	if result.Task.Status != types.ExamStructuringTaskStatusExtracting || enqueuer.task == nil {
		t.Fatalf("force re-extraction was not queued: result=%#v task=%#v", result, enqueuer.task)
	}
}

func TestQuestionGroupDraftServiceRejectsForceReextractAfterApproval(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusReviewing
	approved := newPendingQuestionGroupDraft("draft-1")
	approved.Status = types.ExamQuestionGroupDraftStatusApproved
	repo.drafts = []*types.ExamQuestionGroupDraft{approved}
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{canRead: true, canWrite: true},
		nil,
		nil,
		nil,
	)
	svc.taskEnqueuer = &questionGroupTaskEnqueuerStub{}

	_, err := svc.StartExtraction(context.Background(), 10000, "teacher-1", "task-1", true)

	if !errors.Is(err, ErrExamInvalidRequest) {
		t.Fatalf("expected approved review task to reject force re-extraction, got %v", err)
	}
}

func TestQuestionGroupDraftServiceStartExtractionPersistsEnqueueFailure(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{canRead: true, canWrite: true},
		nil,
		nil,
		nil,
	)
	svc.taskEnqueuer = &questionGroupTaskEnqueuerStub{err: errors.New("redis unavailable")}

	_, err := svc.StartExtraction(context.Background(), 10000, "teacher-1", "task-1", false)

	if err == nil {
		t.Fatal("expected enqueue error")
	}
	if repo.task.Status != types.ExamStructuringTaskStatusFailed {
		t.Fatalf("task status = %s", repo.task.Status)
	}
	if repo.task.Progress.Phase != types.ExamStructuringPhaseFailed {
		t.Fatalf("progress phase = %s", repo.task.Progress.Phase)
	}
}

func TestQuestionGroupDraftServiceProcessesQueuedExtraction(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusExtracting
	repo.task.Progress = types.ExamStructuringProgress{Phase: types.ExamStructuringPhaseQueued}
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{canRead: true, canWrite: true},
		&stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{{ID: "chunk-1", Content: "1. Solve x + 1 = 2"}}},
		&stubQuestionGroupExtractor{candidates: []*types.ExamQuestionGroupDraftCandidate{newMathGroupCandidate()}},
		nil,
	)
	payload, _ := json.Marshal(types.ExamQuestionGroupExtractionPayload{
		TenantID: 10000,
		UserID:   "teacher-1",
		TaskID:   "task-1",
	})

	err := svc.ProcessExtractionTask(context.Background(), asynq.NewTask(types.TypeExamQuestionGroupExtraction, payload))

	if err != nil {
		t.Fatalf("ProcessExtractionTask returned error: %v", err)
	}
	if repo.task.Status != types.ExamStructuringTaskStatusReviewing {
		t.Fatalf("task status = %s", repo.task.Status)
	}
	if repo.task.Progress.Phase != types.ExamStructuringPhaseCompleted || repo.task.Progress.Percent != 100 {
		t.Fatalf("progress = %#v", repo.task.Progress)
	}
	if len(repo.createdDrafts) != 1 {
		t.Fatalf("created drafts = %d", len(repo.createdDrafts))
	}
}

func TestQuestionGroupDraftWorkerUsesValidatedQueuedIdentity(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusExtracting
	chunks := &stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{{ID: "chunk-1", Content: "1. Solve x + 1 = 2"}}}
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{canRead: false, canWrite: false},
		chunks,
		&stubQuestionGroupExtractor{candidates: []*types.ExamQuestionGroupDraftCandidate{newMathGroupCandidate()}},
		nil,
	)
	task := queuedQuestionGroupExtractionTask(t)

	err := svc.ProcessExtractionTask(context.Background(), task)

	if err != nil {
		t.Fatalf("worker re-authorized an already validated task: %v", err)
	}
	if chunks.gotTenantID != 10000 {
		t.Fatalf("chunk reader tenant = %d, want 10000", chunks.gotTenantID)
	}
}

func TestQuestionGroupDraftWorkerPersistsMaterialLoadFailure(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusExtracting
	repo.material = nil
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{canRead: true, canWrite: true},
		nil,
		nil,
		nil,
	)

	err := svc.ProcessExtractionTask(context.Background(), queuedQuestionGroupExtractionTask(t))

	if err == nil {
		t.Fatal("expected material load failure")
	}
	if repo.task.Status != types.ExamStructuringTaskStatusFailed || repo.task.Progress.Phase != types.ExamStructuringPhaseFailed {
		t.Fatalf("task after load failure = %#v", repo.task)
	}
}

func queuedQuestionGroupExtractionTask(t *testing.T) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(types.ExamQuestionGroupExtractionPayload{
		TenantID: 10000,
		UserID:   "teacher-1",
		TaskID:   "task-1",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return asynq.NewTask(types.TypeExamQuestionGroupExtraction, payload)
}

func TestQuestionGroupDraftWorkerConvertsPanicToFailedTask(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusExtracting
	svc := newTestExamQuestionGroupDraftService(
		repo,
		&stubExamQuestionDraftSpace{},
		&stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{{ID: "chunk-1", Content: "1. Q1"}}},
		&stubQuestionGroupExtractor{onExtract: func() { panic("model adapter panic") }},
		nil,
	)

	err := svc.ProcessExtractionTask(context.Background(), queuedQuestionGroupExtractionTask(t))

	if err == nil || !strings.Contains(err.Error(), "model adapter panic") {
		t.Fatalf("worker error = %v", err)
	}
	if repo.task.Status != types.ExamStructuringTaskStatusFailed || repo.task.Progress.Phase != types.ExamStructuringPhaseFailed {
		t.Fatalf("task after panic = %#v", repo.task)
	}
}
