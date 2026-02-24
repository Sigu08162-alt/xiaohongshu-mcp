package xiaohongshu

import (
	"context"
	"log/slog"
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

	// Network interception is not yet supported by the browser.Page interface;
	// using UI-based detection as a fallback.

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
	// WaitForNavigation is not yet part of the browser.Page interface;
	// using WaitForFunction to detect URL change or success toast instead.
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
