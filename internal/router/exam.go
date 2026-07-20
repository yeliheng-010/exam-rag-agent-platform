package router

import (
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterExamRoutes(
	r *gin.RouterGroup,
	domainHandler *handler.ExamDomainHandler,
	spaceHandler *handler.ExamSpaceHandler,
	teacherApplicationHandler *handler.ExamTeacherApplicationHandler,
	classHandler *handler.ExamClassHandler,
	analyticsHandler *handler.ExamAnalyticsHandler,
	interventionHandler *handler.ExamInterventionHandler,
	assignmentHandler *handler.ExamAssignmentHandler,
	questionHandler *handler.ExamQuestionHandler,
	ragDiagnosticHandler *handler.ExamRAGDiagnosticHandler,
	ragEvaluationHandler *handler.ExamRAGEvaluationHandler,
	agentEvaluationHandler *handler.ExamAgentEvaluationHandler,
	evaluationCenterHandler *handler.ExamEvaluationCenterHandler,
	practiceHandler *handler.ExamPracticeHandler,
	questionDraftHandler *handler.ExamQuestionDraftHandler,
	questionGroupDraftHandler *handler.ExamQuestionGroupDraftHandler,
	resourceHandler *handler.ExamResourceHandler,
	materialHandler *handler.ExamMaterialHandler,
	g *rbacGuards,
) {
	exam := r.Group("/exam")
	{
		exam.GET("/domains", g.Viewer(), domainHandler.ListDomains)
		exam.GET("/domains/:domain_id/subjects", g.Viewer(), domainHandler.ListSubjects)

		exam.GET("/spaces", g.Viewer(), spaceHandler.ListSpaces)
		exam.POST("/spaces/personal/ensure", g.Viewer(), spaceHandler.EnsurePersonalSpace)

		exam.GET("/teacher-applications/me", g.Viewer(), teacherApplicationHandler.GetMine)
		exam.POST("/teacher-applications", g.Viewer(), teacherApplicationHandler.Apply)
		exam.GET("/admin/teacher-applications", g.Admin(), teacherApplicationHandler.List)
		exam.POST("/admin/teacher-applications/:application_id/approve", g.Admin(), teacherApplicationHandler.Approve)
		exam.POST("/admin/teacher-applications/:application_id/reject", g.Admin(), teacherApplicationHandler.Reject)

		exam.GET("/classes", g.Viewer(), classHandler.ListClasses)
		exam.POST("/classes", g.Viewer(), classHandler.CreateClass)
		exam.POST("/classes/join", g.Viewer(), classHandler.RequestJoinClass)
		exam.GET("/classes/:class_id", g.Viewer(), classHandler.GetClass)
		exam.PUT("/classes/:class_id", g.Viewer(), classHandler.UpdateClass)
		exam.POST("/classes/:class_id/archive", g.Viewer(), classHandler.ArchiveClass)
		exam.POST("/classes/:class_id/restore", g.Viewer(), classHandler.RestoreClass)
		exam.GET("/classes/:class_id/analytics", g.Viewer(), analyticsHandler.GetClassAnalytics)
		exam.GET("/classes/:class_id/practice-recommendations", g.Viewer(), interventionHandler.GetClassRecommendations)
		exam.GET("/classes/:class_id/members", g.Viewer(), classHandler.ListClassMembers)
		exam.POST("/classes/:class_id/members/:user_id/approve", g.Viewer(), classHandler.ApproveClassMember)
		exam.POST("/classes/:class_id/members/:user_id/reject", g.Viewer(), classHandler.RejectClassMember)
		exam.GET("/classes/:class_id/assignments", g.Viewer(), assignmentHandler.ListClassAssignments)
		exam.POST("/classes/:class_id/assignments", g.Viewer(), assignmentHandler.CreateAssignment)
		exam.PUT("/classes/:class_id/assignments/:assignment_id", g.Viewer(), assignmentHandler.UpdateAssignment)
		exam.POST("/classes/:class_id/assignments/:assignment_id/withdraw", g.Viewer(), assignmentHandler.WithdrawAssignment)
		exam.POST("/classes/:class_id/assignments/:assignment_id/republish", g.Viewer(), assignmentHandler.RepublishAssignment)
		exam.GET("/classes/:class_id/assignments/:assignment_id/progress", g.Viewer(), assignmentHandler.GetAssignmentProgress)
		exam.POST("/classes/:class_id/assignments/:assignment_id/reminders", g.Viewer(), assignmentHandler.SendAssignmentReminders)
		exam.GET("/assignments", g.Viewer(), assignmentHandler.ListMyAssignments)
		exam.POST("/assignments/:assignment_id/attempts", g.Viewer(), assignmentHandler.CreateAssignmentAttempt)
		exam.GET("/notifications", g.Viewer(), assignmentHandler.ListAssignmentNotifications)
		exam.POST("/notifications/read-all", g.Viewer(), assignmentHandler.MarkAllAssignmentNotificationsRead)
		exam.POST("/notifications/:notification_id/read", g.Viewer(), assignmentHandler.MarkAssignmentNotificationRead)

		exam.GET("/question-banks", g.Contributor(), questionHandler.ListQuestionBanks)
		exam.POST("/question-banks", g.Contributor(), questionHandler.CreateQuestionBank)
		exam.GET("/question-banks/:bank_id", g.Contributor(), questionHandler.GetQuestionBank)
		exam.GET("/question-banks/:bank_id/questions", g.Viewer(), questionHandler.ListQuestionDetails)
		exam.GET("/question-banks/:bank_id/question-groups", g.Viewer(), questionHandler.ListQuestionGroupDetails)
		exam.POST("/question-banks/:bank_id/rag-diagnostics", g.Contributor(), ragDiagnosticHandler.EvaluateQuestionBank)
		exam.POST("/question-banks/:bank_id/rag-evaluation-runs", g.Contributor(), ragEvaluationHandler.CreateRun)
		exam.GET("/question-banks/:bank_id/rag-evaluation-runs", g.Contributor(), ragEvaluationHandler.ListRuns)
		exam.GET("/question-banks/:bank_id/rag-evaluation-runs/:run_id", g.Contributor(), ragEvaluationHandler.GetRun)
		exam.POST("/question-banks/:bank_id/agent-evaluation-runs", g.Contributor(), agentEvaluationHandler.CreateRun)
		exam.GET("/question-banks/:bank_id/agent-evaluation-runs", g.Contributor(), agentEvaluationHandler.ListRuns)
		exam.GET("/question-banks/:bank_id/agent-evaluation-runs/:run_id", g.Contributor(), agentEvaluationHandler.GetRun)

		exam.GET("/evaluation-center", g.Contributor(), evaluationCenterHandler.GetCenter)
		exam.PUT("/evaluation-center/baseline", g.Contributor(), evaluationCenterHandler.SetBaseline)
		exam.GET("/evaluation-sets", g.Contributor(), evaluationCenterHandler.ListEvaluationSets)
		exam.POST("/evaluation-sets", g.Contributor(), evaluationCenterHandler.CreateEvaluationSet)
		exam.POST("/evaluation-sets/:set_id/versions", g.Contributor(), evaluationCenterHandler.CreateEvaluationSetVersion)
		exam.POST("/evaluation-sets/:set_id/runs", g.Contributor(), evaluationCenterHandler.RunEvaluationSet)

		exam.GET("/practice/question-groups", g.Viewer(), practiceHandler.ListQuestionGroups)
		exam.GET("/practice/recommendations", g.Viewer(), interventionHandler.GetStudentRecommendations)
		exam.GET("/practice/question-groups/:group_id", g.Viewer(), practiceHandler.GetQuestionGroup)
		exam.POST("/practice/question-groups/:group_id/attempts", g.Viewer(), practiceHandler.CreateAttempt)
		exam.GET("/practice/attempts", g.Viewer(), practiceHandler.ListAttempts)
		exam.GET("/practice/attempts/:attempt_id", g.Viewer(), practiceHandler.GetAttempt)
		exam.GET("/practice/attempts/:attempt_id/questions/:question_id/explanation-context", g.Viewer(), practiceHandler.GetExplanationContext)
		exam.POST("/practice/attempts/:attempt_id/answers", g.Viewer(), practiceHandler.SubmitAnswer)
		exam.POST("/practice/attempts/:attempt_id/complete", g.Viewer(), practiceHandler.CompleteAttempt)
		exam.PATCH("/practice/answers/:answer_id/review", g.Viewer(), practiceHandler.UpdateAnswerReview)
		exam.GET("/practice/wrong-questions", g.Viewer(), practiceHandler.ListWrongQuestions)

		exam.GET("/resources", g.Viewer(), resourceHandler.ListResources)
		exam.POST("/resources/knowledge-bases/:kb_id/bind", g.Contributor(), resourceHandler.BindKnowledgeBase)
		exam.GET("/resources/knowledge-bases/:kb_id", g.Contributor(), resourceHandler.GetKnowledgeBaseBinding)

		exam.GET("/materials", g.Viewer(), materialHandler.ListMaterials)
		exam.POST("/materials", g.Contributor(), materialHandler.RegisterMaterial)
		exam.GET("/structuring-tasks", g.Viewer(), materialHandler.ListStructuringTasks)
		exam.POST("/materials/:material_id/structuring-tasks", g.Contributor(), materialHandler.CreateStructuringTask)
		exam.POST("/structuring-tasks/:task_id/extract", g.Contributor(), questionDraftHandler.ExtractDrafts)
		exam.GET("/structuring-tasks/:task_id/drafts", g.Contributor(), questionDraftHandler.ListDrafts)
		exam.POST("/structuring-tasks/:task_id/group-extract", g.Contributor(), questionGroupDraftHandler.ExtractDrafts)
		exam.GET("/structuring-tasks/:task_id/group-drafts", g.Contributor(), questionGroupDraftHandler.ListDrafts)
		exam.PATCH("/question-drafts/:draft_id", g.Contributor(), questionDraftHandler.UpdateDraft)
		exam.POST("/question-drafts/:draft_id/approve", g.Contributor(), questionDraftHandler.ApproveDraft)
		exam.POST("/question-drafts/:draft_id/reject", g.Contributor(), questionDraftHandler.RejectDraft)
		exam.PATCH("/question-group-drafts/:draft_id", g.Contributor(), questionGroupDraftHandler.UpdateDraft)
		exam.POST("/question-group-drafts/:draft_id/approve", g.Contributor(), questionGroupDraftHandler.ApproveDraft)
		exam.POST("/question-group-drafts/:draft_id/reject", g.Contributor(), questionGroupDraftHandler.RejectDraft)
	}
}
