package xiaohongshu

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

func submitPublish(page browser.Page, title, content string, tags []string, location string, settings PublishImageContent, scheduleTime *time.Time) error {

	titleElem, err := page.Element("div.d-input input")
	if err != nil {
		return errors.Wrap(err, "找不到标题输入框")
	}
	if err := titleElem.Fill(title); err != nil {
		return errors.Wrap(err, "输入标题失败")
	}

	_ = page.WaitForSelector("div.ql-editor, div.edit-container", 3*time.Second)
	if err := checkTitleMaxLength(page); err != nil {
		return err
	}
	slog.Info("检查标题长度：通过")

	_ = page.WaitForSelector("div.ql-editor, [role='textbox']", 3*time.Second)

	if contentElem, ok := getContentElement(page); ok {
		if err := contentElem.Fill(content); err != nil {
			return errors.Wrap(err, "输入内容失败")
		}
		inputTags(page, contentElem, tags)
	} else {
		return errors.New("没有找到内容输入框")
	}

	_ = page.WaitForSelector("div.edit-container", 2*time.Second)

	if err := checkContentMaxLength(page); err != nil {
		return err
	}
	slog.Info("检查正文长度：通过")

	if location != "" {
		slog.Info("开始设置地点", "location", location)
		if err := setLocation(page, location); err != nil {
			return errors.Wrap(err, "设置地点失败")
		}
		slog.Info("地点设置完成", "location", location)
	}

	if settings.Collection != "" {
		if err := setCollection(page, settings.Collection); err != nil {
			return errors.Wrap(err, "设置合集失败")
		}
		slog.Info("合集设置完成", "collection", settings.Collection)
	}

	if settings.GroupChat != "" {
		if err := setGroupChat(page, settings.GroupChat); err != nil {
			return errors.Wrap(err, "设置群聊失败")
		}
		slog.Info("群聊设置完成", "groupChat", settings.GroupChat)
	}

	if len(settings.MarkerTags) > 0 {
		slog.Info("开始设置标记", "markers", settings.MarkerTags)
		if err := setMarkerTags(page, settings.MarkerTags); err != nil {
			return errors.Wrap(err, "设置标记失败")
		}
		slog.Info("标记设置完成", "markers", settings.MarkerTags)
	}

	if settings.OriginalClaim {
		if err := setOriginalClaim(page); err != nil {
			return errors.Wrap(err, "设置原创声明失败")
		}
		slog.Info("原创声明设置完成")
	}

	if settings.ContentType != "" {
		if err := setContentType(page, settings.ContentType); err != nil {
			return errors.Wrap(err, "设置内容类型失败")
		}
		slog.Info("内容类型设置完成", "contentType", settings.ContentType)
	}

	if settings.VisibleScope != "" {
		if err := setVisibleScope(page, settings.VisibleScope); err != nil {
			return errors.Wrap(err, "设置可见范围失败")
		}
		slog.Info("可见范围设置完成", "visibleScope", settings.VisibleScope)
	}

	if settings.AllowDuet != nil {
		if err := setCheckbox(page, "允许合拍", *settings.AllowDuet); err != nil {
			return errors.Wrap(err, "设置允许合拍失败")
		}
		slog.Info("允许合拍设置完成", "allowDuet", *settings.AllowDuet)
	}

	if settings.AllowCopy != nil {
		if err := setCheckbox(page, "允许正文复制", *settings.AllowCopy); err != nil {
			return errors.Wrap(err, "设置允许复制失败")
		}
		slog.Info("允许复制设置完成", "allowCopy", *settings.AllowCopy)
	}

	if scheduleTime != nil {
		if err := setSchedulePublish(page, *scheduleTime); err != nil {
			return errors.Wrap(err, "设置定时发布失败")
		}
		slog.Info("定时发布设置完成", "schedule_time", scheduleTime.Format("2006-01-02 15:04"))
	}

	submitButton, err := page.Element("div.submit div.d-button-content")
	if err != nil {
		return errors.Wrap(err, "找不到发布按钮")
	}
	if err := submitButton.Click(); err != nil {
		return errors.Wrap(err, "点击发布按钮失败")
	}

	slog.Info("已点击发布按钮，等待API响应...")
	return nil
}

