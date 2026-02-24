package xiaohongshu

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

// PublishImageContent 发布图文内容
type PublishImageContent struct {
	Title         string
	Content       string
	Tags          []string
	ImagePaths    []string
	Location      string     // 地点名称
	Collection    string     // 合集名称
	GroupChat     string     // 群聊名称
	MarkerTags    []string   // 标记的地点或用户（支持多个）
	OriginalClaim bool       // 是否声明原创
	ContentType   string     // 内容类型声明: 虚构演绎/AI合成/来源声明
	VisibleScope  string     // 可见范围: 公开可见/仅自己可见/仅互关好友可见/只给谁看/不给谁看
	AllowDuet     *bool      // 是否允许合拍，nil 使用默认值
	AllowCopy     *bool      // 是否允许正文复制，nil 使用默认值
	ScheduleTime  *time.Time // 定时发布时间，nil 表示立即发布
}

type PublishAction struct {
	page browser.Page
}

const (
	// 直接访问图文上传页面，避免需要点击TAB切换
	// 小红书页面更新后，原URL默认打开视频上传页，需添加 target=image 参数
	urlOfPublic = `https://creator.xiaohongshu.com/publish/publish?source=official&target=image`
)

func NewPublishImageAction(page browser.Page) (*PublishAction, error) {

	pp := page.WithTimeout(300 * time.Second)

	// 使用更稳健的导航和等待策略
	if err := pp.Goto(urlOfPublic); err != nil {
		return nil, errors.Wrap(err, "导航到发布页面失败")
	}

	// 等待页面加载
	if err := pp.WaitLoad(); err != nil {
		logrus.Warnf("等待页面加载出现问题: %v，继续尝试", err)
	}

	// 等待上传内容区域出现，替代固定等待
	if err := pp.WaitForSelector("div.upload-content", 10*time.Second); err != nil {
		logrus.Warnf("等待上传区域出现出现问题: %v，继续尝试", err)
	}

	// 等待页面稳定
	if err := pp.WaitDOMStable(time.Second, 0.1); err != nil {
		logrus.Warnf("等待 DOM 稳定出现问题: %v，继续尝试", err)
	}

	if err := mustClickPublishTab(pp, "上传图文"); err != nil {
		logrus.Errorf("点击上传图文 TAB 失败: %v", err)
		return nil, err
	}

	// 等待上传输入框就绪，替代固定等待
	if err := pp.WaitForSelector(".upload-input", 5*time.Second); err != nil {
		logrus.Warnf("等待上传输入框出现出现问题: %v，继续尝试", err)
	}

	return &PublishAction{
		page: pp,
	}, nil
}

func (p *PublishAction) Publish(ctx context.Context, content PublishImageContent) error {
	if len(content.ImagePaths) == 0 {
		return errors.New("图片不能为空")
	}

	page := p.page.WithContext(ctx)

	// TODO: Playwright 网络拦截功能待实现
	// 暂时使用 UI 检测方式

	if err := uploadImages(page, content.ImagePaths); err != nil {
		return errors.Wrap(err, "小红书上传图片失败")
	}

	tags := content.Tags
	if len(tags) >= 10 {
		logrus.Warnf("标签数量超过10，截取前10个标签")
		tags = tags[:10]
	}

	logrus.Infof("发布内容: title=%s, images=%v, tags=%v, location=%s, schedule=%v", content.Title, len(content.ImagePaths), tags, content.Location, content.ScheduleTime)

	if err := submitPublish(page, content.Title, content.Content, tags, content.Location, content, content.ScheduleTime); err != nil {
		return errors.Wrap(err, "小红书发布失败")
	}

	// 等待发布完成（等待页面跳转或成功提示）
	// TODO: replace with WaitForNavigation when page interface supports it
	// 等待页面URL变化或成功提示出现
	if err := page.WaitForFunction(`() => !window.location.href.includes('/publish/publish') || document.querySelector('.d-message--success') !== null`, 10*time.Second); err != nil {
		logrus.Warnf("等待发布完成: %v", err)
	}
	return nil
}

