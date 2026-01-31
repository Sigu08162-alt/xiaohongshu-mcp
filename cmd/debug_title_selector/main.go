package main

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	browserplaywright "github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser/playwright"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.Info("=== 诊断标题输入框选择器 ===")

	// 配置浏览器
	engineCfg := browserplaywright.DefaultConfig()
	engineCfg.Headless = false // 使用有头模式,方便观察
	engineCfg.CookiePath = cookies.GetCookiesFilePath()
	engineCfg.NavigationTimeout = 5 * time.Minute
	engineCfg.ActionTimeout = 30 * time.Second

	engine := browserplaywright.New(engineCfg)

	// 启动浏览器
	if err := engine.Start(); err != nil {
		logrus.Fatalf("启动浏览器失败: %v", err)
	}
	defer engine.Close()

	// 创建页面
	page, err := engine.NewPage()
	if err != nil {
		logrus.Fatalf("创建页面失败: %v", err)
	}
	defer page.Close()

	// 访问发布页面
	publishURL := "https://creator.xiaohongshu.com/publish/publish?source=official&target=image"
	logrus.Infof("访问发布页面: %s", publishURL)
	if err := page.Goto(publishURL); err != nil {
		logrus.Fatalf("访问页面失败: %v", err)
	}

	// 等待页面稳定
	logrus.Info("等待页面稳定...")
	time.Sleep(3 * time.Second)

	// 先等待上传输入框出现（确保页面加载完成）
	logrus.Info("\n等待上传输入框出现（确保页面加载）...")
	if err := page.WaitVisible("input[type='file']"); err != nil {
		logrus.Warnf("上传输入框未出现: %v", err)
	} else {
		logrus.Info("✅ 上传输入框已出现")
	}

	// 再等一下，让所有元素渲染完成
	time.Sleep(2 * time.Second)

	// 扫描页面所有输入框
	logrus.Info("\n=== 扫描页面所有输入框 ===")
	scanAllInputs(page)

	// 检查 .d-text 选择器
	logrus.Info("\n=== 检查 .d-text 选择器 ===")
	checkSelector(page, ".d-text")

	// 检查更精确的选择器
	logrus.Info("\n=== 检查 input.d-text 选择器 ===")
	checkSelector(page, "input.d-text")

	// 检查通过 placeholder 定位的选择器
	logrus.Info("\n=== 检查通过 placeholder 定位 ===")
	checkSelector(page, "input[placeholder*='标题']")

	// 尝试其他可能的选择器
	logrus.Info("\n=== 尝试其他可能的选择器 ===")
	candidates := []string{
		"input[type='text']",
		"input.d-text[type='text']",
		"div.d-input input",
		".d-input input[type='text']",
	}
	for _, sel := range candidates {
		checkSelector(page, sel)
	}

	logrus.Info("\n=== 诊断完成,浏览器保持打开 ===")
	logrus.Info("请在浏览器中手动检查标题输入框的实际 HTML 结构")
	logrus.Info("按 Ctrl+C 退出")

	// 保持浏览器打开,方便手动检查
	select {}
}

func scanAllInputs(page browser.Page) {
	jsCode := `() => {
		const inputs = document.querySelectorAll('input, textarea, [contenteditable="true"]');
		return Array.from(inputs).map((el, idx) => ({
			index: idx,
			tagName: el.tagName.toLowerCase(),
			type: el.type || '',
			id: el.id || '',
			classes: Array.from(el.classList),
			placeholder: el.placeholder || el.getAttribute('data-placeholder') || '',
			name: el.name || '',
			value: el.value || '',
			contentEditable: el.contentEditable,
			visible: el.offsetParent !== null,
			rect: {
				x: el.getBoundingClientRect().x,
				y: el.getBoundingClientRect().y,
				width: el.getBoundingClientRect().width,
				height: el.getBoundingClientRect().height
			}
		}));
	}`

	result, err := page.Eval(jsCode)
	if err != nil {
		logrus.Errorf("扫描输入框失败: %v", err)
		return
	}

	// 格式化输出
	if data, err := json.MarshalIndent(result, "", "  "); err == nil {
		logrus.Infof("页面输入框列表:\n%s", string(data))
	} else {
		logrus.Infof("页面输入框: %+v", result)
	}
}

func checkSelector(page browser.Page, selector string) {
	// 检查元素是否存在
	has, err := page.Has(selector)
	if err != nil {
		logrus.Errorf("检查选择器失败: %v", err)
		return
	}

	if !has {
		logrus.Warnf("选择器 '%s' 未找到任何元素", selector)
		return
	}

	logrus.Infof("✓ 选择器 '%s' 找到元素", selector)

	// 获取元素详细信息（直接在 JS 中硬编码选择器）
	jsCodeWithSelector := `() => {
		const selector = "` + selector + `";
		const el = document.querySelector(selector);
		if (!el) return null;
		return {
			tagName: el.tagName.toLowerCase(),
			type: el.type || '',
			id: el.id || '',
			classes: Array.from(el.classList),
			placeholder: el.placeholder || el.getAttribute('data-placeholder') || '',
			name: el.name || '',
			contentEditable: el.contentEditable,
			visible: el.offsetParent !== null,
			outerHTML: el.outerHTML.substring(0, 300)
		};
	}`

	info, err := page.Eval(jsCodeWithSelector)
	if err != nil {
		logrus.Errorf("获取元素信息失败: %v", err)
	} else if info != nil {
		if data, err := json.MarshalIndent(info, "  ", "  "); err == nil {
			logrus.Infof("  元素详情:\n  %s", string(data))
		}
	}

	// 尝试获取元素并填充
	elem, err := page.Element(selector)
	if err != nil {
		logrus.Errorf("获取元素失败: %v", err)
		return
	}

	// 尝试填充
	testText := "测试标题"
	if err := elem.Fill(testText); err != nil {
		logrus.Errorf("  ✗ Fill 方法失败: %v", err)

		// 尝试替代方案: 点击后输入
		if err := elem.Click(); err != nil {
			logrus.Errorf("  ✗ Click 失败: %v", err)
		} else {
			time.Sleep(200 * time.Millisecond)
			if err := elem.Input(testText); err != nil {
				logrus.Errorf("  ✗ Input 失败: %v", err)
			} else {
				logrus.Info("  ✓ Click + Input 成功")
				// 清空输入
				elem.Fill("")
			}
		}
	} else {
		logrus.Info("  ✓ Fill 方法成功")
		// 清空输入
		elem.Fill("")
	}
}
