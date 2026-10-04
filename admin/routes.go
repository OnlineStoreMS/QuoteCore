package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *Handlers) {
	g.GET("/dashboard/stats", h.DashboardStats)

	g.GET("/quote-templates", h.ListTemplates)
	g.POST("/quote-templates", h.CreateTemplate)
	g.GET("/quote-templates/:id", h.GetTemplate)
	g.PUT("/quote-templates/:id", h.UpdateTemplate)
	g.DELETE("/quote-templates/:id", h.DeleteTemplate)

	g.GET("/quotes", h.ListQuotes)
	g.POST("/quotes", h.CreateQuote)
	g.GET("/quotes/:id", h.GetQuote)
	g.PUT("/quotes/:id", h.UpdateQuote)
	g.DELETE("/quotes/:id", h.DeleteQuote)
	g.POST("/quotes/:id/void", h.VoidQuote)
	g.POST("/quotes/:id/copy", h.CopyQuote)
	g.POST("/quotes/:id/share", h.EnsureShare)
	g.POST("/quotes/:id/customer-share", h.EnsureCustomerShare)
	g.GET("/quotes/:id/second-edit", h.GetSecondEdit)
	g.POST("/quotes/:id/second-edit/approve", h.ApproveSecondEdit)

	g.GET("/customers/search", h.SearchCustomers)
	g.GET("/product-skus/search", h.SearchProductSkus)
}
