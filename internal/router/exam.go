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
	questionHandler *handler.ExamQuestionHandler,
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
		exam.GET("/classes/:class_id/members", g.Viewer(), classHandler.ListClassMembers)
		exam.POST("/classes/:class_id/members/:user_id/approve", g.Viewer(), classHandler.ApproveClassMember)
		exam.POST("/classes/:class_id/members/:user_id/reject", g.Viewer(), classHandler.RejectClassMember)

		exam.GET("/question-banks", g.Contributor(), questionHandler.ListQuestionBanks)
		exam.POST("/question-banks", g.Contributor(), questionHandler.CreateQuestionBank)
		exam.GET("/question-banks/:bank_id", g.Contributor(), questionHandler.GetQuestionBank)
		exam.GET("/question-banks/:bank_id/questions", g.Viewer(), questionHandler.ListQuestionDetails)
		exam.GET("/question-banks/:bank_id/question-groups", g.Viewer(), questionHandler.ListQuestionGroupDetails)

		exam.GET("/practice/question-groups", g.Viewer(), practiceHandler.ListQuestionGroups)
		exam.GET("/practice/question-groups/:group_id", g.Viewer(), practiceHandler.GetQuestionGroup)
		exam.POST("/practice/question-groups/:group_id/attempts", g.Viewer(), practiceHandler.CreateAttempt)
		exam.GET("/practice/attempts", g.Viewer(), practiceHandler.ListAttempts)
		exam.GET("/practice/attempts/:attempt_id", g.Viewer(), practiceHandler.GetAttempt)
		exam.POST("/practice/attempts/:attempt_id/answers", g.Viewer(), practiceHandler.SubmitAnswer)
		exam.POST("/practice/attempts/:attempt_id/complete", g.Viewer(), practiceHandler.CompleteAttempt)
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
