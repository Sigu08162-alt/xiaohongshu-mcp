package main

import (
	"net/http"

	"github.com/xpzouying/xiaohongshu-mcp/cookies"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// respondError 返回错误响应
func respondError(c *gin.Context, statusCode int, code, message string, details any) {
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	logrus.Errorf("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), statusCode)

	c.JSON(statusCode, response)
}

// respondSuccess 返回成功响应
func respondSuccess(c *gin.Context, data any, message string) {
	response := SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	logrus.Infof("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), http.StatusOK)

	c.JSON(http.StatusOK, response)
}

// checkLoginStatusHandler 检查登录状态
// @Summary 检查登录状态
// @Description 检查当前是否已登录小红书
// @Tags 登录认证
// @Produce json
// @Success 200 {object} SuccessResponse "登录状态信息"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /login/status [get]
func (s *AppServer) checkLoginStatusHandler(c *gin.Context) {
	status, err := s.xiaohongshuService.CheckLoginStatus(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "STATUS_CHECK_FAILED",
			"检查登录状态失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, status, "检查登录状态成功")
}

// getLoginQrcodeHandler 处理 [GET /api/login/qrcode] 请求。
// 用于生成并返回登录二维码（Base64 图片 + 超时时间），供前端展示给用户扫码登录。
// @Summary 获取登录二维码
// @Description 生成小红书登录二维码，返回 Base64 编码的图片和超时时间
// @Tags 登录认证
// @Produce json
// @Success 200 {object} SuccessResponse "二维码信息"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /login/qrcode [get]
func (s *AppServer) getLoginQrcodeHandler(c *gin.Context) {
	result, err := s.xiaohongshuService.GetLoginQrcode(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "STATUS_CHECK_FAILED",
			"获取登录二维码失败", err.Error())
		return
	}

	respondSuccess(c, result, "获取登录二维码成功")
}

// deleteCookiesHandler 删除 cookies，重置登录状态
// @Summary 删除 Cookies
// @Description 删除本地保存的 cookies 文件，重置登录状态
// @Tags 登录认证
// @Produce json
// @Success 200 {object} SuccessResponse "删除成功"
// @Failure 500 {object} ErrorResponse "删除失败"
// @Router /login/cookies [delete]
func (s *AppServer) deleteCookiesHandler(c *gin.Context) {
	err := s.xiaohongshuService.DeleteCookies(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DELETE_COOKIES_FAILED",
			"删除 cookies 失败", err.Error())
		return
	}

	cookiePath := cookies.GetCookiesFilePath()
	respondSuccess(c, map[string]interface{}{
		"cookie_path": cookiePath,
		"message":     "Cookies 已成功删除，登录状态已重置。下次操作时需要重新登录。",
	}, "删除 cookies 成功")
}

// publishHandler 发布内容
// @Summary 发布图文笔记
// @Description 发布小红书图文内容，支持标题、正文、图片、标签等
// @Tags 内容发布
// @Accept json
// @Produce json
// @Param request body PublishRequest true "发布请求参数"
// @Success 200 {object} SuccessResponse "发布成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "发布失败"
// @Router /publish [post]
func (s *AppServer) publishHandler(c *gin.Context) {
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 执行发布
	result, err := s.xiaohongshuService.PublishContent(c.Request.Context(), &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_FAILED",
			"发布失败", err.Error())
		return
	}

	respondSuccess(c, result, "发布成功")
}

// publishVideoHandler 发布视频内容
// @Summary 发布视频笔记
// @Description 发布小红书视频内容
// @Tags 内容发布
// @Accept json
// @Produce json
// @Param request body PublishVideoRequest true "视频发布请求参数"
// @Success 200 {object} SuccessResponse "发布成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "发布失败"
// @Router /publish_video [post]
func (s *AppServer) publishVideoHandler(c *gin.Context) {
	var req PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 执行视频发布
	result, err := s.xiaohongshuService.PublishVideo(c.Request.Context(), &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_VIDEO_FAILED",
			"视频发布失败", err.Error())
		return
	}

	respondSuccess(c, result, "视频发布成功")
}

// listFeedsHandler 获取Feeds列表
// @Summary 获取首页 Feeds 列表
// @Description 获取小红书首页推荐 Feed 流
// @Tags 内容发现
// @Produce json
// @Success 200 {object} SuccessResponse "Feed 列表"
// @Failure 500 {object} ErrorResponse "获取失败"
// @Router /feeds/list [get]
func (s *AppServer) listFeedsHandler(c *gin.Context) {
	// 获取 Feeds 列表
	result, err := s.xiaohongshuService.ListFeeds(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIST_FEEDS_FAILED",
			"获取Feeds列表失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, "获取Feeds列表成功")
}

// searchFeedsHandler 搜索Feeds
// @Summary 搜索笔记
// @Description 搜索小红书笔记，支持关键词和多维度筛选
// @Tags 内容发现
// @Accept json
// @Produce json
// @Param keyword query string false "搜索关键词(GET)"
// @Param request body SearchFeedsRequest false "搜索请求参数(POST)"
// @Success 200 {object} SuccessResponse "搜索结果"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "搜索失败"
// @Router /feeds/search [get]
// @Router /feeds/search [post]
func (s *AppServer) searchFeedsHandler(c *gin.Context) {
	var keyword string
	var filters xiaohongshu.FilterOption

	switch c.Request.Method {
	case http.MethodPost:
		// 对于POST请求，从JSON中获取keyword
		var searchReq SearchFeedsRequest
		if err := c.ShouldBindJSON(&searchReq); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
				"请求参数错误", err.Error())
			return
		}
		keyword = searchReq.Keyword
		filters = searchReq.Filters
	default:
		keyword = c.Query("keyword")
	}

	if keyword == "" {
		respondError(c, http.StatusBadRequest, "MISSING_KEYWORD",
			"缺少关键词参数", "keyword parameter is required")
		return
	}

	// 搜索 Feeds
	result, err := s.xiaohongshuService.SearchFeeds(c.Request.Context(), keyword, filters)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SEARCH_FEEDS_FAILED",
			"搜索Feeds失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, "搜索Feeds成功")
}

