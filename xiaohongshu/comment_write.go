package xiaohongshu

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

// CommentFeedAction 表示 Feed 评论动作
type CommentFeedAction struct {
	page    browser.Page
	polling polling.Module
}

// NewCommentFeedAction 创建 Feed 评论动作
func NewCommentFeedAction(page browser.Page, pollingModule polling.Module) (*CommentFeedAction, error) {
	return &CommentFeedAction{
		page:    page,
		polling: pollingModule,
	}, nil
}

// PostComment 发表评论到 Feed
func (f *CommentFeedAction) PostComment(ctx context.Context, feedID, xsecToken, content string) error {
	timeout, err := f.polling.Delay("wait_60000ms")
	if err != nil {
		return err
	}
	page := f.page.WithTimeout(timeout)

	url := makeFeedDetailURL_browser(feedID, xsecToken)
	logrus.Infof("打开 feed 详情页: %s", url)

	if err := page.Goto(url); err != nil {
		return fmt.Errorf("导航失败: %w", err)
	}

	maxWait, err := f.polling.Timeout()
	if err != nil {
		return err
	}
	checkInterval, err := f.polling.Interval()
	if err != nil {
		return err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasNoteData, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.note &&
			       window.__INITIAL_STATE__.note.noteDetailMap !== undefined;
		}`)
		if err == nil && hasNoteData == true {
			break
		}
		time.Sleep(checkInterval)
	}

	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	if err := checkPageAccessible_browser(page, f.polling); err != nil {
		return err
	}

	var inputElem browser.Element
	inputElem, err = f.findInputFallback(page)

	if inputElem == nil {
		logrus.Warnf("Failed to find comment input box with all selectors")
		return fmt.Errorf("未找到评论输入框，该帖子可能不支持评论或网页端不可访问")
	}

	logrus.Info("滚动到评论输入框...")
	if err := inputElem.ScrollIntoView(); err != nil {
		logrus.Warnf("滚动到输入框失败: %v", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_1000ms"); err != nil {
		return err
	}

	logrus.Info("等待输入框可见...")
	if err := inputElem.WaitVisible(); err != nil {
		logrus.Warnf("等待输入框可见失败: %v", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	logrus.Info("点击输入框...")
	if err := inputElem.Click(); err != nil {
		logrus.Warnf("点击输入框失败: %v，尝试继续", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_1000ms"); err != nil {
		return err
	}

	logrus.Info("清空输入框...")
	_, err = page.Eval(`() => {
		const elem = document.querySelector('#content-textarea');
		if (elem) {
			elem.textContent = '';
			elem.innerText = '';
		}
	}`)
	if err != nil {
		logrus.Warnf("清空输入框失败: %v", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	logrus.Infof("输入评论内容: %s", content)
	_, err = page.Eval(fmt.Sprintf(`() => {
		const elem = document.querySelector('#content-textarea');
		if (elem) {
			elem.textContent = %s;
			elem.innerText = %s;
			elem.focus();
			elem.dispatchEvent(new Event('input', { bubbles: true }));
			return true;
		}
		return false;
	}`, strconv.Quote(content), strconv.Quote(content)))

	if err != nil {
		logrus.Warnf("使用 JS 输入失败，尝试 Input 方法: %v", err)
		if err := inputElem.Input(content); err != nil {
			logrus.Warnf("Input 方法也失败: %v", err)
			return fmt.Errorf("无法输入评论内容: %w", err)
		}
	}

	if err := polling.SleepDelay(f.polling, "wait_1000ms"); err != nil {
		return err
	}

	submitTimeout, err := f.polling.Delay("wait_5000ms")
	if err != nil {
		return err
	}
	submitCtx := page.WithTimeout(submitTimeout)

	var submitButton browser.Element
	logrus.Info("查找提交按钮...")
	submitButton, err = f.findSubmitButtonFallback(submitCtx)

	if submitButton == nil {
		logrus.Warnf("Failed to find submit button with all selectors")
		return fmt.Errorf("未找到提交按钮")
	}

	if err := f.prepareButtonForClick(submitCtx, submitButton); err != nil {
		logrus.Warnf("准备点击失败: %v，尝试继续", err)
	}

	disabled, _ := submitButton.Attribute("disabled")
	if disabled != "" {
		logrus.Warn("提交按钮被禁用，可能内容为空或不符合要求")
		return fmt.Errorf("提交按钮被禁用")
	}

	logrus.Info("点击提交按钮（常规点击，5秒超时）...")
	clickErr := submitButton.Click()
	if clickErr != nil {
		logrus.Warnf("常规点击失败(5秒内): %v，等待2秒后尝试 JS 点击", clickErr)
		if err := polling.SleepDelay(f.polling, "wait_2000ms"); err != nil {
			return err
		}

		clicked, err := page.Eval(`() => {
			const buttons = Array.from(document.querySelectorAll('button'));
			const submitBtn = buttons.find(btn =>
				btn.textContent.includes('发布') ||
				btn.textContent.includes('提交') ||
				btn.className.includes('submit')
			);
			if (submitBtn && !submitBtn.disabled) {
				submitBtn.click();
				return true;
			}
			return false;
		}`)

		if err != nil || clicked != true {
			return fmt.Errorf("所有点击方式都失败: 常规点击=%v, JS点击=%v", clickErr, err)
		}
		logrus.Info("JS 点击成功")
	} else {
		logrus.Info("常规点击成功")
	}

	if err := polling.SleepDelay(f.polling, "wait_1000ms"); err != nil {
		return err
	}

	logrus.Infof("Comment posted successfully to feed: %s", feedID)
	return nil
}

// ReplyToComment 回复指定评论
func (f *CommentFeedAction) ReplyToComment(ctx context.Context, feedID, xsecToken, commentID, userID, content string) error {
	timeout, err := f.polling.Delay("wait_300000ms")
	if err != nil {
		return err
	}
	page := f.page.WithTimeout(timeout)
	url := makeFeedDetailURL_browser(feedID, xsecToken)
	logrus.Infof("打开 feed 详情页进行回复: %s", url)

	if err := page.Goto(url); err != nil {
		return fmt.Errorf("导航失败: %w", err)
	}

	maxWait, err := f.polling.Timeout()
	if err != nil {
		return err
	}
	checkInterval, err := f.polling.Interval()
	if err != nil {
		return err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasNoteData, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.note &&
			       window.__INITIAL_STATE__.note.noteDetailMap !== undefined;
		}`)
		if err == nil && hasNoteData == true {
			break
		}
		time.Sleep(checkInterval)
	}

	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	if err := checkPageAccessible_browser(page, f.polling); err != nil {
		return err
	}

	if err := polling.SleepDelay(f.polling, "wait_2000ms"); err != nil {
		return err
	}

	commentEl, err := findCommentElement(page, f.polling, commentID, userID)
	if err != nil {
		return fmt.Errorf("无法找到评论: %w", err)
	}

	logrus.Info("滚动到评论位置...")
	if err := commentEl.ScrollIntoView(); err != nil {
		return fmt.Errorf("滚动到评论失败: %w", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_1000ms"); err != nil {
		return err
	}

	logrus.Info("准备点击回复按钮")

	replyBtn, err := commentEl.Element(".right .interactions .reply")
	if err != nil {
		return fmt.Errorf("无法找到回复按钮: %w", err)
	}

	if err := replyBtn.Click(); err != nil {
		return fmt.Errorf("点击回复按钮失败: %w", err)
	}

	if err := polling.SleepDelay(f.polling, "wait_1000ms"); err != nil {
		return err
	}

	inputEl, err := page.Element("div.input-box div.content-edit p.content-input")
	if err != nil {
		return fmt.Errorf("无法找到回复输入框: %w", err)
	}

	if err := inputEl.Input(content); err != nil {
		return fmt.Errorf("输入回复内容失败: %w", err)
	}

	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	submitBtn, err := page.Element("div.bottom button.submit")
	if err != nil {
		return fmt.Errorf("无法找到提交按钮: %w", err)
	}

	if err := submitBtn.Click(); err != nil {
		return fmt.Errorf("点击提交按钮失败: %w", err)
	}

	if err := polling.SleepDelay(f.polling, "wait_2000ms"); err != nil {
		return err
	}
	logrus.Infof("回复评论成功")
	return nil
}