func removePopCover(page browser.Page) {

	// 先移除弹窗封面
	has, err := page.Has("div.d-popover")
	if err != nil {
		return
	}
	if has {
		elem, err := page.Element("div.d-popover")
		if err == nil {
			elem.Remove()
		}
	}

	// 兜底：点击一下空位置吧
	clickEmptyPosition(page)
}

func clickEmptyPosition(page browser.Page) {
	x := 380 + rand.Intn(100)
	y := 20 + rand.Intn(60)
	mouse := page.Mouse()
	mouse.MoveTo(float64(x), float64(y))
	mouse.Click(browser.MouseButtonLeft)
}

func mustClickPublishTab(page browser.Page, tabname string) error {
	elem, err := page.Element(`div.upload-content`)
	if err != nil {
		return err
	}
	if err := elem.WaitVisible(); err != nil {
		return err
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		tab, blocked, err := getTabElement(page, tabname)
		if err != nil {
			logrus.Warnf("获取发布 TAB 元素失败: %v", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if tab == nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if blocked {
			logrus.Info("发布 TAB 被遮挡，尝试移除遮挡")
			removePopCover(page)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if err := tab.Click(); err != nil {
			logrus.Warnf("点击发布 TAB 失败: %v", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		return nil
	}

	return errors.Errorf("没有找到发布 TAB - %s", tabname)
}

func getTabElement(page browser.Page, tabname string) (browser.Element, bool, error) {
	elems, err := page.Elements("div.creator-tab")
	if err != nil {
		return nil, false, err
	}

	for _, elem := range elems {
		if !isElementVisible(elem) {
			continue
		}

		text, err := elem.Text()
		if err != nil {
			logrus.Debugf("获取发布 TAB 文本失败: %v", err)
			continue
		}

		if strings.TrimSpace(text) != tabname {
			continue
		}

		blocked, err := isElementBlocked(elem)
		if err != nil {
			return nil, false, err
		}

		return elem, blocked, nil
	}

	return nil, false, nil
}

func isElementBlocked(elem browser.Element) (bool, error) {
	result, err := elem.Eval(`() => {
		const rect = this.getBoundingClientRect();
		if (rect.width === 0 || rect.height === 0) {
			return true;
		}
		const x = rect.left + rect.width / 2;
		const y = rect.top + rect.height / 2;
		const target = document.elementFromPoint(x, y);
		return !(target === this || this.contains(target));
	}`)
	if err != nil {
		return false, err
	}

	// 将结果转换为 bool
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

func uploadImages(page browser.Page, imagesPaths []string) error {
	pp := page.WithTimeout(30 * time.Second)

	// 验证文件路径有效性
	validPaths := make([]string, 0, len(imagesPaths))
	for _, path := range imagesPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			logrus.Warnf("图片文件不存在: %s", path)
			continue
		}
		validPaths = append(validPaths, path)

		logrus.Infof("获取有效图片：%s", path)
	}

	// 等待上传输入框出现
	uploadInput, err := pp.Element(".upload-input")
	if err != nil {
		return errors.Wrap(err, "找不到上传输入框")
	}

	// 上传多个文件
	if err := uploadInput.SetFiles(validPaths); err != nil {
		return errors.Wrap(err, "设置文件失败")
	}

	// 等待并验证上传完成
	return waitForUploadComplete(pp, len(validPaths))
}

// waitForUploadComplete 等待并验证上传完成
func waitForUploadComplete(page browser.Page, expectedCount int) error {
	maxWaitTime := 60 * time.Second
	checkInterval := 500 * time.Millisecond
	start := time.Now()

	slog.Info("开始等待图片上传完成", "expected_count", expectedCount)

	for time.Since(start) < maxWaitTime {
		// 使用具体的pr类名检查已上传的图片
		uploadedImages, err := page.Elements(".img-preview-area .pr")

		slog.Info("uploadedImages", "uploadedImages", uploadedImages)

		if err == nil {
			currentCount := len(uploadedImages)
			slog.Info("检测到已上传图片", "current_count", currentCount, "expected_count", expectedCount)
			if currentCount >= expectedCount {
				slog.Info("所有图片上传完成", "count", currentCount)
				return nil
			}
		} else {
			slog.Debug("未找到已上传图片元素")
		}

		time.Sleep(checkInterval)
	}

	return errors.New("上传超时，请检查网络连接和图片大小")
}

func submitPublish(page browser.Page, title, content string, tags []string, location string, settings PublishImageContent, scheduleTime *time.Time) error {

	titleElem, err := page.Element("div.d-input input")
	if err != nil {
		return errors.Wrap(err, "找不到标题输入框")
	}
	if err := titleElem.Fill(title); err != nil {
		return errors.Wrap(err, "输入标题失败")
	}

	// 检查一下 title 的长度
	// 等待长度提示元素渲染（若存在则可见；等待内容区域出现作为页面就绪标志）
	_ = page.WaitForSelector("div.ql-editor, div.edit-container", 3*time.Second) // 等内容区域就绪
	if err := checkTitleMaxLength(page); err != nil {
		return err
	}
	slog.Info("检查标题长度：通过")

	// 等待内容输入框可用
	_ = page.WaitForSelector("div.ql-editor, [role='textbox']", 3*time.Second)

	if contentElem, ok := getContentElement(page); ok {
		if err := contentElem.Fill(content); err != nil {
			return errors.Wrap(err, "输入内容失败")
		}

		inputTags(page, contentElem, tags)

	} else {
		return errors.New("没有找到内容输入框")
	}

	// 等待内容长度检查元素渲染
	_ = page.WaitForSelector("div.edit-container", 2*time.Second)

	// 正文的长度的判定：
	if err := checkContentMaxLength(page); err != nil {
		return err
	}
	slog.Info("检查正文长度：通过")

	// 设置地点
	if location != "" {
		slog.Info("开始设置地点", "location", location)
		if err := setLocation(page, location); err != nil {
			return errors.Wrap(err, "设置地点失败")
		}
		slog.Info("地点设置完成", "location", location)
	}

	// 设置合集
	if settings.Collection != "" {
		if err := setCollection(page, settings.Collection); err != nil {
			return errors.Wrap(err, "设置合集失败")
		}
		slog.Info("合集设置完成", "collection", settings.Collection)
	}

	// 设置群聊
	if settings.GroupChat != "" {
		if err := setGroupChat(page, settings.GroupChat); err != nil {
			return errors.Wrap(err, "设置群聊失败")
		}
		slog.Info("群聊设置完成", "groupChat", settings.GroupChat)
	}

	// 设置标记（地点或用户）
	if len(settings.MarkerTags) > 0 {
		slog.Info("开始设置标记", "markers", settings.MarkerTags)
		if err := setMarkerTags(page, settings.MarkerTags); err != nil {
			return errors.Wrap(err, "设置标记失败")
		}
		slog.Info("标记设置完成", "markers", settings.MarkerTags)
	}

	// 设置原创声明
	if settings.OriginalClaim {
		if err := setOriginalClaim(page); err != nil {
			return errors.Wrap(err, "设置原创声明失败")
		}
		slog.Info("原创声明设置完成")
	}

	// 设置内容类型声明
	if settings.ContentType != "" {
		if err := setContentType(page, settings.ContentType); err != nil {
			return errors.Wrap(err, "设置内容类型失败")
		}
		slog.Info("内容类型设置完成", "contentType", settings.ContentType)
	}

	// 设置可见范围
	if settings.VisibleScope != "" {
		if err := setVisibleScope(page, settings.VisibleScope); err != nil {
			return errors.Wrap(err, "设置可见范围失败")
		}
		slog.Info("可见范围设置完成", "visibleScope", settings.VisibleScope)
	}

	// 设置允许合拍
	if settings.AllowDuet != nil {
		if err := setCheckbox(page, "允许合拍", *settings.AllowDuet); err != nil {
			return errors.Wrap(err, "设置允许合拍失败")
		}
		slog.Info("允许合拍设置完成", "allowDuet", *settings.AllowDuet)
	}

	// 设置允许复制
	if settings.AllowCopy != nil {
		if err := setCheckbox(page, "允许正文复制", *settings.AllowCopy); err != nil {
			return errors.Wrap(err, "设置允许复制失败")
		}
		slog.Info("允许复制设置完成", "allowCopy", *settings.AllowCopy)
	}

	// 处理定时发布
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

// checkForErrorMessage 检查页面是否有错误提示
func checkForErrorMessage(page browser.Page) (bool, string) {
	// 常见的错误提示元素选择器
	errorSelectors := []string{
		".el-message--error .el-message__content",
		".d-message--error",
		".error-message",
		"div[class*='error']",
	}

	for _, selector := range errorSelectors {
		has, err := page.Has(selector)
		if err != nil || !has {
			continue
		}

		elem, err := page.Element(selector)
		if err != nil {
			continue
		}

		// 检查元素是否可见
		visible, err := elem.IsVisible()
		if err != nil || !visible {
			continue
		}

		// 获取错误消息
		if text, err := elem.Text(); err == nil && text != "" {
			return true, text
		}
	}

	return false, ""
}

// checkForSuccessDialog 检查是否有成功提示弹窗
func checkForSuccessDialog(page browser.Page) (bool, string) {
	// 成功提示的选择器
	successSelectors := []string{
		".el-message--success .el-message__content",
		".d-message--success",
		".success-message",
	}

	for _, selector := range successSelectors {
		has, err := page.Has(selector)
		if err != nil || !has {
			continue
		}

		elem, err := page.Element(selector)
		if err != nil {
			continue
		}

		// 检查元素是否可见
		visible, err := elem.IsVisible()
		if err != nil || !visible {
			continue
		}

		// 获取成功消息
		if text, err := elem.Text(); err == nil && text != "" {
			return true, text
		}
	}

	// 检查是否跳转到了内容管理页面
	url := page.URL()
	if strings.Contains(url, "creator.xiaohongshu.com/creator/post") {
		slog.Info("检测到页面跳转到内容管理页", "url", url)
		return true, "页面已跳转到内容管理"
	}

	return false, ""
}

// checkSubmitButtonState 检查提交按钮状态
func checkSubmitButtonState(page browser.Page) bool {
	has, err := page.Has("div.submit div.d-button-content")
	if err != nil || !has {
		return false
	}

	elem, err := page.Element("div.submit div.d-button-content")
	if err != nil {
		return false
	}

	// 使用 Eval 检查父元素是否被禁用
	result, err := elem.Eval(`() => {
		const parent = this.parentElement;
		if (!parent) return false;

		const className = parent.className || '';
		if (className.includes('disabled') || className.includes('is-disabled')) {
			return true;
		}

		if (parent.hasAttribute('disabled')) {
			return true;
		}

		return false;
	}`)

	if err != nil {
		return false
	}

	if disabled, ok := result.(bool); ok {
		return disabled
	}

	return false
}

// 检查标题是否超过最大长度
func checkTitleMaxLength(page browser.Page) error {
	has, err := page.Has(`div.title-container div.max_suffix`)
	if err != nil {
		return errors.Wrap(err, "检查标题长度元素失败")
	}

	// 元素不存在，说明标题没超长
	if !has {
		return nil
	}

	// 元素存在，说明标题超长
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

	// 元素不存在，说明正文没超长
	if !has {
		return nil
	}

	// 元素存在，说明正文超长
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
	// 方式1：尝试 div.ql-editor
	elem, err := page.Element("div.ql-editor")
	if err == nil {
		slog.Info("找到内容元素：div.ql-editor")
		return elem, true
	}

	// 方式2：通过 placeholder 查找
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

	// 等待内容元素稳定后再操作
	_ = contentElem.WaitStable(500 * time.Millisecond)

	// 按下箭头键移动到底部
	for i := 0; i < 20; i++ {
		contentElem.Press("ArrowDown")
		// short UI delay between key presses (no WaitForTimeout on Page interface)
		time.Sleep(10 * time.Millisecond)
	}

	// 按两次回车换行
	contentElem.Press("Enter")
	contentElem.Press("Enter")

	// 等待光标就绪
	_ = contentElem.WaitStable(500 * time.Millisecond)

	for _, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		inputTag(page, contentElem, tag)
	}
}

func inputTag(page browser.Page, contentElem browser.Element, tag string) {
	contentElem.Input("#")
	// 等待标签联想容器出现
	_ = page.WaitForSelector("#creator-editor-topic-container", 2*time.Second)

	for _, char := range tag {
		contentElem.Input(string(char))
		// short character typing delay (no WaitForTimeout on Page interface)
		time.Sleep(50 * time.Millisecond)
	}

	// 等待标签联想容器出现（输入完成后再次确认）
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
			// 如果没有找到联想选项，输入空格结束
			contentElem.Input(" ")
		}
	} else {
		slog.Warn("未找到标签联想下拉框，直接输入空格", "tag", tag)
		// 如果没有找到下拉框，输入空格结束
		contentElem.Input(" ")
	}

	// 等待标签处理完成（等待联想容器消失，表示标签已选定）
	_ = page.WaitHidden("#creator-editor-topic-container")
}

func findTextboxByPlaceholder(page browser.Page) (browser.Element, error) {
	elements, err := page.Elements("p")
	if err != nil || elements == nil {
		return nil, errors.New("no p elements found")
	}

	// 查找包含指定placeholder的元素
	placeholderElem := findPlaceholderElement(elements, "输入正文描述")
	if placeholderElem == nil {
		return nil, errors.New("no placeholder element found")
	}

	// 向上查找textbox父元素
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
	// 使用 JavaScript 向上查找 role="textbox" 的父元素，并设置临时标记
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

	// 通过临时属性获取元素
	textbox, err := page.Element("[data-temp-textbox='true']")
	if err == nil {
		// 清除临时属性
		textbox.Eval(`() => { this.removeAttribute('data-temp-textbox'); }`)
		return textbox
	}

	return nil
}

// isElementVisible 检查元素是否可见
func isElementVisible(elem browser.Element) bool {

	// 检查是否有隐藏样式
	style, err := elem.Attribute("style")
	if err == nil && style != "" {
		if strings.Contains(style, "left: -9999px") ||
			strings.Contains(style, "top: -9999px") ||
			strings.Contains(style, "position: absolute; left: -9999px") ||
			strings.Contains(style, "display: none") ||
			strings.Contains(style, "visibility: hidden") {
			return false
		}
	}

	visible, err := elem.IsVisible()
	if err != nil {
		slog.Warn("无法获取元素可见性", "error", err)
		return true
	}

	return visible
}

// setSchedulePublish 设置定时发布时间
func setSchedulePublish(page browser.Page, t time.Time) error {
	// 1. 点击"定时发布" radio button
	if err := clickScheduleRadio(page); err != nil {
		return err
	}
	// 等待时间选择器出现
	if err := page.WaitForSelector("input.el-input__inner[placeholder='选择日期和时间']", 5*time.Second); err != nil {
		logrus.Warnf("等待时间选择器出现: %v，继续尝试", err)
	}

	// 2. 点击时间选择器打开面板
	if err := clickDateTimePicker(page); err != nil {
		return err
	}
	// 等待日期输入框出现（面板已打开的标志）
	if err := page.WaitForSelector("input.el-input__inner[placeholder='选择日期']", 5*time.Second); err != nil {
		logrus.Warnf("等待日期输入框出现: %v，继续尝试", err)
	}

	// 3. 设置日期和时间
	if err := setDateTime(page, t); err != nil {
		return err
	}
	// 等待时间输入处理完成（确定按钮可见）
	if err := page.WaitForSelector("button.el-picker-panel__link-btn", 3*time.Second); err != nil {
		logrus.Warnf("等待确定按钮: %v，继续尝试", err)
	}

	// 4. 点击确定按钮
	if err := clickConfirmButton(page); err != nil {
		return err
	}
	// 等待面板关闭（日期选择器消失）
	if err := page.WaitHidden("input.el-input__inner[placeholder='选择日期']"); err != nil {
		logrus.Warnf("等待日期面板关闭: %v，继续尝试", err)
	}

	return nil
}

// clickScheduleRadio 点击定时发布 radio
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

// clickDateTimePicker 点击时间选择器
func clickDateTimePicker(page browser.Page) error {
	// 查找日期时间选择器输入框
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

// setDateTime 设置日期和时间
func setDateTime(page browser.Page, t time.Time) error {
	dateStr := t.Format("2006-01-02")
	timeStr := t.Format("15:04")

	// 设置日期
	dateInput, err := page.Element("input.el-input__inner[placeholder='选择日期']")
	if err != nil {
		return errors.Wrap(err, "查找日期输入框失败")
	}
	// 先点击聚焦，然后选择全部文本并输入
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

	// 等待时间输入框可用（日期确认后出现）
	if err := page.WaitForSelector("input.el-input__inner[placeholder='选择时间']", 3*time.Second); err != nil {
		logrus.Warnf("等待时间输入框: %v，继续尝试", err)
	}

	// 设置时间
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

// clickConfirmButton 点击确定按钮
func clickConfirmButton(page browser.Page) error {
	// 查找日期选择器弹窗中的确定按钮
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
	// 1. 查找地点输入框
	locationInput, err := page.Element(".address-box input.d-text")
	if err != nil {
		return errors.Wrap(err, "查找地点输入框失败")
	}

	// 2. 点击并输入地点关键词
	if err := locationInput.Click(); err != nil {
		return errors.Wrap(err, "点击地点输入框失败")
	}
	// 等待输入框获得焦点并就绪
	_ = page.WaitIdle()

	if err := locationInput.Fill(location); err != nil {
		return errors.Wrap(err, "输入地点关键词失败")
	}
	slog.Info("已输入地点关键词", "location", location)

	// 3. 等待下拉列表出现
	if err := page.WaitForSelector(".d-dropdown-wrapper", 5*time.Second); err != nil {
		logrus.Warnf("等待地点下拉列表出现: %v，继续尝试", err)
	}

	// 4. 查找并点击第一个地点选项
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

	// 等待地点选择完成（下拉框消失）
	if err := page.WaitHidden(".d-dropdown-wrapper"); err != nil {
		logrus.Warnf("等待地点下拉框关闭: %v，继续尝试", err)
	}
	return nil
}

// findVisibleLocationDropdown 查找可见的地点下拉列表
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
	// 查找合集选择器
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

		// 点击合集选择器
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找合集选择器失败")
		}

		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击合集选择器失败")
		}
		// 等待合集下拉列表出现
		_ = page.WaitVisible(".collection-drop-down")

		// 在下拉列表中查找并点击合集
		dropdown, err := page.Element(".collection-drop-down")
		if err != nil {
			return errors.Wrap(err, "合集下拉列表未出现")
		}

		// 查找匹配的合集项
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
				// 等待下拉列表关闭
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
	// 查找群聊选择器
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

		// 点击群聊选择器
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找群聊选择器失败")
		}

		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击群聊选择器失败")
		}
		// 等待群聊下拉列表出现
		_ = page.WaitVisible(".d-dropdown")

		// 在下拉列表中查找并点击群聊
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
					// 等待页面响应
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
	// 查找"去声明"按钮
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

			// 可能会弹出确认对话框，需要点击确认
			// 这里简单等待，实际使用中可能需要处理弹窗
			return nil
		}
	}

	return errors.New("未找到去声明按钮")
}

