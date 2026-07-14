package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const maxFrequentWrongQuestions = 20

type analyticsWrongQuestionBucket struct {
	item     types.ExamClassFrequentWrongQuestion
	students map[string]bool
}

type analyticsAttemptContext struct {
	UserID       string
	GroupID      string
	AssignmentID string
}

func newAnalyticsAttemptContext(attempt *types.ExamPracticeAttempt) analyticsAttemptContext {
	context := analyticsAttemptContext{UserID: attempt.UserID, GroupID: attempt.GroupID}
	if attempt.AssignmentID != nil {
		context.AssignmentID = *attempt.AssignmentID
	}
	return context
}

func (s *examAnalyticsService) frequentWrongQuestions(
	ctx context.Context,
	tenantID uint64,
	attemptContexts map[string]analyticsAttemptContext,
) ([]*types.ExamClassFrequentWrongQuestion, error) {
	attemptIDs := make([]string, 0, len(attemptContexts))
	for attemptID := range attemptContexts {
		attemptIDs = append(attemptIDs, attemptID)
	}
	answers, err := s.practiceRepo.ListAnswersByAttempts(ctx, tenantID, attemptIDs)
	if err != nil {
		return nil, err
	}
	return aggregateFrequentWrongQuestions(answers, attemptContexts), nil
}

func aggregateFrequentWrongQuestions(
	answers []*types.ExamPracticeAnswer,
	attemptContexts map[string]analyticsAttemptContext,
) []*types.ExamClassFrequentWrongQuestion {
	buckets := map[string]*analyticsWrongQuestionBucket{}
	for _, answer := range answers {
		attemptContext, ok := attemptContexts[answerAttemptID(answer)]
		if !ok || answer == nil || strings.TrimSpace(answer.QuestionID) == "" {
			continue
		}
		bucket := buckets[answer.QuestionID]
		if bucket == nil {
			bucket = newAnalyticsWrongQuestionBucket(answer, attemptContext)
			buckets[answer.QuestionID] = bucket
		}
		bucket.item.AnswerCount++
		if !answer.IsCorrect {
			bucket.item.WrongCount++
			bucket.students[attemptContext.UserID] = true
		}
	}
	return finalizeFrequentWrongQuestions(buckets)
}

func answerAttemptID(answer *types.ExamPracticeAnswer) string {
	if answer == nil {
		return ""
	}
	return answer.AttemptID
}

func newAnalyticsWrongQuestionBucket(
	answer *types.ExamPracticeAnswer,
	attemptContext analyticsAttemptContext,
) *analyticsWrongQuestionBucket {
	return &analyticsWrongQuestionBucket{
		item: types.ExamClassFrequentWrongQuestion{
			QuestionID:   answer.QuestionID,
			GroupID:      attemptContext.GroupID,
			AssignmentID: attemptContext.AssignmentID,
			QuestionNo:   firstAnalyticsText(answer.QuestionNo, analyticsSnapshotText(answer.QuestionSnapshot, "question_no")),
			Stem:         analyticsSnapshotText(answer.QuestionSnapshot, "stem"),
		},
		students: map[string]bool{},
	}
}

func finalizeFrequentWrongQuestions(buckets map[string]*analyticsWrongQuestionBucket) []*types.ExamClassFrequentWrongQuestion {
	out := make([]*types.ExamClassFrequentWrongQuestion, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket.item.WrongCount == 0 {
			continue
		}
		bucket.item.WrongRate = float64(bucket.item.WrongCount) / float64(bucket.item.AnswerCount)
		bucket.item.AffectedStudentCount = len(bucket.students)
		item := bucket.item
		out = append(out, &item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WrongCount != out[j].WrongCount {
			return out[i].WrongCount > out[j].WrongCount
		}
		if out[i].WrongRate != out[j].WrongRate {
			return out[i].WrongRate > out[j].WrongRate
		}
		return frequentWrongQuestionComesFirst(out[i], out[j])
	})
	if len(out) > maxFrequentWrongQuestions {
		out = out[:maxFrequentWrongQuestions]
	}
	return out
}

func frequentWrongQuestionComesFirst(
	left *types.ExamClassFrequentWrongQuestion,
	right *types.ExamClassFrequentWrongQuestion,
) bool {
	leftNo, leftErr := strconv.Atoi(strings.TrimSpace(left.QuestionNo))
	rightNo, rightErr := strconv.Atoi(strings.TrimSpace(right.QuestionNo))
	if leftErr == nil && rightErr == nil && leftNo != rightNo {
		return leftNo > rightNo
	}
	if left.QuestionNo != right.QuestionNo {
		return left.QuestionNo > right.QuestionNo
	}
	return left.QuestionID < right.QuestionID
}

func analyticsSnapshotText(snapshot types.JSONMap, key string) string {
	if snapshot == nil {
		return ""
	}
	value, ok := snapshot[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func firstAnalyticsText(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}