// findInputFallback 传统方式查找输入框
func (f *CommentFeedAction) findInputFallback(page browser.Page) (browser.Element, error) {
	selectors := []string{
		"#content-textarea",
		"p.content-input[contenteditable]",
		"[contenteditable='true']#content-textarea",
		"p[contenteditable='true']",
		"textarea",
	}

	for _, sel := range selectors {
		timeout, err := f.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			logrus.Infof("找到输入框（传统方式）: %s", sel)
			return elem, nil
		}
	}

	return nil, fmt.Errorf("所有传统选择器都失败")
}

// findSubmitButtonFallback 传统方式查找提交按钮
func (f *CommentFeedAction) findSubmitButtonFallback(page browser.Page) (browser.Element, error) {
	selectors := []string{
		"button.submit",
		"div.bottom button.submit",
		"button[class*='submit']",
		"div.bottom button",
	}

	for _, sel := range selectors {
		timeout, err := f.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			logrus.Infof("找到提交按钮（传统方式）: %s", sel)
			return elem, nil
		}
	}

	buttons, err := page.Elements("button")
	if err == nil {
		for _, btn := range buttons {
			text, _ := btn.Text()
			if text == "发布" || text == "提交" {
				logrus.Info("通过文本找到提交按钮（传统方式）")
				return btn, nil
			}
		}
	}

	return nil, fmt.Errorf("所有传统选择器都失败")
}