// setContentType 设置内容类型声明
func setContentType(page browser.Page, contentType string) error {
	// 查找内容类型声明选择器
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

		// 点击选择器
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找内容类型选择器失败")
		}

		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击内容类型选择器失败")
		}
		// 等待内容类型下拉列表出现
		_ = page.WaitVisible(".d-dropdown")

		// 在下拉列表中查找并点击选项
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
	// 查找可见范围选择器
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

		// 点击选择器
		selectEl, err := item.Element(".d-select")
		if err != nil {
			return errors.Wrap(err, "查找可见范围选择器失败")
		}

		if err := selectEl.Click(); err != nil {
			return errors.Wrap(err, "点击可见范围选择器失败")
		}
		// 等待可见范围下拉列表出现
		_ = page.WaitVisible(".d-dropdown")

		// 在下拉列表中查找并点击选项
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
	// 查找对应的checkbox
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

		// 查找checkbox
		checkbox, err := item.Element("input[type='checkbox']")
		if err != nil {
			return errors.Wrapf(err, "查找 %s checkbox失败", labelText)
		}

		// 检查当前状态
		isChecked, err := checkbox.Eval(`() => this.checked`)
		if err != nil {
			return errors.Wrap(err, "获取checkbox状态失败")
		}

		currentChecked := false
		if b, ok := isChecked.(bool); ok {
			currentChecked = b
		}

		// 如果状态不匹配，点击切换
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
// markers 列表中的每个元素会被搜索，自动从地点和用户两个选项卡中查找匹配项
func setMarkerTags(page browser.Page, markers []string) error {
	// 查找并点击"添加标记"按钮
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
			// 查找"添加标记"或"修改"按钮
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

	// 点击按钮打开标记对话框
	if err := markerButton.Click(); err != nil {
		return errors.Wrap(err, "点击标记按钮失败")
	}
	// 等待对话框出现（WaitForFunction already waits; no extra sleep needed）

	// 等待对话框出现
	if err := page.WaitForFunction(`() => document.querySelector('div[role="dialog"]') !== null`, 5*time.Second); err != nil {
		return errors.Wrap(err, "标记对话框未出现")
	}

	// 对每个标记进行搜索和选择
	for _, marker := range markers {
		// 先尝试在"地点"选项卡中搜索
		found, err := searchAndSelectInTab(page, "地点", marker)
		if err != nil {
			slog.Warn("在地点选项卡搜索失败", "marker", marker, "error", err)
		}
		if found {
			slog.Info("在地点选项卡找到标记", "marker", marker)
			continue
		}

		// 如果地点中未找到，尝试在"用户"选项卡中搜索
		found, err = searchAndSelectInTab(page, "用户", marker)
		if err != nil {
			slog.Warn("在用户选项卡搜索失败", "marker", marker, "error", err)
		}
		if found {
			slog.Info("在用户选项卡找到标记", "marker", marker)
			continue
		}

		// 如果两个选项卡都未找到
		slog.Warn("未找到标记", "marker", marker)
	}

	// 点击"确定"按钮
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

// searchAndSelectInTab 在指定选项卡（"用户"或"地点"）中搜索并选择标记
// 返回是否找到匹配项
func searchAndSelectInTab(page browser.Page, tabName, keyword string) (bool, error) {
	// 点击选项卡
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

	// 查找搜索框并输入关键词
	searchInput, err := page.Element("div[role=\"dialog\"] input[type=\"text\"]")
	if err != nil {
		return false, errors.Wrap(err, "未找到搜索框")
	}

	// 清空并输入关键词
	if err := searchInput.Click(); err != nil {
		return false, errors.Wrap(err, "点击搜索框失败")
	}

	if err := searchInput.Fill(keyword); err != nil {
		return false, errors.Wrap(err, "输入关键词失败")
	}
	// 等待搜索结果加载
	_ = page.WaitDOMStable(3*time.Second, 0.1)

	// 查找搜索结果列表
	// 检查是否有"没有找到"提示
	hasNoResult, _ := page.Has("div[role=\"dialog\"]:has-text(\"没有找到\")")
	if hasNoResult {
		return false, nil
	}

	// 查找结果列表中包含关键词的第一个项
	// 使用 JavaScript 在对话框内查找匹配的文本元素
	result, err := page.Eval(fmt.Sprintf(`() => {
		const dialog = document.querySelector('div[role="dialog"]');
		if (!dialog) return null;

		// 查找所有可能的结果项（StaticText元素）
		const items = Array.from(dialog.querySelectorAll('div'));

		// 过滤出包含关键词的项
		const matches = items.filter(item => {
			const text = item.textContent;
			return text && text.includes(%s) &&
				   !text.includes('搜索') &&
				   !text.includes('没有找到') &&
				   !text.includes('加载中') &&
				   !text.includes('为你推荐');
		});

		// 返回第一个匹配项的文本（用于验证）
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