// 检查标题是否超过最大长度
func checkTitleMaxLength(page browser.Page) error {
	has, err := page.Has(`div.title-container div.max_suffix`)
	if err != nil {
		return errors.Wrap(err, "检查标题长度元素失败")
	}
	if !has {
		return nil
	}
	elem, err := page.Element(`div.title-container div.max_suffix`)
	if err != nil {
		return errors.Wrap(err, "获取标题长度元素失败")
	}
	titleLength, err := elem.Text()
	if err != nil {
		return errors.Wrap(err, "获取标题长度文本失败")
	}
	return makeMaxLengthError(titleLength)
}

func checkContentMaxLength(page browser.Page) error {
	has, err := page.Has(`div.edit-container div.length-error`)
	if err != nil {
		return errors.Wrap(err, "检查正文长度元素失败")
	}
	if !has {
		return nil
	}
	elem, err := page.Element(`div.edit-container div.length-error`)
	if err != nil {
		return errors.Wrap(err, "获取正文长度元素失败")
	}
	contentLength, err := elem.Text()
	if err != nil {
		return errors.Wrap(err, "获取正文长度文本失败")
	}
	return makeMaxLengthError(contentLength)
}

func makeMaxLengthError(elemText string) error {
	parts := strings.Split(elemText, "/")
	if len(parts) != 2 {
		return errors.Errorf("长度超过限制: %s", elemText)
	}
	currLen, maxLen := parts[0], parts[1]
	return errors.Errorf("当前输入长度为%s，最大长度为%s", currLen, maxLen)
}

// 查找内容输入框 - 尝试两种样式
func getContentElement(page browser.Page) (browser.Element, bool) {
	elem, err := page.Element("div.ql-editor")
	if err == nil {
		slog.Info("找到内容元素：div.ql-editor")
		return elem, true
	}
	elem, err = findTextboxByPlaceholder(page)
	if err == nil {
		slog.Info("找到内容元素：通过 placeholder")
		return elem, true
	}
	slog.Warn("no content element found by any method")
	return nil, false
}

func inputTags(page browser.Page, contentElem browser.Element, tags []string) {
	if len(tags) == 0 {
		return
	}
	_ = contentElem.WaitStable(500 * time.Millisecond)
	for i := 0; i < 20; i++ {
		contentElem.Press("ArrowDown")
		time.Sleep(10 * time.Millisecond)
	}
	contentElem.Press("Enter")
	contentElem.Press("Enter")
	_ = contentElem.WaitStable(500 * time.Millisecond)
	for _, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		inputTag(page, contentElem, tag)
	}
}

func inputTag(page browser.Page, contentElem browser.Element, tag string) {
	contentElem.Input("#")
	_ = page.WaitForSelector("#creator-editor-topic-container", 2*time.Second)
	for _, char := range tag {
		contentElem.Input(string(char))
		time.Sleep(50 * time.Millisecond)
	}
	_ = page.WaitForSelector("#creator-editor-topic-container", 2*time.Second)
	topicContainer, err := page.Element("#creator-editor-topic-container")
	if err == nil && topicContainer != nil {
		firstItem, err := topicContainer.Element(".item")
		if err == nil && firstItem != nil {
			firstItem.Click()
			slog.Info("成功点击标签联想选项", "tag", tag)
			_ = page.WaitIdle()
		} else {
			slog.Warn("未找到标签联想选项，直接输入空格", "tag", tag)
			contentElem.Input(" ")
		}
	} else {
		slog.Warn("未找到标签联想下拉框，直接输入空格", "tag", tag)
		contentElem.Input(" ")
	}
	_ = page.WaitHidden("#creator-editor-topic-container")
}

func findTextboxByPlaceholder(page browser.Page) (browser.Element, error) {
	elements, err := page.Elements("p")
	if err != nil || elements == nil {
		return nil, errors.New("no p elements found")
	}
	placeholderElem := findPlaceholderElement(elements, "输入正文描述")
	if placeholderElem == nil {
		return nil, errors.New("no placeholder element found")
	}
	textboxElem := findTextboxParent(page, placeholderElem)
	if textboxElem == nil {
		return nil, errors.New("no textbox parent found")
	}
	return textboxElem, nil
}