// makeFeedDetailURL_browser 构造 Feed 详情页 URL (browser 版本)
func makeFeedDetailURL_browser(feedID, xsecToken string) string {
	return fmt.Sprintf("https://www.xiaohongshu.com/explore/%s?xsec_token=%s&xsec_source=pc_feed", feedID, xsecToken)
}

// prepareButtonForClick 准备点击提交按钮：移除遮挡层、优化滚动、等待稳定
func (f *CommentFeedAction) prepareButtonForClick(page browser.Page, button browser.Element) error {
	logrus.Info("准备点击：移除遮挡层、优化滚动、等待稳定...")

	f.dismissOverlays(page)

	logrus.Info("滚动到提交按钮（视口中部）...")
	if err := button.ScrollIntoView(); err != nil {
		logrus.Warnf("滚动到按钮失败: %v", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_300ms"); err != nil {
		return err
	}

	_, _ = page.Eval(`() => {
		const vh = window.innerHeight;
		window.scrollBy(0, -vh * 0.3);
	}`)
	if err := polling.SleepDelay(f.polling, "wait_300ms"); err != nil {
		return err
	}

	logrus.Info("等待提交按钮可见...")
	if err := button.WaitVisible(); err != nil {
		logrus.Warnf("等待按钮可见失败: %v", err)
	}

	logrus.Info("等待按钮位置稳定...")
	waitStable, err := f.polling.Delay("wait_300ms")
	if err != nil {
		return err
	}
	if err := button.WaitStable(waitStable); err != nil {
		logrus.Warnf("等待按钮稳定失败: %v，尝试继续", err)
	}

	if err := polling.SleepDelay(f.polling, "wait_200ms"); err != nil {
		return err
	}
	return nil
}

// dismissOverlays 移除常见的遮挡层（下载App、隐私弹窗等）
func (f *CommentFeedAction) dismissOverlays(page browser.Page) {
	logrus.Info("尝试移除遮挡层...")

	downloadBarSelectors := []string{
		"text=下载App",
		"text=打开App",
		".download-bar",
		".app-download-bar",
	}

	for _, selector := range downloadBarSelectors {
		timeout, err := f.polling.Delay("wait_1000ms")
		if err != nil {
			return
		}
		has, err := page.WithTimeout(timeout).Has(selector)
		if err == nil && has {
			logrus.Infof("检测到下载提示: %s，尝试关闭", selector)

			closeSelectors := []string{
				selector + " button:has-text('×')",
				selector + " button[aria-label='关闭']",
				selector + " .close-icon",
				selector + " .close-button",
			}

			closed := false
			for _, closeSelector := range closeSelectors {
				timeout, err := f.polling.Delay("wait_500ms")
				if err != nil {
					return
				}
				if err := page.WithTimeout(timeout).Click(closeSelector); err == nil {
					logrus.Info("成功关闭下载提示")
					closed = true
					break
				}
			}

			if !closed {
				_, _ = page.Eval(fmt.Sprintf(`() => {
					const el = document.querySelector('%s');
					if (el) el.style.display = 'none';
				}`, selector))
				logrus.Info("强制隐藏下载提示")
			}
			break
		}
	}

	privacySelectors := []string{
		"text=隐私",
		"text=我知道了",
		"text=同意",
		".privacy-popup",
		".cookie-consent",
	}

	for _, selector := range privacySelectors {
		timeout, err := f.polling.Delay("wait_1000ms")
		if err != nil {
			return
		}
		has, err := page.WithTimeout(timeout).Has(selector)
		if err == nil && has {
			logrus.Infof("检测到隐私弹窗: %s，尝试关闭", selector)

			agreeButtons := []string{
				selector + " button:has-text('同意')",
				selector + " button:has-text('我知道了')",
				selector + " button:has-text('确定')",
			}

			for _, btnSelector := range agreeButtons {
				timeout, err := f.polling.Delay("wait_500ms")
				if err != nil {
					return
				}
				if err := page.WithTimeout(timeout).Click(btnSelector); err == nil {
					logrus.Info("成功关闭隐私弹窗")
					break
				}
			}
			break
		}
	}

	_ = polling.SleepDelay(f.polling, "wait_300ms")
}
