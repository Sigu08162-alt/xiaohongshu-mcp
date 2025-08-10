package main

import (
	"context"
	"errors"

	"github.com/xpzouying/xiaohongshu-mcp/browser"
	"github.com/xpzouying/xiaohongshu-mcp/configs"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
)

// XiaohongshuService 小红书业务服务
type XiaohongshuService struct{}

// NewXiaohongshuService 创建小红书服务实例
func NewXiaohongshuService() *XiaohongshuService {
	return &XiaohongshuService{}
}

// PublishRequest 发布请求
type PublishRequest struct {
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	ImagePaths []string `json:"image_paths" binding:"required,min=1"`
}

// LoginStatusResponse 登录状态响应
type LoginStatusResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Username   string `json:"username,omitempty"`
}

// PublishResponse 发布响应
type PublishResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Images  int    `json:"images"`
	Status  string `json:"status"`
	PostID  string `json:"post_id,omitempty"`
}

// CheckLoginStatus 检查登录状态
func (s *XiaohongshuService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	// 使用全局单例浏览器创建新页面
	page := browser.NewPage()
	defer page.Close()
	loginAction := xiaohongshu.NewLogin(page)

	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, err
	}

	response := &LoginStatusResponse{
		IsLoggedIn: isLoggedIn,
		Username:   configs.Username,
	}

	return response, nil
}

// PublishContent 发布内容
func (s *XiaohongshuService) PublishContent(ctx context.Context, req *PublishRequest) (*PublishResponse, error) {
	// 验证参数
	if req.Title == "" {
		return nil, errors.New("标题不能为空")
	}
	if req.Content == "" {
		return nil, errors.New("内容不能为空")
	}
	if len(req.ImagePaths) == 0 {
		return nil, errors.New("至少需要一个图片ID")
	}

	// 构建发布内容
	content := xiaohongshu.PublishImageContent{
		Title:      req.Title,
		Content:    req.Content,
		ImagePaths: req.ImagePaths,
	}

	// 使用全局单例浏览器创建新页面
	page := browser.NewPage()
	defer page.Close()
	action, err := xiaohongshu.NewPublishImageAction(page)
	if err != nil {
		return nil, err
	}

	// 执行发布
	if err := action.Publish(ctx, content); err != nil {
		return nil, err
	}

	response := &PublishResponse{
		Title:   req.Title,
		Content: req.Content,
		Images:  len(req.ImagePaths),
		Status:  "发布完成",
	}

	return response, nil
}