// getFeedDetailHandler 获取Feed详情
// @Summary 获取笔记详情
// @Description 获取小红书笔记完整详情和评论列表
// @Tags 内容发现
// @Accept json
// @Produce json
// @Param request body FeedDetailRequest true "详情请求参数"
// @Success 200 {object} SuccessResponse "笔记详情"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "获取失败"
// @Router /feeds/detail [post]
func (s *AppServer) getFeedDetailHandler(c *gin.Context) {
	var req FeedDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	var result *FeedDetailResponse
	var err error

	if req.CommentConfig != nil {
		// 使用配置参数
		config := xiaohongshu.CommentLoadConfig{
			ClickMoreReplies:    req.CommentConfig.ClickMoreReplies,
			MaxRepliesThreshold: req.CommentConfig.MaxRepliesThreshold,
			MaxCommentItems:     req.CommentConfig.MaxCommentItems,
			ScrollSpeed:         req.CommentConfig.ScrollSpeed,
		}
		result, err = s.xiaohongshuService.GetFeedDetailWithConfig(c.Request.Context(), req.FeedID, req.XsecToken, req.LoadAllComments, config)
	} else {
		// 使用默认配置
		result, err = s.xiaohongshuService.GetFeedDetail(c.Request.Context(), req.FeedID, req.XsecToken, req.LoadAllComments)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_FEED_DETAIL_FAILED",
			"获取Feed详情失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, "获取Feed详情成功")
}

// userProfileHandler 用户主页
// @Summary 获取用户主页
// @Description 获取指定用户的主页信息和笔记列表
// @Tags 用户信息
// @Accept json
// @Produce json
// @Param request body UserProfileRequest true "用户主页请求参数"
// @Success 200 {object} SuccessResponse "用户主页信息"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "获取失败"
// @Router /user/profile [post]
func (s *AppServer) userProfileHandler(c *gin.Context) {
	var req UserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 获取用户信息
	result, err := s.xiaohongshuService.UserProfile(c.Request.Context(), req.UserID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_USER_PROFILE_FAILED",
			"获取用户主页失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, map[string]any{"data": result}, "result.Message")
}

// postCommentHandler 发表评论到Feed
// @Summary 发表评论
// @Description 在小红书笔记下发表评论
// @Tags 内容互动
// @Accept json
// @Produce json
// @Param request body PostCommentRequest true "评论请求参数"
// @Success 200 {object} SuccessResponse "评论成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "评论失败"
// @Router /feeds/comment [post]
func (s *AppServer) postCommentHandler(c *gin.Context) {
	var req PostCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 发表评论
	result, err := s.xiaohongshuService.PostCommentToFeed(c.Request.Context(), req.FeedID, req.XsecToken, req.Content)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "POST_COMMENT_FAILED",
			"发表评论失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// replyCommentHandler 回复指定评论
// @Summary 回复评论
// @Description 回复笔记下的指定评论
// @Tags 内容互动
// @Accept json
// @Produce json
// @Param request body ReplyCommentRequest true "回复请求参数"
// @Success 200 {object} SuccessResponse "回复成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "回复失败"
// @Router /feeds/comment/reply [post]
func (s *AppServer) replyCommentHandler(c *gin.Context) {
	var req ReplyCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	result, err := s.xiaohongshuService.ReplyCommentToFeed(c.Request.Context(), req.FeedID, req.XsecToken, req.CommentID, req.UserID, req.Content)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "REPLY_COMMENT_FAILED",
			"回复评论失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// healthHandler 健康检查
func healthHandler(c *gin.Context) {
	respondSuccess(c, map[string]any{
		"status":    "healthy",
		"service":   "xiaohongshu-mcp",
		"account":   "ai-report",
		"timestamp": "now",
	}, "服务正常")
}

// myProfileHandler 我的信息
// @Summary 获取我的资料
// @Description 获取当前登录用户的主页信息
// @Tags 用户信息
// @Produce json
// @Success 200 {object} SuccessResponse "用户信息"
// @Failure 500 {object} ErrorResponse "获取失败"
// @Router /user/me [get]
func (s *AppServer) myProfileHandler(c *gin.Context) {
	// 获取当前登录用户信息
	result, err := s.xiaohongshuService.GetMyProfile(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_MY_PROFILE_FAILED",
			"获取我的主页失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, map[string]any{"data": result}, "获取我的主页成功")
}
