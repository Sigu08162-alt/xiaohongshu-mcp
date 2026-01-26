package main

import (
	"encoding/base64"
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

// likeFeedHandler 点赞/取消点赞笔记
// @Summary 点赞笔记
// @Description 为指定笔记点赞或取消点赞
// @Tags 内容互动
// @Accept json
// @Produce json
// @Param request body LikeFeedRequest true "点赞请求参数"
// @Success 200 {object} SuccessResponse "操作成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "操作失败"
// @Router /feeds/like [post]
func (s *AppServer) likeFeedHandler(c *gin.Context) {
	var req LikeFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	var result *ActionResult
	var err error

	if req.Unlike {
		result, err = s.xiaohongshuService.UnlikeFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	} else {
		result, err = s.xiaohongshuService.LikeFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIKE_FEED_FAILED",
			"点赞操作失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// favoriteFeedHandler 收藏/取消收藏笔记
// @Summary 收藏笔记
// @Description 收藏或取消收藏指定笔记
// @Tags 内容互动
// @Accept json
// @Produce json
// @Param request body FavoriteFeedRequest true "收藏请求参数"
// @Success 200 {object} SuccessResponse "操作成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "操作失败"
// @Router /feeds/favorite [post]
func (s *AppServer) favoriteFeedHandler(c *gin.Context) {
	var req FavoriteFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	var result *ActionResult
	var err error

	if req.Unfavorite {
		result, err = s.xiaohongshuService.UnfavoriteFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	} else {
		result, err = s.xiaohongshuService.FavoriteFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, "FAVORITE_FEED_FAILED",
			"收藏操作失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// likeCommentHandler 点赞/取消点赞评论
// @Summary 点赞评论
// @Description 为指定评论点赞或取消点赞
// @Tags 内容互动
// @Accept json
// @Produce json
// @Param request body LikeCommentRequest true "点赞评论请求参数"
// @Success 200 {object} SuccessResponse "操作成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "操作失败"
// @Router /feeds/comment/like [post]
func (s *AppServer) likeCommentHandler(c *gin.Context) {
	var req LikeCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	var result *ActionResult
	var err error

	if req.Unlike {
		result, err = s.xiaohongshuService.UnlikeComment(c.Request.Context(), req.FeedID, req.XsecToken, req.CommentID, req.UserID)
	} else {
		result, err = s.xiaohongshuService.LikeComment(c.Request.Context(), req.FeedID, req.XsecToken, req.CommentID, req.UserID)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIKE_COMMENT_FAILED",
			"点赞评论操作失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// followUserHandler 关注/取关用户
// @Summary 关注用户
// @Description 关注或取关指定用户
// @Tags 内容互动
// @Accept json
// @Produce json
// @Param request body FollowUserRequest true "关注请求参数"
// @Success 200 {object} SuccessResponse "操作成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "操作失败"
// @Router /user/follow [post]
func (s *AppServer) followUserHandler(c *gin.Context) {
	var req FollowUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	var result *ActionResult
	var err error

	if req.Unfollow {
		result, err = s.xiaohongshuService.UnfollowUser(c.Request.Context(), req.UserID, req.XsecToken)
	} else {
		result, err = s.xiaohongshuService.FollowUser(c.Request.Context(), req.UserID, req.XsecToken)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, "FOLLOW_USER_FAILED",
			"关注操作失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// deleteFeedHandler 删除笔记
// @Summary 删除笔记
// @Description 删除自己发布的笔记
// @Tags 内容管理
// @Accept json
// @Produce json
// @Param feed_id path string true "笔记ID"
// @Param request body DeleteFeedRequest true "删除请求参数"
// @Success 200 {object} SuccessResponse "删除成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "删除失败"
// @Router /feeds/{feed_id} [delete]
func (s *AppServer) deleteFeedHandler(c *gin.Context) {
	feedID := c.Param("feed_id")

	var req DeleteFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 使用路径参数中的 feed_id（优先级更高）
	if feedID != "" {
		req.FeedID = feedID
	}

	result, err := s.xiaohongshuService.DeleteFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DELETE_FEED_FAILED",
			"删除笔记失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// deleteCommentHandler 删除评论
// @Summary 删除评论
// @Description 删除自己发表的评论
// @Tags 内容管理
// @Accept json
// @Produce json
// @Param feed_id path string true "笔记ID"
// @Param comment_id path string true "评论ID"
// @Param request body DeleteCommentRequest true "删除请求参数"
// @Success 200 {object} SuccessResponse "删除成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "删除失败"
// @Router /feeds/{feed_id}/comments/{comment_id} [delete]
func (s *AppServer) deleteCommentHandler(c *gin.Context) {
	feedID := c.Param("feed_id")
	commentID := c.Param("comment_id")

	var req DeleteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 使用路径参数（优先级更高）
	if feedID != "" {
		req.FeedID = feedID
	}
	if commentID != "" {
		req.CommentID = commentID
	}

	result, err := s.xiaohongshuService.DeleteComment(c.Request.Context(), req.FeedID, req.XsecToken, req.CommentID, req.UserID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DELETE_COMMENT_FAILED",
			"删除评论失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// saveDraftHandler 保存图文草稿
// @Summary 保存图文草稿
// @Description 保存小红书图文草稿（暂存离开，不立即发布）
// @Tags 内容发布
// @Accept json
// @Produce json
// @Param request body SaveDraftRequest true "草稿请求参数"
// @Success 200 {object} SuccessResponse "保存成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "保存失败"
// @Router /draft [post]
func (s *AppServer) saveDraftHandler(c *gin.Context) {
	var req SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 转换为 interface{} 数组
	images := make([]interface{}, len(req.Images))
	for i, img := range req.Images {
		images[i] = img
	}

	tags := make([]interface{}, len(req.Tags))
	for i, tag := range req.Tags {
		tags[i] = tag
	}

	// 调用 MCP handler
	argsMap := map[string]interface{}{
		"title":   req.Title,
		"content": req.Content,
		"images":  images,
		"tags":    tags,
	}

	result := s.handleSaveDraft(c.Request.Context(), argsMap)
	if result.IsError {
		respondError(c, http.StatusInternalServerError, "SAVE_DRAFT_FAILED",
			"保存草稿失败", result.Content[0].Text)
		return
	}

	respondSuccess(c, map[string]any{
		"message": result.Content[0].Text,
	}, "保存草稿成功")
}

// saveVideoDraftHandler 保存视频草稿
// @Summary 保存视频草稿
// @Description 保存小红书视频草稿（暂存离开，不立即发布）
// @Tags 内容发布
// @Accept json
// @Produce json
// @Param request body SaveVideoDraftRequest true "视频草稿请求参数"
// @Success 200 {object} SuccessResponse "保存成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "保存失败"
// @Router /draft_video [post]
func (s *AppServer) saveVideoDraftHandler(c *gin.Context) {
	var req SaveVideoDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 转换为 interface{} 数组
	tags := make([]interface{}, len(req.Tags))
	for i, tag := range req.Tags {
		tags[i] = tag
	}

	// 调用 MCP handler
	argsMap := map[string]interface{}{
		"title":   req.Title,
		"content": req.Content,
		"video":   req.Video,
		"tags":    tags,
	}

	result := s.handleSaveVideoDraft(c.Request.Context(), argsMap)
	if result.IsError {
		respondError(c, http.StatusInternalServerError, "SAVE_VIDEO_DRAFT_FAILED",
			"保存视频草稿失败", result.Content[0].Text)
		return
	}

	respondSuccess(c, map[string]any{
		"message": result.Content[0].Text,
	}, "保存视频草稿成功")
}

// shareFeedHandler 分享笔记
// @Summary 分享笔记
// @Description 分享指定笔记，获取分享链接
// @Tags 内容互动
// @Accept json
// @Produce json
// @Param request body ShareFeedRequest true "分享请求参数"
// @Success 200 {object} SuccessResponse "分享成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "分享失败"
// @Router /feeds/share [post]
func (s *AppServer) shareFeedHandler(c *gin.Context) {
	var req ShareFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	shareLink, err := s.xiaohongshuService.ShareFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SHARE_FEED_FAILED",
			"分享笔记失败", err.Error())
		return
	}

	result := ShareFeedResponse{
		FeedID:    req.FeedID,
		ShareLink: shareLink,
		Success:   true,
		Message:   "分享成功",
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, result.Message)
}

// getMyFeedsHandler 获取自己的笔记列表
// @Summary 获取我的笔记
// @Description 获取当前登录用户发布的笔记列表
// @Tags 用户信息
// @Accept json
// @Produce json
// @Param limit query int false "限制返回数量（默认20，最大100）"
// @Param user_id query string false "可选，传入则读取该用户主页"
// @Success 200 {object} SuccessResponse "笔记列表"
// @Failure 500 {object} ErrorResponse "获取失败"
// @Router /user/me/feeds [get]
func (s *AppServer) getMyFeedsHandler(c *gin.Context) {
	var req GetMyFeedsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// 查询参数绑定失败不是致命错误，使用默认值
		req.Limit = 20
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	feeds, err := s.xiaohongshuService.GetMyFeeds(c.Request.Context(), req.Limit, req.UserID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_MY_FEEDS_FAILED",
			"获取我的笔记失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, map[string]any{
		"feeds": feeds,
		"count": len(feeds),
	}, "获取我的笔记成功")
}

// syncCookiesHandler 上传cookies
// @Summary 上传Cookies
// @Description 上传cookies JSON并写入服务端文件（推荐先本地有头登录后上传）
// @Tags 登录认证
// @Accept json
// @Produce json
// @Param request body SyncCookiesRequest true "Cookies数据"
// @Success 200 {object} SuccessResponse "上传成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "上传失败"
// @Router /login/sync_cookies [post]
func (s *AppServer) syncCookiesHandler(c *gin.Context) {
	var req SyncCookiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	if req.CookiesBase64 == "" && req.CookiesJSON == "" {
		respondError(c, http.StatusBadRequest, "MISSING_COOKIES",
			"缺少cookies数据", "cookies_base64 or cookies_json is required")
		return
	}

	var cookiesData []byte
	var err error

	if req.CookiesBase64 != "" {
		// 使用 Base64 编码的数据
		cookiesData, err = base64.StdEncoding.DecodeString(req.CookiesBase64)
		if err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_BASE64",
				"Base64解码失败", err.Error())
			return
		}
	} else {
		// 使用 JSON 字符串
		cookiesData = []byte(req.CookiesJSON)
	}

	cookiePath, fileSize, err := s.xiaohongshuService.SyncCookies(c.Request.Context(), cookiesData)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SYNC_COOKIES_FAILED",
			"上传cookies失败", err.Error())
		return
	}

	result := SyncCookiesResponse{
		Success:    true,
		CookiePath: cookiePath,
		FileSize:   fileSize,
		Message:    "Cookies已成功上传并写入服务端文件",
	}

	respondSuccess(c, result, result.Message)
}

// getFanAnalyticsHandler 获取粉丝分析
// @Summary 获取粉丝分析
// @Description 获取粉丝分析数据，包括粉丝概览、粉丝画像和活跃粉丝列表
// @Tags 数据分析
// @Accept json
// @Produce json
// @Param period query string false "统计周期（7d或30d，默认7d）"
// @Success 200 {object} SuccessResponse "粉丝分析数据"
// @Failure 500 {object} ErrorResponse "获取失败"
// @Router /analytics/fans [get]
func (s *AppServer) getFanAnalyticsHandler(c *gin.Context) {
	var req GetFanAnalyticsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Period = "7d"
	}

	if req.Period == "" {
		req.Period = "7d"
	}

	analytics, err := s.xiaohongshuService.GetFanAnalytics(c.Request.Context(), req.Period)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_FAN_ANALYTICS_FAILED",
			"获取粉丝分析失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, analytics, "获取粉丝分析成功")
}

// getContentAnalyticsHandler 获取内容分析
// @Summary 获取内容分析
// @Description 获取内容分析数据，包括每篇笔记的详细指标
// @Tags 数据分析
// @Accept json
// @Produce json
// @Param limit query int false "限制返回笔记数量（默认20，最大100）"
// @Param sort_by query string false "排序字段（exposure/views/likes/comments等）"
// @Param sort_order query string false "排序方向（asc/desc，默认desc）"
// @Success 200 {object} SuccessResponse "内容分析数据"
// @Failure 500 {object} ErrorResponse "获取失败"
// @Router /analytics/content [get]
func (s *AppServer) getContentAnalyticsHandler(c *gin.Context) {
	var req GetContentAnalyticsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Limit = 20
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// 转换为 xiaohongshu 包的类型
	var sortBy xiaohongshu.SortField
	if req.SortBy != "" {
		sortBy = xiaohongshu.SortField(req.SortBy)
	}

	var sortOrder xiaohongshu.SortOrder
	if req.SortOrder == "asc" {
		sortOrder = xiaohongshu.SortAsc
	} else {
		sortOrder = xiaohongshu.SortDesc
	}

	analytics, err := s.xiaohongshuService.GetContentAnalytics(c.Request.Context(), req.Limit, sortBy, sortOrder)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_CONTENT_ANALYTICS_FAILED",
			"获取内容分析失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, analytics, "获取内容分析成功")
}
