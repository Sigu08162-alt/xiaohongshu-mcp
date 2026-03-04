package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	appanalytics "github.com/vmxmy/xiaohongshu-mcp/internal/app/analytics"
	appfeed "github.com/vmxmy/xiaohongshu-mcp/internal/app/feed"
	appinteraction "github.com/vmxmy/xiaohongshu-mcp/internal/app/interaction"
	apppublish "github.com/vmxmy/xiaohongshu-mcp/internal/app/publish"
	appuser "github.com/vmxmy/xiaohongshu-mcp/internal/app/user"
	domainpublish "github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/ratelimit"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
	"log/slog"
)

type loginProvider interface {
	GetQRCode(ctx context.Context) (loginQRResult, error)
}

type PollingModules struct {
	Publish     polling.Module
	Draft       polling.Module
	Video       polling.Module
	Interaction polling.Module
	Analytics   polling.Module
	Auth        polling.Module
}

// XiaohongshuService 小红书业务服务
type XiaohongshuService struct {
	publishUsecase   *apppublish.Usecase
	feedUsecase      *appfeed.Usecase
	interactUsecase  *appinteraction.Usecase
	userUsecase      *appuser.Usecase
	analyticsUsecase *appanalytics.Usecase
	loginManager     loginProvider
	loginSessionNew  func() (loginSession, error)
	polling          PollingModules
	limiter          *ratelimit.Limiter
}

// NewXiaohongshuServiceWithModules 使用已加载的轮询配置创建服务
func NewXiaohongshuServiceWithModules(
	publishUsecase *apppublish.Usecase,
	feedUsecase *appfeed.Usecase,
	interactUsecase *appinteraction.Usecase,
	userUsecase *appuser.Usecase,
	analyticsUsecase *appanalytics.Usecase,
	modules PollingModules,
) (*XiaohongshuService, error) {
	loginTTL := time.Duration(modules.Auth.TimeoutMs) * time.Millisecond
	if loginTTL <= 0 {
		return nil, fmt.Errorf("polling.auth.timeout_ms missing or invalid")
	}
	return &XiaohongshuService{
		publishUsecase:   publishUsecase,
		feedUsecase:      feedUsecase,
		interactUsecase:  interactUsecase,
		userUsecase:      userUsecase,
		analyticsUsecase: analyticsUsecase,
		loginManager:     NewLoginManager(newPlaywrightLoginSession, loginTTL),
		loginSessionNew:  newPlaywrightLoginSession,
		polling:          modules,
		limiter:          ratelimit.DefaultLimiter(),
	}, nil
}

// PublishRequest 发布请求
type PublishRequest struct {
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	Images     []string `json:"images" binding:"required,min=1"`
	Tags       []string `json:"tags,omitempty"`
	Location   string   `json:"location,omitempty"`
	MarkerTags []string `json:"marker_tags,omitempty"`
	ScheduleAt string   `json:"schedule_at,omitempty"` // 定时发布时间，ISO8601格式，为空则立即发布
}

// LoginStatusResponse 登录状态响应
type LoginStatusResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Username   string `json:"username,omitempty"`
}

