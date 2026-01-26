package main

import (
	"flag"
	"os"

	_ "github.com/xpzouying/xiaohongshu-mcp/docs" // Swagger docs

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/config"
	"github.com/xpzouying/xiaohongshu-mcp/configs"
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
// @contact.url https://github.com/xpzouying/xiaohongshu-mcp

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
		headless   bool
		binPath    string // 浏览器二进制文件路径
		port       string
		configPath string // 配置文件路径
	)
	flag.BoolVar(&headless, "headless", true, "是否无头模式")
	flag.StringVar(&binPath, "bin", "", "浏览器二进制文件路径")
	flag.StringVar(&port, "port", ":18060", "端口")
	flag.StringVar(&configPath, "config", "", "配置文件路径（默认自动查找 config.yaml）")
	flag.Parse()

	// 加载配置文件
	var err error
	if configPath != "" {
		_, err = config.Load(configPath)
	} else {
		_, err = config.LoadDefault()
	}
	if err != nil {
		logrus.Warnf("加载配置文件失败（将使用默认值）: %v", err)
		// 不退出，继续使用代码中的默认值
	} else {
		logrus.Infof("配置文件加载成功")
	}

	if len(binPath) == 0 {
		binPath = os.Getenv("ROD_BROWSER_BIN")
	}

	configs.InitHeadless(headless)
	configs.SetBinPath(binPath)

	// 初始化服务
	publishUsecase := initPublishUsecase(headless)
	xiaohongshuService := NewXiaohongshuServiceWithUsecase(publishUsecase)

	// 创建并启动应用服务器
	appServer := NewAppServerWithPublish(xiaohongshuService, publishUsecase)
	if err := appServer.Start(port); err != nil {
		logrus.Fatalf("failed to run server: %v", err)
	}
}