func findPlaceholderElement(elements []browser.Element, searchText string) browser.Element {
	for _, elem := range elements {
		placeholder, err := elem.Attribute("data-placeholder")
		if err != nil {
			continue
		}
		if strings.Contains(placeholder, searchText) {
			return elem
		}
	}
	return nil
}

func findTextboxParent(page browser.Page, elem browser.Element) browser.Element {
	_, err := elem.Eval(`() => {
		let current = this;
		for (let i = 0; i < 5; i++) {
			const parent = current.parentElement;
			if (!parent) break;
			const role = parent.getAttribute('role');
			if (role === 'textbox') {
				parent.setAttribute('data-temp-textbox', 'true');
				return true;
			}
			current = parent;
		}
		return false;
	}`)
	if err != nil {
		return nil
	}
	textbox, err := page.Element("[data-temp-textbox='true']")
	if err == nil {
		textbox.Eval(`() => { this.removeAttribute('data-temp-textbox'); }`)
		return textbox
	}
	return nil
}

// setSchedulePublish 设置定时发布时间
func setSchedulePublish(page browser.Page, t time.Time) error {
	if err := clickScheduleRadio(page); err != nil {
		return err
	}
	if err := page.WaitForSelector("input.el-input__inner[placeholder='选择日期和时间']", 5*time.Second); err != nil {
		logrus.Warnf("等待时间选择器出现: %v，继续尝试", err)
	}
	if err := clickDateTimePicker(page); err != nil {
		return err
	}
	if err := page.WaitForSelector("input.el-input__inner[placeholder='选择日期']", 5*time.Second); err != nil {
		logrus.Warnf("等待日期输入框出现: %v，继续尝试", err)
	}
	if err := setDateTime(page, t); err != nil {
		return err
	}
	if err := page.WaitForSelector("button.el-picker-panel__link-btn", 3*time.Second); err != nil {
		logrus.Warnf("等待确定按钮: %v，继续尝试", err)
	}
	if err := clickConfirmButton(page); err != nil {
		return err
	}
	if err := page.WaitHidden("input.el-input__inner[placeholder='选择日期']"); err != nil {
		logrus.Warnf("等待日期面板关闭: %v，继续尝试", err)
	}
	return nil
}

func clickScheduleRadio(page browser.Page) error {
	labels, err := page.Elements("span.el-radio__label")
	if err != nil {
		return errors.Wrap(err, "查找 radio label 失败")
	}
	for _, label := range labels {
		text, err := label.Text()
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) == "定时发布" {
			if err := label.Click(); err != nil {
				return errors.Wrap(err, "点击定时发布按钮失败")
			}
			slog.Info("已点击定时发布按钮")
			return nil
		}
	}
	return errors.New("未找到定时发布按钮")
}

func clickDateTimePicker(page browser.Page) error {
	picker, err := page.Element("input.el-input__inner[placeholder='选择日期和时间']")
	if err != nil {
		return errors.Wrap(err, "查找时间选择器失败")
	}
	if err := picker.Click(); err != nil {
		return errors.Wrap(err, "点击时间选择器失败")
	}
	slog.Info("已点击时间选择器")
	return nil
}

func setDateTime(page browser.Page, t time.Time) error {
	dateStr := t.Format("2006-01-02")
	timeStr := t.Format("15:04")

	dateInput, err := page.Element("input.el-input__inner[placeholder='选择日期']")
	if err != nil {
		return errors.Wrap(err, "查找日期输入框失败")
	}
	if err := dateInput.Click(); err != nil {
		return errors.Wrap(err, "点击日期输入框失败")
	}
	if err := dateInput.Press("Control+a"); err != nil {
		return errors.Wrap(err, "选择日期文本失败")
	}
	if err := dateInput.Input(dateStr); err != nil {
		return errors.Wrap(err, "输入日期失败")
	}
	slog.Info("已设置日期", "date", dateStr)

	if err := page.WaitForSelector("input.el-input__inner[placeholder='选择时间']", 3*time.Second); err != nil {
		logrus.Warnf("等待时间输入框: %v，继续尝试", err)
	}
	timeInput, err := page.Element("input.el-input__inner[placeholder='选择时间']")
	if err != nil {
		return errors.Wrap(err, "查找时间输入框失败")
	}
	if err := timeInput.Click(); err != nil {
		return errors.Wrap(err, "点击时间输入框失败")
	}
	if err := timeInput.Press("Control+a"); err != nil {
		return errors.Wrap(err, "选择时间文本失败")
	}
	if err := timeInput.Input(timeStr); err != nil {
		return errors.Wrap(err, "输入时间失败")
	}
	slog.Info("已设置时间", "time", timeStr)
	return nil
}

