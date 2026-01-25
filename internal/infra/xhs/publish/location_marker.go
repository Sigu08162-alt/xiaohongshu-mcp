package publish

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

func setLocation(page browser.Page, location string) error {
	locationInput, err := page.Element(".address-box input.d-text")
	if err != nil {
		return fmt.Errorf("查找地点输入框失败: %w", err)
	}

	if err := locationInput.Click(); err != nil {
		return fmt.Errorf("点击地点输入框失败: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	if err := locationInput.Fill(location); err != nil {
		return fmt.Errorf("输入地点关键词失败: %w", err)
	}

	time.Sleep(1500 * time.Millisecond)

	dropdown, err := findVisibleLocationDropdown(page, location)
	if err != nil {
		return fmt.Errorf("查找地点下拉列表失败: %w", err)
	}

	firstItem, err := dropdown.Element(".item")
	if err != nil {
		return fmt.Errorf("查找地点选项失败: %w", err)
	}

	if err := firstItem.Click(); err != nil {
		return fmt.Errorf("点击地点选项失败: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func findVisibleLocationDropdown(page browser.Page, keyword string) (browser.Element, error) {
	dropdowns, err := page.Elements(".d-dropdown-wrapper")
	if err != nil {
		return nil, err
	}

	for _, dropdown := range dropdowns {
		visible, err := dropdown.IsVisible()
		if err != nil || !visible {
			continue
		}

		text, err := dropdown.Text()
		if err != nil {
			continue
		}

		if strings.Contains(text, keyword) {
			return dropdown, nil
		}
	}

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
