package types

type ExamPracticeRecommendationScope string
type ExamPracticeRecommendationReasonCode string

const (
	ExamPracticeRecommendationScopeClass   ExamPracticeRecommendationScope = "class"
	ExamPracticeRecommendationScopeStudent ExamPracticeRecommendationScope = "student"

	ExamPracticeRecommendationReasonTargetedReview   ExamPracticeRecommendationReasonCode = "targeted_review"
	ExamPracticeRecommendationReasonSameSubject      ExamPracticeRecommendationReasonCode = "same_subject"
	ExamPracticeRecommendationReasonSameGroupType    ExamPracticeRecommendationReasonCode = "same_group_type"
	ExamPracticeRecommendationReasonSameDomain       ExamPracticeRecommendationReasonCode = "same_domain"
	ExamPracticeRecommendationReasonLowAccuracyRetry ExamPracticeRecommendationReasonCode = "low_accuracy_retry"
	ExamPracticeRecommendationReasonSupplemental     ExamPracticeRecommendationReasonCode = "supplemental_practice"
)

type ExamPracticeRecommendationRequest struct {
	SpaceID         string `json:"space_id,omitempty" form:"space_id"`
	Limit           int    `json:"limit,omitempty" form:"limit"`
	IncludeAssigned bool   `json:"include_assigned,omitempty" form:"include_assigned"`
}

type ExamPracticeDiagnosisEvidence struct {
	QuestionID           string                     `json:"question_id"`
	GroupID              string                     `json:"group_id"`
	QuestionNo           string                     `json:"question_no"`
	Stem                 string                     `json:"stem"`
	WrongRate            float64                    `json:"wrong_rate"`
	AffectedStudentCount int                        `json:"affected_student_count"`
	ReviewStatus         PracticeAnswerReviewStatus `json:"review_status,omitempty"`
}

type ExamPracticeRecommendationReason struct {
	Code  ExamPracticeRecommendationReasonCode `json:"code"`
	Score int                                  `json:"score"`
}

type ExamPracticeRecommendation struct {
	Group    *QuestionGroupPracticeSummary      `json:"group"`
	Score    int                                `json:"score"`
	Reasons  []ExamPracticeRecommendationReason `json:"reasons"`
	Evidence []ExamPracticeDiagnosisEvidence    `json:"evidence"`
	Assigned bool                               `json:"assigned"`
}

type ExamPracticeRecommendationResult struct {
	Scope           ExamPracticeRecommendationScope `json:"scope"`
	Class           *ExamClass                      `json:"class,omitempty"`
	Diagnosis       []ExamPracticeDiagnosisEvidence `json:"diagnosis"`
	Recommendations []*ExamPracticeRecommendation   `json:"recommendations"`
	Warnings        []string                        `json:"warnings"`
}