func clickConfirmButton(page browser.Page) error {
	buttons, err := page.Elements("button.el-picker-panel__link-btn")
	if err != nil {
		return errors.Wrap(err, "查找确定按钮失败")
	}
	for _, btn := range buttons {
		text, err := btn.Text()
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) == "确定" {
			if err := btn.Click(); err != nil {
				return errors.Wrap(err, "点击确定按钮失败")
			}
			slog.Info("已点击确定按钮")
			return nil
		}
	}
	return errors.New("未找到确定按钮")
}

// setLocation 设置地点
func setLocation(page browser.Page, location string) error {
	locationInput, err := page.Element(".address-box input.d-text")
	if err != nil {
		return errors.Wrap(err, "查找地点输入框失败")
	}
	if err := locationInput.Click(); err != nil {
		return errors.Wrap(err, "点击地点输入框失败")
	}
	_ = page.WaitIdle()
	if err := locationInput.Fill(location); err != nil {
		return errors.Wrap(err, "输入地点关键词失败")
	}
	slog.Info("已输入地点关键词", "location", location)

	if err := page.WaitForSelector(".d-dropdown-wrapper", 5*time.Second); err != nil {
		logrus.Warnf("等待地点下拉列表出现: %v，继续尝试", err)
	}
	dropdown, err := findVisibleLocationDropdown(page, location)
	if err != nil {
		return errors.Wrap(err, "查找地点下拉列表失败")
	}
	firstItem, err := dropdown.Element(".item")
	if err != nil {
		return errors.Wrap(err, "查找地点选项失败")
	}
	if err := firstItem.Click(); err != nil {
		return errors.Wrap(err, "点击地点选项失败")
	}
	slog.Info("已选择地点", "location", location)
	if err := page.WaitHidden(".d-dropdown-wrapper"); err != nil {
		logrus.Warnf("等待地点下拉框关闭: %v，继续尝试", err)
	}
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
	return nil, errors.New("未找到可见的地点下拉列表")
}

// setCollection 设置合集
func setCollection(page browser.Page, collection string) error {
	formItems, err := page.Elements(".d-new-form-item")
	if err != nil {
		return errors.Wrap(err, "查找form-item失败")
	}
	for _, item := range formItems {
		title, err := item.Text()
		if err != nil {
			continue
		}
		if !strings.Contains(title, "添加合集") {
			continue
		}
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找合集选择器失败")
		}
		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击合集选择器失败")
		}
		_ = page.WaitVisible(".collection-drop-down")
		dropdown, err := page.Element(".collection-drop-down")
		if err != nil {
			return errors.Wrap(err, "合集下拉列表未出现")
		}
		items, err := dropdown.Elements(".d-option, .item")
		if err != nil || len(items) == 0 {
			return errors.New("未找到合集选项，请先在APP端创建合集")
		}
		for _, option := range items {
			text, err := option.Text()
			if err != nil {
				continue
			}
			if strings.Contains(text, collection) {
				if err := option.Click(); err != nil {
					return errors.Wrap(err, "点击合集选项失败")
				}
				_ = page.WaitHidden(".collection-drop-down")
				return nil
			}
		}
		return errors.Errorf("未找到名为 '%s' 的合集", collection)
	}
	return errors.New("未找到合集设置区域")
}

