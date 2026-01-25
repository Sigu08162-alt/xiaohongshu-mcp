package publish

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

func setLocation(page browser.Page, location string) error {
	logrus.Infof("📍 开始设置地点: %s", location)

	// 1. 查找地点输入框
	logrus.Info("  [1/4] 查找地点输入框 (选择器: .address-box input.d-text)...")
	locationInput, err := page.Element(".address-box input.d-text")
	if err != nil {
		logrus.Errorf("  ❌ 查找地点输入框失败: %v", err)
		// 尝试截图调试
		screenshotPath := fmt.Sprintf("debug_location_input_not_found_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		logrus.Infof("  📸 已保存截图: %s", screenshotPath)
		return fmt.Errorf("查找地点输入框失败: %w", err)
	}
	logrus.Info("  ✅ 找到地点输入框")

	// 2. 点击输入框
	logrus.Info("  [2/4] 点击地点输入框...")

	// 先检查元素是否可见
	visible, err := locationInput.IsVisible()
	if err != nil {
		logrus.Warnf("  ⚠️  无法检查输入框可见性: %v", err)
	} else {
		logrus.Infof("  📊 输入框可见性: %v", visible)
	}

	// 尝试滚动到元素
	logrus.Info("  📜 尝试滚动到元素位置...")
	if err := locationInput.ScrollIntoView(); err != nil {
		logrus.Warnf("  ⚠️  滚动失败: %v (继续)", err)
	} else {
		logrus.Info("  ✅ 滚动完成")
		time.Sleep(300 * time.Millisecond)
	}

	// 直接使用强制点击，避免等待可点击状态超时
	logrus.Info("  🖱️  强制点击输入框...")
	if err := locationInput.ClickForce(); err != nil {
		logrus.Errorf("  ❌ 点击失败: %v", err)
		// 截图调试
		screenshotPath := fmt.Sprintf("debug_location_click_failed_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		logrus.Infof("  📸 已保存截图: %s", screenshotPath)
		return fmt.Errorf("点击地点输入框失败: %w", err)
	}
	logrus.Info("  ✅ 点击成功")
	time.Sleep(300 * time.Millisecond)

	// 3. 输入地点关键词
	logrus.Infof("  [3/4] 输入地点关键词: '%s'...", location)
	if err := locationInput.Fill(location); err != nil {
		logrus.Errorf("  ❌ 输入地点关键词失败: %v", err)
		return fmt.Errorf("输入地点关键词失败: %w", err)
	}
	logrus.Info("  ✅ 关键词已输入")

	// 等待下拉列表出现
	logrus.Info("  ⏱️  等待下拉列表出现...")
	time.Sleep(1500 * time.Millisecond)

	// 4. 查找并点击下拉选项
	logrus.Info("  [4/4] 查找地点下拉列表...")
	dropdown, err := findVisibleLocationDropdown(page, location)
	if err != nil {
		logrus.Errorf("  ❌ 查找地点下拉列表失败: %v", err)
		// 截图调试
		screenshotPath := fmt.Sprintf("debug_location_dropdown_not_found_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		logrus.Infof("  📸 已保存截图: %s", screenshotPath)
		return fmt.Errorf("查找地点下拉列表失败: %w", err)
	}
	logrus.Info("  ✅ 找到下拉列表")

	firstItem, err := dropdown.Element(".item")
	if err != nil {
		logrus.Errorf("  ❌ 查找地点选项失败: %v", err)
		return fmt.Errorf("查找地点选项失败: %w", err)
	}
	logrus.Info("  ✅ 找到第一个地点选项")

	if err := firstItem.Click(); err != nil {
		logrus.Errorf("  ❌ 点击地点选项失败: %v", err)
		return fmt.Errorf("点击地点选项失败: %w", err)
	}
	logrus.Info("  ✅ 地点选项已点击")

	time.Sleep(500 * time.Millisecond)
	logrus.Infof("✅ 地点设置完成: %s", location)
	return nil
}

func findVisibleLocationDropdown(page browser.Page, keyword string) (browser.Element, error) {
	logrus.Infof("  🔍 搜索包含关键词 '%s' 的下拉列表...", keyword)

	// 分解关键词（如 "香港 铜锣湾" -> ["香港", "铜锣湾"]）
	keywords := strings.Fields(keyword)
	logrus.Debugf("  📝 分解后的关键词: %v", keywords)

	dropdowns, err := page.Elements(".d-dropdown-wrapper")
	if err != nil {
		logrus.Errorf("  ❌ 查找 .d-dropdown-wrapper 元素失败: %v", err)
		return nil, err
	}
	logrus.Infof("  📋 找到 %d 个 .d-dropdown-wrapper 元素", len(dropdowns))

	visibleCount := 0
	for i, dropdown := range dropdowns {
		visible, err := dropdown.IsVisible()
		if err != nil {
			logrus.Debugf("  [%d/%d] 检查可见性失败: %v", i+1, len(dropdowns), err)
			continue
		}

		if !visible {
			logrus.Debugf("  [%d/%d] 不可见，跳过", i+1, len(dropdowns))
			continue
		}

		visibleCount++
		logrus.Debugf("  [%d/%d] 可见，检查文本内容...", i+1, len(dropdowns))

		text, err := dropdown.Text()
		if err != nil {
			logrus.Debugf("  [%d/%d] 获取文本失败: %v", i+1, len(dropdowns), err)
			continue
		}

		logrus.Debugf("  [%d/%d] 文本内容: %s", i+1, len(dropdowns), text)

		// 检查是否包含任一关键词
		matched := false
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				matched = true
				logrus.Debugf("  [%d/%d] 匹配关键词: %s", i+1, len(dropdowns), kw)
				break
			}
		}

		if matched {
			logrus.Infof("  ✅ 找到匹配的下拉列表 [%d/%d]", i+1, len(dropdowns))
			return dropdown, nil
		}
	}

	logrus.Errorf("  ❌ 未找到包含关键词 %v 的下拉列表", keywords)
	logrus.Errorf("  📊 统计: 总共 %d 个元素，其中 %d 个可见", len(dropdowns), visibleCount)
	return nil, fmt.Errorf("未找到可见的地点下拉列表")
}

func setMarkerTags(page browser.Page, markers []string) error {
	formItems, err := page.Elements(".d-new-form-item")
	if err != nil {
		return fmt.Errorf("查找form-item失败: %w", err)
	}

	var markerButton browser.Element
	for _, item := range formItems {
		text, err := item.Text()
		if err != nil {
			continue
		}
		if strings.Contains(text, "标记地点或标记朋友") {
			btn, err := item.Element("button")
			if err != nil {
				continue
			}
			markerButton = btn
			break
		}
	}

	if markerButton == nil {
		return fmt.Errorf("未找到标记按钮")
	}

	if err := markerButton.Click(); err != nil {
		return fmt.Errorf("点击标记按钮失败: %w", err)
	}
	time.Sleep(800 * time.Millisecond)

	if err := page.WaitForFunction(`() => document.querySelector('div[role="dialog"]') !== null`, 5*time.Second); err != nil {
		return fmt.Errorf("标记对话框未出现: %w", err)
	}

	for _, marker := range markers {
		found, err := searchAndSelectInTab(page, "地点", marker)
		if err != nil {
			continue
		}
		if found {
			continue
		}

		found, err = searchAndSelectInTab(page, "用户", marker)
		if err != nil {
			continue
		}
		if found {
			continue
		}
	}

	confirmButton, err := page.Element("div[role=\"dialog\"] button:has-text(\"确定\")")
	if err != nil {
		return fmt.Errorf("未找到确定按钮: %w", err)
	}

	if err := confirmButton.Click(); err != nil {
		return fmt.Errorf("点击确定按钮失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	return nil
}

func searchAndSelectInTab(page browser.Page, tabName, keyword string) (bool, error) {
	tabs, err := page.Elements("div[role=\"dialog\"] div[role=\"banner\"] ~ *")
	if err != nil {
		return false, fmt.Errorf("查找选项卡失败: %w", err)
	}

	var targetTab browser.Element
	for _, tab := range tabs {
		text, err := tab.Text()
		if err != nil {
			continue
		}
		if strings.Contains(text, tabName) {
			targetTab = tab
			break
		}
	}

	if targetTab == nil {
		return false, fmt.Errorf("未找到 %s 选项卡", tabName)
	}

	if err := targetTab.Click(); err != nil {
		return false, fmt.Errorf("点击 %s 选项卡失败: %w", tabName, err)
	}
	time.Sleep(300 * time.Millisecond)

	searchInput, err := page.Element("div[role=\"dialog\"] input[type=\"text\"]")
	if err != nil {
		return false, fmt.Errorf("未找到搜索框: %w", err)
	}

	if err := searchInput.Click(); err != nil {
		return false, fmt.Errorf("点击搜索框失败: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	if err := searchInput.Fill(keyword); err != nil {
		return false, fmt.Errorf("输入关键词失败: %w", err)
	}
	time.Sleep(1500 * time.Millisecond)

	hasNoResult, _ := page.Has("div[role=\"dialog\"]:has-text(\"没有找到\")")
	if hasNoResult {
		return false, nil
	}

	result, err := page.Eval(fmt.Sprintf(`() => {
		const dialog = document.querySelector('div[role="dialog"]');
		if (!dialog) return null;

		const items = Array.from(dialog.querySelectorAll('div'));

		const matches = items.filter(item => {
			const text = item.textContent;
			return text && text.includes(%s) &&
				!text.includes('搜索') &&
				!text.includes('没有找到') &&
				!text.includes('加载中') &&
				!text.includes('为你推荐');
		});

		if (matches.length > 0) {
			matches[0].click();
			return matches[0].textContent;
		}
		return null;
	}`, strconv.Quote(keyword)))
	if err != nil {
		return false, fmt.Errorf("查找搜索结果失败: %w", err)
	}

	if result == nil {
		return false, nil
	}

	time.Sleep(500 * time.Millisecond)
	return true, nil
}
