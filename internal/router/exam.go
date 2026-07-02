package router

import (
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterExamRoutes(
	r *gin.RouterGroup,
	domainHandler *handler.ExamDomainHandler,
	spaceHandler *handler.ExamSpaceHandler,
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

		exam.GET("/classes", g.Viewer(), classHandler.ListClasses)
		exam.POST("/classes", g.Contributor(), classHandler.CreateClass)
		exam.GET("/classes/:class_id", g.Viewer(), classHandler.GetClass)

		exam.GET("/question-banks", g.Viewer(), questionHandler.ListQuestionBanks)
		exam.POST("/question-banks", g.Contributor(), questionHandler.CreateQuestionBank)
		exam.GET("/question-banks/:bank_id", g.Viewer(), questionHandler.GetQuestionBank)

		exam.GET("/resources", g.Viewer(), resourceHandler.ListResources)
		exam.POST("/resources/knowledge-bases/:kb_id/bind", g.Contributor(), resourceHandler.BindKnowledgeBase)
		exam.GET("/resources/knowledge-bases/:kb_id", g.Viewer(), resourceHandler.GetKnowledgeBaseBinding)
	}
}