// setGroupChat 设置群聊
func setGroupChat(page browser.Page, groupChat string) error {
	formItems, err := page.Elements(".d-new-form-item")
	if err != nil {
		return errors.Wrap(err, "查找form-item失败")
	}
	for _, item := range formItems {
		title, err := item.Text()
		if err != nil {
			continue
		}
		if !strings.Contains(title, "关联群聊") {
			continue
		}
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找群聊选择器失败")
		}
		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击群聊选择器失败")
		}
		_ = page.WaitVisible(".d-dropdown")
		dropdowns, err := page.Elements(".d-dropdown")
		if err != nil {
			return errors.Wrap(err, "群聊下拉列表未出现")
		}
		for _, dropdown := range dropdowns {
			visible, _ := dropdown.IsVisible()
			if !visible {
				continue
			}
			items, err := dropdown.Elements(".d-option, .item")
			if err != nil {
				continue
			}
			for _, option := range items {
				text, err := option.Text()
				if err != nil {
					continue
				}
				if strings.Contains(text, groupChat) {
					if err := option.Click(); err != nil {
						return errors.Wrap(err, "点击群聊选项失败")
					}
					_ = page.WaitIdle()
					return nil
				}
			}
		}
		return errors.Errorf("未找到名为 '%s' 的群聊", groupChat)
	}
	return errors.New("未找到群聊设置区域")
}

// setOriginalClaim 设置原创声明
func setOriginalClaim(page browser.Page) error {
	buttons, err := page.Elements("span, button")
	if err != nil {
		return errors.Wrap(err, "查找按钮失败")
	}
	for _, btn := range buttons {
		text, err := btn.Text()
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) == "去声明" {
			if err := btn.Click(); err != nil {
				return errors.Wrap(err, "点击去声明按钮失败")
			}
			slog.Info("已点击去声明按钮")
			_ = page.WaitIdle()
			return nil
		}
	}
	return errors.New("未找到去声明按钮")
}

// setContentType 设置内容类型声明
func setContentType(page browser.Page, contentType string) error {
	formItems, err := page.Elements(".d-new-form-item")
	if err != nil {
		return errors.Wrap(err, "查找form-item失败")
	}
	for _, item := range formItems {
		title, err := item.Text()
		if err != nil {
			continue
		}
		if !strings.Contains(title, "内容类型声明") {
			continue
		}
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找内容类型选择器失败")
		}
		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击内容类型选择器失败")
		}
		_ = page.WaitVisible(".d-dropdown")
		dropdowns, err := page.Elements(".d-dropdown")
		if err != nil {
			return errors.Wrap(err, "内容类型下拉列表未出现")
		}
		for _, dropdown := range dropdowns {
			visible, _ := dropdown.IsVisible()
			if !visible {
				continue
			}
			items, err := dropdown.Elements(".d-option")
			if err != nil {
				continue
			}
			for _, option := range items {
				text, err := option.Text()
				if err != nil {
					continue
				}
				if strings.Contains(text, contentType) {
					if err := option.Click(); err != nil {
						return errors.Wrap(err, "点击内容类型选项失败")
					}
					_ = page.WaitIdle()
					return nil
				}
			}
		}
		return errors.Errorf("未找到内容类型 '%s'", contentType)
	}
	return errors.New("未找到内容类型设置区域")
}

// setVisibleScope 设置可见范围
func setVisibleScope(page browser.Page, scope string) error {
	formItems, err := page.Elements(".d-new-form-item")
	if err != nil {
		return errors.Wrap(err, "查找form-item失败")
	}
	for _, item := range formItems {
		title, err := item.Text()
		if err != nil {
			continue
		}
		if !strings.Contains(title, "可见范围") {
			continue
		}
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找可见范围选择器失败")
		}
		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击可见范围选择器失败")
		}
		_ = page.WaitVisible(".d-dropdown")
		dropdowns, err := page.Elements(".d-dropdown")
		if err != nil {
			return errors.Wrap(err, "可见范围下拉列表未出现")
		}
		for _, dropdown := range dropdowns {
			visible, _ := dropdown.IsVisible()
			if !visible {
				continue
			}
			items, err := dropdown.Elements(".d-option, .custom-option")
			if err != nil {
				continue
			}
			for _, option := range items {
				text, err := option.Text()
				if err != nil {
					continue
				}
				if strings.Contains(text, scope) {
					if err := option.Click(); err != nil {
						return errors.Wrap(err, "点击可见范围选项失败")
					}
					_ = page.WaitIdle()
					return nil
				}
			}
		}
		return errors.Errorf("未找到可见范围 '%s'", scope)
	}
	return errors.New("未找到可见范围设置区域")
}

