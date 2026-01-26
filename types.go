package main

import "github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"

// HTTP API 响应类型

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// MCP 相关类型（用于内部转换）

// MCPToolResult MCP 工具结果（内部使用）
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent MCP 内容（内部使用）
type MCPContent struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// CommentLoadConfig 评论加载配置
type CommentLoadConfig struct {
	// 是否点击"更多回复"按钮
	ClickMoreReplies bool `json:"click_more_replies,omitempty"`
	// 回复数量阈值，超过这个数量的"更多"按钮将被跳过（0表示不跳过任何）
	MaxRepliesThreshold int `json:"max_replies_threshold,omitempty"`
	// 最大加载评论数（.parent-comment数量），0表示加载所有
	MaxCommentItems int `json:"max_comment_items,omitempty"`
	// 滚动速度等级: slow(慢速), normal(正常), fast(快速)
	ScrollSpeed string `json:"scroll_speed,omitempty"`
}

// FeedDetailRequest Feed详情请求
type FeedDetailRequest struct {
	FeedID          string             `json:"feed_id" binding:"required"`
	XsecToken       string             `json:"xsec_token" binding:"required"`
	LoadAllComments bool               `json:"load_all_comments,omitempty"`
	CommentConfig   *CommentLoadConfig `json:"comment_config,omitempty"`
}

type SearchFeedsRequest struct {
	Keyword string                   `json:"keyword" binding:"required"`
	Filters xiaohongshu.FilterOption `json:"filters,omitempty"`
}

// FeedDetailResponse Feed详情响应
type FeedDetailResponse struct {
	FeedID string `json:"feed_id"`
	Data   any    `json:"data"`
}

// PostCommentRequest 发表评论请求
type PostCommentRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

// PostCommentResponse 发表评论响应
type PostCommentResponse struct {
	FeedID  string `json:"feed_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ReplyCommentRequest 回复评论请求
type ReplyCommentRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	CommentID string `json:"comment_id" binding:"required_without=UserID"`
	UserID    string `json:"user_id" binding:"required_without=CommentID"`
	Content   string `json:"content" binding:"required"`
}

// ReplyCommentResponse 回复评论响应
type ReplyCommentResponse struct {
	FeedID          string `json:"feed_id"`
	TargetCommentID string `json:"target_comment_id,omitempty"`
	TargetUserID    string `json:"target_user_id,omitempty"`
	Success         bool   `json:"success"`
	Message         string `json:"message"`
}

// UserProfileRequest 用户主页请求
type UserProfileRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
}

// ActionResult 通用动作响应（点赞/收藏等）
type ActionResult struct {
	FeedID  string `json:"feed_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// LikeFeedRequest 点赞笔记请求
type LikeFeedRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	Unlike    bool   `json:"unlike,omitempty"` // true表示取消点赞
}

// FavoriteFeedRequest 收藏笔记请求
type FavoriteFeedRequest struct {
	FeedID     string `json:"feed_id" binding:"required"`
	XsecToken  string `json:"xsec_token" binding:"required"`
	Unfavorite bool   `json:"unfavorite,omitempty"` // true表示取消收藏
}

// LikeCommentRequest 点赞评论请求
type LikeCommentRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	CommentID string `json:"comment_id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Unlike    bool   `json:"unlike,omitempty"` // true表示取消点赞
}

// FollowUserRequest 关注用户请求
type FollowUserRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	Unfollow  bool   `json:"unfollow,omitempty"` // true表示取关
}

// DeleteFeedRequest 删除笔记请求
type DeleteFeedRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
}

// DeleteCommentRequest 删除评论请求
type DeleteCommentRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	CommentID string `json:"comment_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
}

// SaveDraftRequest 保存图文草稿请求
type SaveDraftRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Images  []string `json:"images" binding:"required,min=1"`
	Tags    []string `json:"tags,omitempty"`
}

// SaveVideoDraftRequest 保存视频草稿请求
type SaveVideoDraftRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Video   string   `json:"video" binding:"required"`
	Tags    []string `json:"tags,omitempty"`
}

// ShareFeedRequest 分享笔记请求
type ShareFeedRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
}

// ShareFeedResponse 分享笔记响应
type ShareFeedResponse struct {
	FeedID    string `json:"feed_id"`
	ShareLink string `json:"share_link"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

// GetMyFeedsRequest 获取自己笔记列表请求
type GetMyFeedsRequest struct {
	Limit int `json:"limit,omitempty"` // 默认20，最大100
}

// SyncCookiesRequest 上传cookies请求
type SyncCookiesRequest struct {
	CookiesBase64 string `json:"cookies_base64,omitempty"`
	CookiesJSON   string `json:"cookies_json,omitempty"`
}

// SyncCookiesResponse 上传cookies响应
type SyncCookiesResponse struct {
	Success    bool   `json:"success"`
	CookiePath string `json:"cookie_path"`
	FileSize   int64  `json:"file_size"`
	Message    string `json:"message"`
}

// GetFanAnalyticsRequest 获取粉丝分析请求
type GetFanAnalyticsRequest struct {
	Period string `json:"period,omitempty"` // 7d或30d，默认7d
}

// GetContentAnalyticsRequest 获取内容分析请求
type GetContentAnalyticsRequest struct {
	Limit     int    `json:"limit,omitempty"`      // 默认20，最大100
	SortBy    string `json:"sort_by,omitempty"`    // 排序字段
	SortOrder string `json:"sort_order,omitempty"` // asc或desc
}
