package types

import "time"

type ExamClassAnalyticsSummary struct {
	Class                  *ExamClass                        `json:"class"`
	TotalStudents          int                               `json:"total_students"`
	AssignmentCount        int                               `json:"assignment_count"`
	TotalAssignmentSlots   int                               `json:"total_assignment_slots"`
	StartedCount           int                               `json:"started_count"`
	CompletedCount         int                               `json:"completed_count"`
	CompletionRate         float64                           `json:"completion_rate"`
	AverageCorrectRate     float64                           `json:"average_correct_rate"`
	Members                []*ExamClassAnalyticsMember       `json:"members"`
	Assignments            []*ExamClassAnalyticsAssignment   `json:"assignments"`
	FrequentWrongQuestions []*ExamClassFrequentWrongQuestion `json:"frequent_wrong_questions"`
}

type ExamClassFrequentWrongQuestion struct {
	QuestionID           string  `json:"question_id"`
	GroupID              string  `json:"group_id"`
	AssignmentID         string  `json:"assignment_id"`
	QuestionNo           string  `json:"question_no"`
	Stem                 string  `json:"stem"`
	AnswerCount          int     `json:"answer_count"`
	WrongCount           int     `json:"wrong_count"`
	WrongRate            float64 `json:"wrong_rate"`
	AffectedStudentCount int     `json:"affected_student_count"`
}

type ExamClassAnalyticsMember struct {
	Member             *ExamClassMember `json:"member"`
	AssignmentCount    int              `json:"assignment_count"`
	StartedCount       int              `json:"started_count"`
	CompletedCount     int              `json:"completed_count"`
	CompletionRate     float64          `json:"completion_rate"`
	AverageCorrectRate float64          `json:"average_correct_rate"`
	LastActivityAt     *time.Time       `json:"last_activity_at,omitempty"`
}

type ExamClassAnalyticsAssignment struct {
	Assignment         *ExamClassAssignment `json:"assignment"`
	StartedCount       int                  `json:"started_count"`
	CompletedCount     int                  `json:"completed_count"`
	CompletionRate     float64              `json:"completion_rate"`
	AverageCorrectRate float64              `json:"average_correct_rate"`
}
