package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupRoutes 设置路由配置
func setupRoutes(appServer *AppServer) *gin.Engine {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 添加中间件
	router.Use(errorHandlingMiddleware())
	router.Use(corsMiddleware())

	// 健康检查
	router.GET("/health", healthHandler)
	router.GET("/swagger", swaggerRedirectHandler)
	router.HEAD("/swagger", swaggerRedirectHandler)
	router.GET("/swagger/index.html", swaggerIndexHandler)
	router.HEAD("/swagger/index.html", swaggerIndexHandler)
	router.GET("/swagger/doc.json", swaggerDocJSONHandler)
	router.HEAD("/swagger/doc.json", swaggerDocJSONHandler)
	router.GET("/swagger/doc.yaml", swaggerDocYAMLHandler)
	router.HEAD("/swagger/doc.yaml", swaggerDocYAMLHandler)

	// MCP 端点 - 使用官方 SDK 的 Streamable HTTP Handler
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return appServer.mcpServer
		},
		&mcp.StreamableHTTPOptions{
			JSONResponse: true, // 支持 JSON 响应
		},
	)
	router.Any("/mcp", gin.WrapH(mcpHandler))
	router.Any("/mcp/*path", gin.WrapH(mcpHandler))

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 登录认证
		api.GET("/login/status", appServer.checkLoginStatusHandler)
		api.GET("/login/qrcode", appServer.getLoginQrcodeHandler)
		api.GET("/login/qrcode/image", appServer.getLoginQrcodeImageHandler)
		api.DELETE("/login/cookies", appServer.deleteCookiesHandler)
		api.POST("/login/sync_cookies", appServer.syncCookiesHandler)

		// 内容发布
		api.POST("/publish", appServer.publishHandler)
		api.POST("/publish_video", appServer.publishVideoHandler)
		api.POST("/draft", appServer.saveDraftHandler)
		api.POST("/draft_video", appServer.saveVideoDraftHandler)

		// 内容发现
		api.GET("/feeds/list", appServer.listFeedsHandler)
		api.GET("/feeds/search", appServer.searchFeedsHandler)
		api.POST("/feeds/search", appServer.searchFeedsHandler)
		api.POST("/feeds/detail", appServer.getFeedDetailHandler)

		// 内容互动
		api.POST("/feeds/comment", appServer.postCommentHandler)
		api.POST("/feeds/comment/reply", appServer.replyCommentHandler)
		api.POST("/feeds/like", appServer.likeFeedHandler)
		api.POST("/feeds/favorite", appServer.favoriteFeedHandler)
		api.POST("/feeds/comment/like", appServer.likeCommentHandler)
		api.POST("/feeds/share", appServer.shareFeedHandler)

		// 内容管理
		api.DELETE("/feeds/:feed_id", appServer.deleteFeedHandler)
		api.DELETE("/feeds/:feed_id/comments/:comment_id", appServer.deleteCommentHandler)

		// 用户信息
		api.POST("/user/profile", appServer.userProfileHandler)
		api.POST("/user/follow", appServer.followUserHandler)
		api.GET("/user/me", appServer.myProfileHandler)
		api.GET("/user/me/feeds", appServer.getMyFeedsHandler)

		// 数据分析
		api.GET("/analytics/fans", appServer.getFanAnalyticsHandler)
		api.GET("/analytics/content", appServer.getContentAnalyticsHandler)
	}

	return router
}
