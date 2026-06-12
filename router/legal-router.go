package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetLegalDocumentRouter(router *gin.Engine) {
	router.GET("/user-agreement", func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		controller.GetUserAgreementHTML(c)
	})
	router.GET("/privacy-policy", func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		controller.GetPrivacyPolicyHTML(c)
	})
}
