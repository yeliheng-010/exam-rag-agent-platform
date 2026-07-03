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
	resourceHandler *handler.ExamResourceHandler,
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

		exam.GET("/resources", g.Viewer(), resourceHandler.ListResources)
		exam.POST("/resources/knowledge-bases/:kb_id/bind", g.Contributor(), resourceHandler.BindKnowledgeBase)
		exam.GET("/resources/knowledge-bases/:kb_id", g.Contributor(), resourceHandler.GetKnowledgeBaseBinding)
	}
}