// LoginQrcodeResponse 登录扫码二维码
type LoginQrcodeResponse struct {
	Timeout    string `json:"timeout"`
	IsLoggedIn bool   `json:"is_logged_in"`
	Img        string `json:"img,omitempty"`
	Stage      string `json:"stage,omitempty"`
	Status     string `json:"status,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
}

// PublishResponse 发布响应
type PublishResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Images  int    `json:"images"`
	Status  string `json:"status"`
	PostID  string `json:"post_id,omitempty"`
}

// PublishVideoRequest 发布视频请求（仅支持本地单个视频文件）
type PublishVideoRequest struct {
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	Video      string   `json:"video" binding:"required"`
	Tags       []string `json:"tags,omitempty"`
	ScheduleAt string   `json:"schedule_at,omitempty"` // 定时发布时间，ISO8601格式，为空则立即发布
}

// PublishVideoResponse 发布视频响应
type PublishVideoResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Video   string `json:"video"`
	Status  string `json:"status"`
	PostID  string `json:"post_id,omitempty"`
}

// FeedsListResponse Feeds列表响应
type FeedsListResponse struct {
	Feeds []xiaohongshu.Feed `json:"feeds"`
	Count int                `json:"count"`
}

// UserProfileResponse 用户主页响应
type UserProfileResponse struct {
	UserBasicInfo xiaohongshu.UserBasicInfo      `json:"userBasicInfo"`
	Interactions  []xiaohongshu.UserInteractions `json:"interactions"`
	Feeds         []xiaohongshu.Feed             `json:"feeds"`
}

// DeleteCookies 删除 cookies 文件，用于登录重置
func (s *XiaohongshuService) DeleteCookies(ctx context.Context) error {
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)
	return cookieLoader.DeleteCookies()
}

// SyncCookies 写入 cookies 文件，供无头模式加载。
func (s *XiaohongshuService) SyncCookies(ctx context.Context, data []byte) (string, int64, error) {
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)
	if err := cookieLoader.SaveCookies(data); err != nil {
		return "", 0, err
	}
	return cookiePath, int64(len(data)), nil
}

// CheckLoginStatus 检查登录状态
func (s *XiaohongshuService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	newSession := s.loginSessionNew
	if newSession == nil {
		newSession = newPlaywrightLoginSession
	}

	session, err := newSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	if err := session.Open(ctx); err != nil {
		return nil, err
	}

	loggedIn, err := session.LoggedIn(ctx)
	if err != nil {
		return nil, err
	}

	username := ""
	if loggedIn {
		provider, ok := session.(interface {
			Nickname(context.Context) (string, error)
		})
		if !ok {
			slog.Warn("login session does not support nickname extraction, treat as logged out")
			loggedIn = false
		} else {
			name, nameErr := provider.Nickname(ctx)
			if nameErr != nil {
				slog.Warn("extract login nickname failed, treat as logged out", "error", nameErr)
				loggedIn = false
			} else {
				username = strings.TrimSpace(name)
				if isPlaceholderStatusUsername(username) {
					slog.Warn("extract login nickname unavailable, treat as logged out", "nickname", username)
					loggedIn = false
					username = ""
				}
			}
		}
	}

	return &LoginStatusResponse{
		IsLoggedIn: loggedIn,
		Username:   username,
	}, nil
}

func isPlaceholderStatusUsername(name string) bool {
	n := strings.TrimSpace(strings.ToLower(name))
	if strings.Contains(n, "小红书") && strings.Contains(n, "创作服务平台") {
		return true
	}
	switch n {
	case "", "我", "我的", "me", "my", "小红书", "xiaohongshu", "xiaohongshu creator", "xhs creator":
		return true
	default:
		return false
	}
}

// GetLoginQrcode 获取登录的扫码二维码
func (s *XiaohongshuService) GetLoginQrcode(ctx context.Context) (*LoginQrcodeResponse, error) {
	if s.loginManager == nil {
		return s.getLoginQrcodeLegacy(ctx)
	}

	result, err := s.loginManager.GetQRCode(ctx)
	if err != nil {
		return nil, err
	}

	img := result.Img
	if img != "" && !strings.HasPrefix(img, "data:image/png;base64,") {
		img = "data:image/png;base64," + img
	}

	return &LoginQrcodeResponse{
		Timeout:    result.Timeout,
		Img:        img,
		IsLoggedIn: result.IsLoggedIn,
		Stage:      result.Stage,
		Status:     result.Status,
		SessionID:  result.SessionID,
	}, nil
}

func (s *XiaohongshuService) getLoginQrcodeLegacy(ctx context.Context) (*LoginQrcodeResponse, error) {
	var img string
	var loggedIn bool
	var err error

	// 注意: 这个函数需要保持页面打开来等待登录，但 withBrowserPage 会在函数返回后关闭页面
	// 这里保持原有的逻辑，但使用新的浏览器引擎
	engine := newBrowserEngine()
	if err := engine.Start(); err != nil {
		return nil, err
	}

	page, err := engine.NewPage()
	if err != nil {
		engine.Close()
		return nil, err
	}

	deferFunc := func() {
		_ = page.Close()
		engine.Close()
	}

	loginAction := xiaohongshu.NewLogin(page, s.polling.Auth)

	img, loggedIn, err = loginAction.FetchQrcodeImage(ctx)
	if err != nil || loggedIn {
		defer deferFunc()
	}
	if err != nil {
		return nil, err
	}

	timeout := 4 * time.Minute

	if !loggedIn {
		go func() {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			defer deferFunc()

			// Playwright 引擎会自动保存 cookies，无需手动调用 saveCookies
			_ = loginAction.WaitForLogin(ctxTimeout)
		}()
	}

	return &LoginQrcodeResponse{
		Timeout: func() string {
			if loggedIn {
				return "0s"
			}
			return timeout.String()
		}(),
		Img:        img,
		IsLoggedIn: loggedIn,
	}, nil
}

// PublishContent 发布内容
func (s *XiaohongshuService) PublishContent(ctx context.Context, req *PublishRequest) (*PublishResponse, error) {
	if err := s.limiter.Wait(ctx, "publish"); err != nil {
		return nil, err
	}
	// 验证标题长度
	// 小红书限制：最大40个单位长度
	// 中文/日文/韩文占2个单位，英文/数字占1个单位
	if titleWidth := runewidth.StringWidth(req.Title); titleWidth > 40 {
		return nil, fmt.Errorf("标题长度超过限制")
	}

	// 解析定时发布时间
	var scheduleTime *time.Time
	if req.ScheduleAt != "" {
		t, err := time.Parse(time.RFC3339, req.ScheduleAt)
		if err != nil {
			return nil, fmt.Errorf("定时发布时间格式错误，请使用 ISO8601 格式: %v", err)
		}

		// 校验定时发布时间范围：1小时至14天
		now := time.Now()
		minTime := now.Add(1 * time.Hour)
		maxTime := now.Add(14 * 24 * time.Hour)

		if t.Before(minTime) {
			return nil, fmt.Errorf("定时发布时间必须至少在1小时后，当前设置: %s，最早可选: %s",
				t.Format("2006-01-02 15:04"), minTime.Format("2006-01-02 15:04"))
		}
		if t.After(maxTime) {
			return nil, fmt.Errorf("定时发布时间不能超过14天，当前设置: %s，最晚可选: %s",
				t.Format("2006-01-02 15:04"), maxTime.Format("2006-01-02 15:04"))
		}

		scheduleTime = &t
		slog.Info("设置定时发布时间:", "arg1", t.Format("2006-01-02 15:04"))
	}

	markerTags := domainpublish.FilterMarkerTags(req.MarkerTags)

	// 构建发布内容（使用原始图片路径，Usecase层会处理）
	content := xiaohongshu.PublishImageContent{
		Title:        req.Title,
		Content:      req.Content,
		Tags:         req.Tags,
		Location:     req.Location,
		MarkerTags:   markerTags,
		ImagePaths:   req.Images, // 使用原始路径，不再预处理
		ScheduleTime: scheduleTime,
	}

	// 执行发布（优先使用新用例）
	if s.publishUsecase != nil {
		if err := s.publishUsecase.PublishImage(ctx, domainpublish.ImageContent{
			Title:        content.Title,
			Content:      content.Content,
			Tags:         content.Tags,
			ImagePaths:   content.ImagePaths, // Usecase会调用ImageProcessor处理
			Location:     content.Location,
			MarkerTags:   markerTags,
			ScheduleTime: content.ScheduleTime,
		}); err != nil {
			slog.Error("发布内容失败(新用例): title=", "arg1", content.Title, "arg2", err)
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("发布用例未初始化，无法执行发布")
	}

	response := &PublishResponse{
		Title:   req.Title,
		Content: req.Content,
		Images:  len(req.Images), // 使用原始图片数量
		Status:  "发布完成",
	}

	return response, nil
}

// publishContent 执行内容发布
func (s *XiaohongshuService) publishContent(ctx context.Context, content xiaohongshu.PublishImageContent) error {
	return withBrowserPage(func(page browser.Page) error {
		action, err := xiaohongshu.NewPublishImageAction(ctx, page)
		if err != nil {
			return err
		}

		// 执行发布
		return action.Publish(ctx, content)
	})
}

// PublishVideo 发布视频（本地文件）
func (s *XiaohongshuService) PublishVideo(ctx context.Context, req *PublishVideoRequest) (*PublishVideoResponse, error) {
	if err := s.limiter.Wait(ctx, "publish"); err != nil {
		return nil, err
	}
	// 标题长度校验
	if titleWidth := runewidth.StringWidth(req.Title); titleWidth > 40 {
		return nil, fmt.Errorf("标题长度超过限制")
	}

	// 本地视频文件校验
	if req.Video == "" {
		return nil, fmt.Errorf("必须提供本地视频文件")
	}
	if _, err := os.Stat(req.Video); err != nil {
		return nil, fmt.Errorf("视频文件不存在或不可访问: %v", err)
	}

	// 解析定时发布时间
	var scheduleTime *time.Time
	if req.ScheduleAt != "" {
		t, err := time.Parse(time.RFC3339, req.ScheduleAt)
		if err != nil {
			return nil, fmt.Errorf("定时发布时间格式错误，请使用 ISO8601 格式: %v", err)
		}

		// 校验定时发布时间范围：1小时至14天
		now := time.Now()
		minTime := now.Add(1 * time.Hour)
		maxTime := now.Add(14 * 24 * time.Hour)

		if t.Before(minTime) {
			return nil, fmt.Errorf("定时发布时间必须至少在1小时后，当前设置: %s，最早可选: %s",
				t.Format("2006-01-02 15:04"), minTime.Format("2006-01-02 15:04"))
		}
		if t.After(maxTime) {
			return nil, fmt.Errorf("定时发布时间不能超过14天，当前设置: %s，最晚可选: %s",
				t.Format("2006-01-02 15:04"), maxTime.Format("2006-01-02 15:04"))
		}

		scheduleTime = &t
		slog.Info("设置定时发布时间:", "arg1", t.Format("2006-01-02 15:04"))
	}

	// 构建发布内容
	content := xiaohongshu.PublishVideoContent{
		Title:        req.Title,
		Content:      req.Content,
		Tags:         req.Tags,
		VideoPath:    req.Video,
		ScheduleTime: scheduleTime,
	}

	// 执行发布
	if err := s.publishVideo(ctx, content); err != nil {
		return nil, err
	}

	resp := &PublishVideoResponse{
		Title:   req.Title,
		Content: req.Content,
		Video:   req.Video,
		Status:  "发布完成",
	}
	return resp, nil
}

// SaveDraft 保存图文草稿（暂存离开，不立即发布）
func (s *XiaohongshuService) SaveDraft(ctx context.Context, req *SaveDraftRequest) (*ActionResult, error) {
	if s.publishUsecase == nil {
		return nil, fmt.Errorf("发布服务未初始化")
	}

	content := domainpublish.ImageContent{
		Title:      req.Title,
		Content:    req.Content,
		Tags:       req.Tags,
		ImagePaths: req.Images,
	}
	if err := s.publishUsecase.SaveImageDraft(ctx, content); err != nil {
		return nil, err
	}

	return &ActionResult{
		Success: true,
		Message: "草稿保存成功",
	}, nil
}

// SaveVideoDraft 保存视频草稿（暂存离开，不立即发布）
func (s *XiaohongshuService) SaveVideoDraft(ctx context.Context, req *SaveVideoDraftRequest) (*ActionResult, error) {
	if s.publishUsecase == nil {
		return nil, fmt.Errorf("发布服务未初始化")
	}
	if strings.TrimSpace(req.Video) == "" {
		return nil, fmt.Errorf("缺少本地视频文件路径")
	}

	content := domainpublish.VideoContent{
		Title:     req.Title,
		Content:   req.Content,
		Tags:      req.Tags,
		VideoPath: req.Video,
	}
	if err := s.publishUsecase.SaveVideoDraft(ctx, content); err != nil {
		return nil, err
	}

	return &ActionResult{
		Success: true,
		Message: "视频草稿保存成功",
	}, nil
}

// publishVideo 执行视频发布
func (s *XiaohongshuService) publishVideo(ctx context.Context, content xiaohongshu.PublishVideoContent) error {
	return withBrowserPage(func(page browser.Page) error {
		action, err := xiaohongshu.NewPublishVideoAction(ctx, page, s.polling.Video)
		if err != nil {
			return err
		}

		return action.PublishVideo(ctx, content)
	})
}

// ListFeeds 获取Feeds列表
func (s *XiaohongshuService) ListFeeds(ctx context.Context) (*FeedsListResponse, error) {
	feeds, err := s.feedUsecase.ListFeeds(ctx)
	if err != nil {
		slog.Error("获取 Feeds 列表失败:", "arg1", err)
		return nil, err
	}

	return &FeedsListResponse{Feeds: feeds, Count: len(feeds)}, nil
}

func (s *XiaohongshuService) SearchFeeds(ctx context.Context, keyword string, filters ...xiaohongshu.FilterOption) (*FeedsListResponse, error) {
	if err := s.limiter.Wait(ctx, "search"); err != nil {
		return nil, err
	}
	var filter xiaohongshu.FilterOption
	if len(filters) > 0 {
		filter = filters[0]
	}
	feeds, err := s.feedUsecase.SearchFeeds(ctx, keyword, filter)
	if err != nil {
		return nil, err
	}

	response := &FeedsListResponse{
		Feeds: feeds,
		Count: len(feeds),
	}

	return response, nil
}

// GetFeedDetail 获取Feed详情
func (s *XiaohongshuService) GetFeedDetail(ctx context.Context, feedID, xsecToken string, loadAllComments bool) (*FeedDetailResponse, error) {
	return s.GetFeedDetailWithConfig(ctx, feedID, xsecToken, loadAllComments, xiaohongshu.DefaultCommentLoadConfig())
}

// GetFeedDetailWithConfig 使用配置获取Feed详情
func (s *XiaohongshuService) GetFeedDetailWithConfig(ctx context.Context, feedID, xsecToken string, loadAllComments bool, config xiaohongshu.CommentLoadConfig) (*FeedDetailResponse, error) {
	result, err := s.feedUsecase.GetFeedDetail(ctx, feedID, xsecToken)
	if err != nil {
		return nil, err
	}

	return &FeedDetailResponse{FeedID: feedID, Data: result}, nil
}

// UserProfile 获取用户信息
func (s *XiaohongshuService) UserProfile(ctx context.Context, userID, xsecToken string) (*UserProfileResponse, error) {
	result, err := s.userUsecase.GetUserProfile(ctx, userID, xsecToken)
	if err != nil {
		return nil, err
	}

	response := &UserProfileResponse{
		UserBasicInfo: result.UserBasicInfo,
		Interactions:  result.Interactions,
		Feeds:         result.Feeds,
	}

	return response, nil
}

// PostCommentToFeed 发表评论到Feed
func (s *XiaohongshuService) PostCommentToFeed(ctx context.Context, feedID, xsecToken, content string) (*PostCommentResponse, error) {
	if err := s.limiter.Wait(ctx, "comment"); err != nil {
		return nil, err
	}
	if err := s.interactUsecase.PostComment(ctx, feedID, xsecToken, content); err != nil {
		return nil, err
	}
	return &PostCommentResponse{FeedID: feedID, Success: true, Message: "评论发表成功"}, nil
}

// LikeFeed 点赞笔记
func (s *XiaohongshuService) LikeFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.limiter.Wait(ctx, "like"); err != nil {
		return nil, err
	}
	if err := s.interactUsecase.LikeFeed(ctx, feedID, xsecToken); err != nil {
		return nil, err
	}
	return &ActionResult{FeedID: feedID, Success: true, Message: "点赞成功或已点赞"}, nil
}

// UnlikeFeed 取消点赞笔记
func (s *XiaohongshuService) UnlikeFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.limiter.Wait(ctx, "like"); err != nil {
		return nil, err
	}
	if err := s.interactUsecase.UnlikeFeed(ctx, feedID, xsecToken); err != nil {
		return nil, err
	}

	return &ActionResult{FeedID: feedID, Success: true, Message: "取消点赞成功或未点赞"}, nil
}

// FavoriteFeed 收藏笔记
func (s *XiaohongshuService) FavoriteFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.interactUsecase.FavoriteFeed(ctx, feedID, xsecToken); err != nil {
		return nil, err
	}
	return &ActionResult{FeedID: feedID, Success: true, Message: "收藏成功或已收藏"}, nil
}

// UnfavoriteFeed 取消收藏笔记
func (s *XiaohongshuService) UnfavoriteFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.interactUsecase.UnfavoriteFeed(ctx, feedID, xsecToken); err != nil {
		return nil, err
	}

	return &ActionResult{FeedID: feedID, Success: true, Message: "取消收藏成功或未收藏"}, nil
}

// ReplyCommentToFeed 回复指定评论
func (s *XiaohongshuService) ReplyCommentToFeed(ctx context.Context, feedID, xsecToken, commentID, userID, content string) (*ReplyCommentResponse, error) {
	if err := s.limiter.Wait(ctx, "comment"); err != nil {
		return nil, err
	}
	if err := s.interactUsecase.ReplyComment(ctx, feedID, xsecToken, commentID, userID, content); err != nil {
		return nil, err
	}

	return &ReplyCommentResponse{
		FeedID:          feedID,
		TargetCommentID: commentID,
		TargetUserID:    userID,
		Success:         true,
		Message:         "评论回复成功",
	}, nil
}

// 注意：newBrowserEngine, withBrowserPage 已移至 browser_factory.go

// GetMyProfile 获取当前登录用户的个人信息
func (s *XiaohongshuService) GetMyProfile(ctx context.Context) (*UserProfileResponse, error) {
	result, err := s.userUsecase.GetMyProfile(ctx)
	if err != nil {
		return nil, err
	}

	response := &UserProfileResponse{
		UserBasicInfo: result.UserBasicInfo,
		Interactions:  result.Interactions,
		Feeds:         result.Feeds,
	}

	return response, nil
}

// FollowUser 关注用户
func (s *XiaohongshuService) FollowUser(ctx context.Context, userID, xsecToken string) (*ActionResult, error) {
	if err := s.limiter.Wait(ctx, "follow"); err != nil {
		return nil, err
	}
	if err := s.userUsecase.FollowUser(ctx, userID, xsecToken); err != nil {
		return nil, err
	}
	return &ActionResult{FeedID: userID, Success: true, Message: "关注成功"}, nil
}

// UnfollowUser 取关用户
func (s *XiaohongshuService) UnfollowUser(ctx context.Context, userID, xsecToken string) (*ActionResult, error) {
	if err := s.userUsecase.UnfollowUser(ctx, userID, xsecToken); err != nil {
		return nil, err
	}

	return &ActionResult{
		FeedID:  userID,
		Success: true,
		Message: "取关成功",
	}, nil
}

// LikeComment 点赞评论
func (s *XiaohongshuService) LikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string) (*ActionResult, error) {
	if err := s.interactUsecase.LikeComment(ctx, feedID, xsecToken, commentID, userID); err != nil {
		return nil, err
	}

	return &ActionResult{
		FeedID:  feedID,
		Success: true,
		Message: "评论点赞成功",
	}, nil
}

// UnlikeComment 取消点赞评论
func (s *XiaohongshuService) UnlikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string) (*ActionResult, error) {
	if err := s.interactUsecase.UnlikeComment(ctx, feedID, xsecToken, commentID, userID); err != nil {
		return nil, err
	}

	return &ActionResult{
		FeedID:  feedID,
		Success: true,
		Message: "取消评论点赞成功",
	}, nil
}

// ShareFeed 分享笔记，获取分享链接
func (s *XiaohongshuService) ShareFeed(ctx context.Context, feedID, xsecToken string) (string, error) {
	return s.feedUsecase.ShareFeed(ctx, feedID, xsecToken)
}

// DeleteFeed 删除自己的笔记
func (s *XiaohongshuService) DeleteFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.feedUsecase.DeleteFeed(ctx, feedID, xsecToken); err != nil {
		return nil, err
	}

	return &ActionResult{FeedID: feedID, Success: true, Message: "笔记删除成功"}, nil
}

// DeleteComment 删除自己的评论
func (s *XiaohongshuService) DeleteComment(ctx context.Context, feedID, xsecToken, commentID, userID string) (*ActionResult, error) {
	if err := s.interactUsecase.DeleteComment(ctx, feedID, xsecToken, commentID, userID); err != nil {
		return nil, err
	}

	return &ActionResult{
		FeedID:  feedID,
		Success: true,
		Message: "评论删除成功",
	}, nil
}

// GetMyStats 获取当前用户的统计数据
func (s *XiaohongshuService) GetMyStats(ctx context.Context) (*xiaohongshu.UserStats, error) {
	raw, err := s.userUsecase.GetMyStats(ctx)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	stats := &xiaohongshu.UserStats{}
	if v, ok := raw["follower_count"]; ok {
		if n, ok := v.(int); ok {
			stats.FollowerCount = n
		}
	}
	if v, ok := raw["follow_count"]; ok {
		if n, ok := v.(int); ok {
			stats.FollowCount = n
		}
	}
	if v, ok := raw["note_count"]; ok {
		if n, ok := v.(int); ok {
			stats.NoteCount = n
		}
	}
	if v, ok := raw["liked_count"]; ok {
		if n, ok := v.(int); ok {
			stats.LikedCount = n
		}
	}
	if v, ok := raw["collect_count"]; ok {
		if n, ok := v.(int); ok {
			stats.CollectCount = n
		}
	}
	return stats, nil
}

// GetMyFeeds 获取自己发布的笔记列表
func (s *XiaohongshuService) GetMyFeeds(ctx context.Context, limit int, userID string) ([]xiaohongshu.Feed, error) {
	return s.feedUsecase.GetMyFeeds(ctx, userID, limit)
}

// GetFanAnalytics 获取粉丝分析数据
func (s *XiaohongshuService) GetFanAnalytics(ctx context.Context, period string) (*xiaohongshu.FanAnalytics, error) {
	return s.analyticsUsecase.GetFanAnalytics(ctx, period)
}

// GetContentAnalytics 获取内容分析数据
func (s *XiaohongshuService) GetContentAnalytics(ctx context.Context, limit int, sortBy xiaohongshu.SortField, sortOrder xiaohongshu.SortOrder) (*xiaohongshu.ContentAnalytics, error) {
	return s.analyticsUsecase.GetContentAnalytics(ctx, limit, string(sortBy), string(sortOrder))
}