// setCheckbox 设置checkbox状态
func setCheckbox(page browser.Page, labelText string, checked bool) error {
	formItems, err := page.Elements(".d-new-form-item")
	if err != nil {
		return errors.Wrap(err, "查找form-item失败")
	}
	for _, item := range formItems {
		title, err := item.Text()
		if err != nil {
			continue
		}
		if !strings.Contains(title, labelText) {
			continue
		}
		checkbox, err := item.Element("input[type='checkbox']")
		if err != nil {
			return errors.Wrapf(err, "查找 %s checkbox失败", labelText)
		}
		isChecked, err := checkbox.Eval(`() => this.checked`)
		if err != nil {
			return errors.Wrap(err, "获取checkbox状态失败")
		}
		currentChecked := false
		if b, ok := isChecked.(bool); ok {
			currentChecked = b
		}
		if currentChecked != checked {
			if err := checkbox.Click(); err != nil {
				return errors.Wrapf(err, "点击 %s checkbox失败", labelText)
			}
			slog.Info("已切换checkbox状态", "label", labelText, "checked", checked)
			_ = page.WaitIdle()
		}
		return nil
	}
	return errors.Errorf("未找到 %s checkbox", labelText)
}

// setMarkerTags 设置标记（地点或用户）
func setMarkerTags(page browser.Page, markers []string) error {
	formItems, err := page.Elements(".d-new-form-item")
	if err != nil {
		return errors.Wrap(err, "查找form-item失败")
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
		return errors.New("未找到标记按钮")
	}
	if err := markerButton.Click(); err != nil {
		return errors.Wrap(err, "点击标记按钮失败")
	}
	if err := page.WaitForFunction(`() => document.querySelector('div[role="dialog"]') !== null`, 5*time.Second); err != nil {
		return errors.Wrap(err, "标记对话框未出现")
	}
	for _, marker := range markers {
		found, err := searchAndSelectInTab(page, "地点", marker)
		if err != nil {
			slog.Warn("在地点选项卡搜索失败", "marker", marker, "error", err)
		}
		if found {
			slog.Info("在地点选项卡找到标记", "marker", marker)
			continue
		}
		found, err = searchAndSelectInTab(page, "用户", marker)
		if err != nil {
			slog.Warn("在用户选项卡搜索失败", "marker", marker, "error", err)
		}
		if found {
			slog.Info("在用户选项卡找到标记", "marker", marker)
			continue
		}
		slog.Warn("未找到标记", "marker", marker)
	}
	confirmButton, err := page.Element("div[role=\"dialog\"] button:has-text(\"确定\")")
	if err != nil {
		return errors.Wrap(err, "未找到确定按钮")
	}
	if err := confirmButton.Click(); err != nil {
		return errors.Wrap(err, "点击确定按钮失败")
	}
	_ = page.WaitIdle()
	return nil
}

// searchAndSelectInTab 在指定选项卡中搜索并选择标记
func searchAndSelectInTab(page browser.Page, tabName, keyword string) (bool, error) {
	tabs, err := page.Elements("div[role=\"dialog\"] div[role=\"banner\"] ~ *")
	if err != nil {
		return false, errors.Wrap(err, "查找选项卡失败")
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
		return false, errors.Errorf("未找到 %s 选项卡", tabName)
	}
	if err := targetTab.Click(); err != nil {
		return false, errors.Wrapf(err, "点击 %s 选项卡失败", tabName)
	}
	_ = page.WaitDOMStable(2*time.Second, 0.1)

	searchInput, err := page.Element("div[role=\"dialog\"] input[type=\"text\"]")
	if err != nil {
		return false, errors.Wrap(err, "未找到搜索框")
	}
	if err := searchInput.Click(); err != nil {
		return false, errors.Wrap(err, "点击搜索框失败")
	}
	if err := searchInput.Fill(keyword); err != nil {
		return false, errors.Wrap(err, "输入关键词失败")
	}
	_ = page.WaitDOMStable(3*time.Second, 0.1)

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
		return false, errors.Wrap(err, "查找搜索结果失败")
	}
	if result == nil {
		return false, nil
	}
	slog.Info("已选择搜索结果", "keyword", keyword, "selected", result)
	_ = page.WaitIdle()
	return true, nil
}
