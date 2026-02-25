package publish

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

func setLocation(page browser.Page, location string) error {
	slog.Info("📍 开始设置地点", "location", location)

	// 1. 查找地点输入框
	slog.Info("  [1/4] 查找地点输入框 (选择器: .address-box input.d-text)...")
	locationInput, err := page.Element(".address-box input.d-text")
	if err != nil {
		slog.Error("  ❌ 查找地点输入框失败", "error", err)
		// 尝试截图调试
		screenshotPath := fmt.Sprintf("debug_location_input_not_found_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		slog.Info("  📸 已保存截图", "path", screenshotPath)
		return fmt.Errorf("查找地点输入框失败: %w", err)
	}
	slog.Info("  ✅ 找到地点输入框")

	// 2. 点击输入框
	slog.Info("  [2/4] 点击地点输入框...")

	// 先检查元素是否可见
	visible, err := locationInput.IsVisible()
	if err != nil {
		slog.Warn("  ⚠️  无法检查输入框可见性", "error", err)
	} else {
		slog.Info("  📊 输入框可见性", "visible", visible)
	}

	// 尝试滚动到元素
	slog.Info("  📜 尝试滚动到元素位置...")
	if err := locationInput.ScrollIntoView(); err != nil {
		slog.Warn("  ⚠️  滚动失败 (继续)", "error", err)
	} else {
		slog.Info("  ✅ 滚动完成")
	}

	// 直接使用强制点击，避免等待可点击状态超时
	slog.Info("  🖱️  强制点击输入框...")
	if err := locationInput.ClickForce(); err != nil {
		slog.Error("  ❌ 点击失败", "error", err)
		// 截图调试
		screenshotPath := fmt.Sprintf("debug_location_click_failed_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		slog.Info("  📸 已保存截图", "path", screenshotPath)
		return fmt.Errorf("点击地点输入框失败: %w", err)
	}
	slog.Info("  ✅ 点击成功")

	// 3. 输入地点关键词
	slog.Info("  [3/4] 输入地点关键词", "location", location)
	if err := locationInput.Fill(location); err != nil {
		slog.Error("  ❌ 输入地点关键词失败", "error", err)
		return fmt.Errorf("输入地点关键词失败: %w", err)
	}
	slog.Info("  ✅ 关键词已输入")

	slog.Info("  ⏱️  等待下拉列表出现...")
	if err := page.WaitVisible(".d-dropdown-wrapper"); err != nil {
		slog.Warn("  ⚠️  等待下拉列表超时", "error", err)
	}

	// 4. 查找并点击下拉选项
	slog.Info("  [4/4] 查找地点下拉列表...")
	dropdown, err := findVisibleLocationDropdown(page, location)
	if err != nil {
		slog.Error("  ❌ 查找地点下拉列表失败", "error", err)
		// 截图调试
		screenshotPath := fmt.Sprintf("debug_location_dropdown_not_found_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		slog.Info("  📸 已保存截图", "path", screenshotPath)
		return fmt.Errorf("查找地点下拉列表失败: %w", err)
	}
	slog.Info("  ✅ 找到下拉列表")

	firstItem, err := dropdown.Element(".item")
	if err != nil {
		slog.Error("  ❌ 查找地点选项失败", "error", err)
		return fmt.Errorf("查找地点选项失败: %w", err)
	}
	slog.Info("  ✅ 找到第一个地点选项")

	if err := firstItem.Click(); err != nil {
		slog.Error("  ❌ 点击地点选项失败", "error", err)
		return fmt.Errorf("点击地点选项失败: %w", err)
	}
	slog.Info("  ✅ 地点选项已点击")
	if err := page.WaitIdle(); err != nil {
		slog.Warn("  ⚠️  等待页面稳定超时", "error", err)
	}
	slog.Info("✅ 地点设置完成", "location", location)
	return nil
}

func findVisibleLocationDropdown(page browser.Page, keyword string) (browser.Element, error) {
	slog.Info("  🔍 搜索包含关键词的下拉列表", "keyword", keyword)

	// 分解关键词（如 "香港 铜锣湾" -> ["香港", "铜锣湾"]）
	keywords := strings.Fields(keyword)
	slog.Debug("  📝 分解后的关键词", "keywords", keywords)

	dropdowns, err := page.Elements(".d-dropdown-wrapper")
	if err != nil {
		slog.Error("  ❌ 查找 .d-dropdown-wrapper 元素失败", "error", err)
		return nil, err
	}
	slog.Info("  📋 找到 .d-dropdown-wrapper 元素", "count", len(dropdowns))

	visibleCount := 0
	for i, dropdown := range dropdowns {
		visible, err := dropdown.IsVisible()
		if err != nil {
			slog.Debug("  检查可见性失败", "index", i+1, "total", len(dropdowns), "error", err)
			continue
		}

		if !visible {
			slog.Debug("  不可见，跳过", "index", i+1, "total", len(dropdowns))
			continue
		}

		visibleCount++
		slog.Debug("  可见，检查文本内容...", "index", i+1, "total", len(dropdowns))

		text, err := dropdown.Text()
		if err != nil {
			slog.Debug("  获取文本失败", "index", i+1, "total", len(dropdowns), "error", err)
			continue
		}

		slog.Debug("  文本内容", "index", i+1, "total", len(dropdowns), "text", text)

		// 检查是否包含任一关键词
		matched := false
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				matched = true
				slog.Debug("  匹配关键词", "index", i+1, "total", len(dropdowns), "keyword", kw)
				break
			}
		}

		if matched {
			slog.Info("  ✅ 找到匹配的下拉列表", "index", i+1, "total", len(dropdowns))
			return dropdown, nil
		}
	}

	slog.Error("  ❌ 未找到包含关键词的下拉列表", "keywords", keywords)
	slog.Error("  📊 统计", "total", len(dropdowns), "visible", visibleCount)
	return nil, fmt.Errorf("未找到可见的地点下拉列表")
}

func setMarkerTags(page browser.Page, markers []string) error {
	// 功能已禁用
	slog.Warn("⚠️  标记功能已禁用，跳过设置标记", "count", len(markers), "markers", markers)
	return nil

	// 以下代码已禁用
	/*
		slog.Info("🏷️ 开始设置标记", "count", len(markers), "markers", markers)

		// 1. 查找标记按钮
		slog.Info("  [1/5] 查找标记按钮...")
		formItems, err := page.Elements(".d-new-form-item")
		if err != nil {
			slog.Error("  ❌ 查找form-item失败", "error", err)
			return fmt.Errorf("查找form-item失败: %w", err)
		}
		slog.Info("  📋 找到 form-item 元素", "count", len(formItems))

		var markerButton browser.Element
		for i, item := range formItems {
			text, err := item.Text()
			if err != nil {
				slog.Debug("  获取文本失败", "index", i+1, "total", len(formItems), "error", err)
				continue
			}
			slog.Debug("  form-item 文本", "index", i+1, "total", len(formItems), "text", text)

			if strings.Contains(text, "标记地点或标记朋友") {
				btn, err := item.Element("button")
				if err != nil {
					slog.Warn("  查找按钮失败", "index", i+1, "total", len(formItems), "error", err)
					continue
				}
				markerButton = btn
				slog.Info("  ✅ 找到标记按钮", "index", i+1, "total", len(formItems))
				break
			}
		}

		if markerButton == nil {
			slog.Error("  ❌ 未找到标记按钮")
			screenshotPath := fmt.Sprintf("debug_marker_button_not_found_%d.png", time.Now().Unix())
			page.Screenshot(screenshotPath)
			slog.Info("  📸 已保存截图", "path", screenshotPath)
			return fmt.Errorf("未找到标记按钮")
		}

		// 2. 点击标记按钮
		slog.Info("  [2/5] 点击标记按钮...")
		if err := markerButton.Click(); err != nil {
			slog.Error("  ❌ 点击失败", "error", err)
			return fmt.Errorf("点击标记按钮失败: %w", err)
		}
		slog.Info("  ✅ 按钮已点击")
		time.Sleep(800 * time.Millisecond)

		// 3. 等待对话框出现
		slog.Info("  [3/5] 等待标记对话框出现...")
		if err := page.WaitForFunction(`() => document.querySelector('div[role="dialog"]') !== null`, 5*time.Second); err != nil {
			slog.Error("  ❌ 对话框未出现", "error", err)
			screenshotPath := fmt.Sprintf("debug_marker_dialog_not_appear_%d.png", time.Now().Unix())
			page.Screenshot(screenshotPath)
			slog.Info("  📸 已保存截图", "path", screenshotPath)
			return fmt.Errorf("标记对话框未出现: %w", err)
		}
		slog.Info("  ✅ 对话框已出现")

		// 4. 搜索并选择标记
		slog.Info("  [4/5] 搜索并选择标记...")
		for i, marker := range markers {
			slog.Info("  处理标记", "index", i+1, "total", len(markers), "marker", marker)

			// 先在地点选项卡搜索
			found, err := searchAndSelectInTab(page, "地点", marker)
			if err != nil {
				slog.Warn("    ⚠️ 在地点选项卡搜索失败", "error", err)
			}
			if found {
				slog.Info("    ✅ 在地点选项卡找到", "marker", marker)
				continue
			}

			// 在用户选项卡搜索
			found, err = searchAndSelectInTab(page, "用户", marker)
			if err != nil {
				slog.Warn("    ⚠️ 在用户选项卡搜索失败", "error", err)
			}
			if found {
				slog.Info("    ✅ 在用户选项卡找到", "marker", marker)
				continue
			}

			slog.Warn("    ⚠️ 未找到标记", "marker", marker)
		}

		// 5. 点击确定按钮
		slog.Info("  [5/5] 点击确定按钮...")
		confirmButton, err := page.Element("div[role=\"dialog\"] button:has-text(\"确定\")")
		if err != nil {
			slog.Error("  ❌ 未找到确定按钮", "error", err)
			return fmt.Errorf("未找到确定按钮: %w", err)
		}

		if err := confirmButton.Click(); err != nil {
			slog.Error("  ❌ 点击确定按钮失败", "error", err)
			return fmt.Errorf("点击确定按钮失败: %w", err)
		}
		slog.Info("  ✅ 确定按钮已点击")
		time.Sleep(500 * time.Millisecond)

		slog.Info("✅ 标记设置完成")
		return nil
	*/
}

func searchAndSelectInTab(page browser.Page, tabName, keyword string) (bool, error) {
	slog.Info("    🔍 在选项卡中搜索", "tabName", tabName, "keyword", keyword)

	// 1. 查找选项卡
	slog.Info("      [1/5] 查找选项卡", "tabName", tabName)
	tabs, err := page.Elements("div[role=\"dialog\"] div[role=\"banner\"] ~ *")
	if err != nil {
		slog.Error("      ❌ 查找选项卡失败", "error", err)
		return false, fmt.Errorf("查找选项卡失败: %w", err)
	}
	slog.Info("      📋 找到选项卡元素", "count", len(tabs))

	var targetTab browser.Element
	for i, tab := range tabs {
		text, err := tab.Text()
		if err != nil {
			slog.Debug("      获取文本失败", "index", i+1, "total", len(tabs), "error", err)
			continue
		}
		slog.Debug("      选项卡文本", "index", i+1, "total", len(tabs), "text", text)
		if strings.Contains(text, tabName) {
			targetTab = tab
			slog.Info("      ✅ 找到选项卡", "tabName", tabName, "index", i+1, "total", len(tabs))
			break
		}
	}

	if targetTab == nil {
		slog.Error("      ❌ 未找到选项卡", "tabName", tabName)
		return false, fmt.Errorf("未找到 %s 选项卡", tabName)
	}

	// 2. 点击选项卡
	slog.Info("      [2/5] 点击选项卡", "tabName", tabName)
	if err := targetTab.Click(); err != nil {
		slog.Error("      ❌ 点击失败", "error", err)
		return false, fmt.Errorf("点击 %s 选项卡失败: %w", tabName, err)
	}
	slog.Info("      ✅ 选项卡已点击")
	if err := page.WaitDOMStable(2*time.Second, 0.1); err != nil {
		slog.Warn("      ⚠️  等待DOM稳定超时", "error", err)
	}

	// 3. 查找搜索框
	slog.Info("      [3/5] 查找搜索框...")
	searchInput, err := page.Element("div[role=\"dialog\"] input[type=\"text\"]")
	if err != nil {
		slog.Error("      ❌ 查找搜索框失败", "error", err)
		return false, fmt.Errorf("未找到搜索框: %w", err)
	}
	slog.Info("      ✅ 找到搜索框")

	if err := searchInput.Click(); err != nil {
		slog.Error("      ❌ 点击搜索框失败", "error", err)
		return false, fmt.Errorf("点击搜索框失败: %w", err)
	}
	slog.Info("      ✅ 搜索框已点击")

	// 4. 输入关键词
	slog.Info("      [4/5] 输入关键词", "keyword", keyword)
	if err := searchInput.Fill(keyword); err != nil {
		slog.Error("      ❌ 输入关键词失败", "error", err)
		return false, fmt.Errorf("输入关键词失败: %w", err)
	}
	slog.Info("      ✅ 关键词已输入")
	slog.Info("      ⏱️  等待搜索结果出现...")
	if err := page.WaitDOMStable(3*time.Second, 0.1); err != nil {
		slog.Warn("      ⚠️  等待搜索结果超时", "error", err)
	}

	// 5. 查找并点击搜索结果
	slog.Info("      [5/5] 查找搜索结果...")
	hasNoResult, _ := page.Has("div[role=\"dialog\"]:has-text(\"没有找到\")")
	if hasNoResult {
		slog.Warn("      ⚠️  没有找到匹配的结果", "keyword", keyword)
		return false, nil
	}

	slog.Info("      🔍 执行JavaScript查找并点击匹配项...")
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
		slog.Error("      ❌ JavaScript查找失败", "error", err)
		return false, fmt.Errorf("查找搜索结果失败: %w", err)
	}

	if result == nil {
		slog.Warn("      ⚠️  未找到匹配的搜索结果", "keyword", keyword)
		return false, nil
	}

	resultText, _ := result.(string)
	slog.Info("      ✅ 找到并点击了匹配项", "resultText", resultText)
	if err := page.WaitIdle(); err != nil {
		slog.Warn("      ⚠️  等待页面稳定超时", "error", err)
	}
	return true, nil
}
