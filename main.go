package main

import (
	"flag"
	"os"

	"log/slog"

	"github.com/vmxmy/xiaohongshu-mcp/configs"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/interfaces/wiring"
)

// @title 小红书 MCP API
// @version 2.0.0
// @description 小红书 MCP 服务的 REST API 接口文档
// @description
// @description 提供完整的小红书内容发布、互动和数据分析功能
// @description
// @description ## 功能模块
// @description - **登录认证**: 登录状态检查、二维码登录、Cookie 管理
// @description - **内容发布**: 图文笔记、视频笔记发布
// @description - **内容发现**: Feed 列表、搜索、笔记详情
// @description - **用户信息**: 用户主页、个人资料
// @description - **内容互动**: 评论、点赞、收藏、关注
// @description
// @description ## MCP 协议
// @description 本服务同时提供 MCP (Model Context Protocol) 接口供 AI 工具使用
// @description - MCP 端点: /mcp
// @description - REST 端点: /api/v1/*

// @contact.name API Support
// @contact.url

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @BasePath /api/v1
// @schemes http https

// @tag.name 登录认证
// @tag.description 登录状态管理和认证相关接口

// @tag.name 内容发布
// @tag.description 发布图文和视频笔记

// @tag.name 内容发现
// @tag.description 浏览和搜索小红书内容

// @tag.name 用户信息
// @tag.description 获取用户资料和主页信息

// @tag.name 内容互动
// @tag.description 评论、点赞、收藏等互动操作

func main() {
	var (
		headless bool
		binPath  string // 浏览器二进制文件路径
		port     string
	)
	flag.BoolVar(&headless, "headless", true, "是否无头模式")
	flag.StringVar(&binPath, "bin", "", "浏览器二进制文件路径")
	flag.StringVar(&port, "port", ":18060", "端口")
	flag.Parse()

	if len(binPath) == 0 {
		binPath = os.Getenv("ROD_BROWSER_BIN")
	}

	configs.InitHeadless(headless)
	configs.SetBinPath(binPath)

	// 初始化服务
	publishUsecase := wiring.InitPublishUsecase(headless)

	// 加载轮询配置
	modules, err := loadPollingModules()
	if err != nil {
		slog.Error("加载轮询配置失败", "error", err)
		os.Exit(1)
	}

	// 构建 engine factory（使用全局 headless 配置）
	engineFactory := func() browser.Engine { return newBrowserEngine() }

	// 使用 wiring 包构建所有用例
	feedUsecase := wiring.BuildFeedUsecase(engineFactory, modules.Interaction)
	interactUsecase := wiring.BuildInteractionUsecase(engineFactory, modules.Interaction)
	userUsecase := wiring.BuildUserUsecase(engineFactory, modules.Interaction)
	analyticsUsecase := wiring.BuildAnalyticsUsecase(engineFactory, modules.Analytics)

	// 创建服务，注入所有用例
	xiaohongshuService, err := NewXiaohongshuServiceWithModules(
		publishUsecase,
		feedUsecase,
		interactUsecase,
		userUsecase,
		analyticsUsecase,
		modules,
	)
	if err != nil {
		slog.Error("初始化服务失败:", "arg1", err)
	}

	// 创建并启动应用服务器
	appServer := NewAppServer(xiaohongshuService)
	if err := appServer.Start(port); err != nil {
		slog.Error("failed to run server:", "arg1", err)
	}
}
