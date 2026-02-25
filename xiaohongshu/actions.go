package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"
	myerrors "github.com/vmxmy/xiaohongshu-mcp/errors"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

// ===== Like / Favorite =====

// ActionResult 通用动作响应（点赞/收藏等）
type ActionResult struct {
	FeedID  string `json:"feed_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// 选择器常量
const (
	SelectorLikeButton    = ".like-wrapper .like-lottie"
	SelectorCollectButton = ".collect-wrapper .collect-icon"
)

// interactActionType 交互动作类型
type interactActionType string

const (
	actionLike       interactActionType = "点赞"
	actionFavorite   interactActionType = "收藏"
	actionUnlike     interactActionType = "取消点赞"
	actionUnfavorite interactActionType = "取消收藏"
)

type interactAction struct {
	page    browser.Page
	polling polling.Module
}

func newInteractAction(page browser.Page, pollingModule polling.Module) *interactAction {
	return &interactAction{page: page, polling: pollingModule}
}

func (a *interactAction) preparePage(ctx context.Context, actionType interactActionType, feedID, xsecToken string) (browser.Page, error) {
	timeout, err := a.polling.Delay("wait_60000ms")
	if err != nil {
		return nil, err
	}
	page := a.page.WithContext(ctx).WithTimeout(timeout)
	url := makeFeedDetailURL(feedID, xsecToken)
	slog.Info("Opening feed detail page", "actionType", actionType, "url", url)

	if err := page.Goto(url); err != nil {
		slog.Warn("failed to navigate", "url", url, "error", err)
	}
	waitDOMStable, err := a.polling.Delay("wait_5000ms")
	if err != nil {
		return nil, err
	}
	if err := page.WaitDOMStable(waitDOMStable, 0.95); err != nil {
		slog.Warn("WaitDOMStable failed", "error", err)
	}
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return nil, err
	}

	if err := a.waitForInitialState(page); err != nil {
		return nil, err
	}

	return page, nil
}

// waitForInitialState 等待页面 __INITIAL_STATE__ 数据就绪
func (a *interactAction) waitForInitialState(page browser.Page) error {
	maxRetries := a.polling.MaxRetries
	if maxRetries <= 0 {
		return errors.New("polling max_retries missing")
	}
	for i := 0; i < maxRetries; i++ {
		result, err := page.Eval(`() => {
			return !!(window.__INITIAL_STATE__ &&
				window.__INITIAL_STATE__.note &&
				window.__INITIAL_STATE__.note.noteDetailMap &&
				Object.keys(window.__INITIAL_STATE__.note.noteDetailMap).length > 0);
		}`)
		if err != nil {
			slog.Warn("Eval error when waiting for __INITIAL_STATE__", "error", err)
			if err := polling.SleepDelay(a.polling, "wait_1000ms"); err != nil {
				return err
			}
			continue
		}

		if boolResult, ok := result.(bool); ok && boolResult {
			slog.Info("__INITIAL_STATE__ 数据就绪")
			return nil
		}

		slog.Info("等待 __INITIAL_STATE__ 就绪...", "attempt", i+1, "maxRetries", maxRetries)
		if err := polling.SleepDelay(a.polling, "wait_1000ms"); err != nil {
			return err
		}
	}
	slog.Warn("__INITIAL_STATE__ 等待超时，继续尝试操作")
	return nil
}

func (a *interactAction) performClick(page browser.Page, selector string) {
	if err := page.Click(selector); err != nil {
		slog.Warn("click selector failed", "selector", selector, "error", err)
	}
}

// LikeAction 负责处理点赞相关交互
type LikeAction struct {
	*interactAction
}

func NewLikeAction(page browser.Page, pollingModule polling.Module) *LikeAction {
	return &LikeAction{interactAction: newInteractAction(page, pollingModule)}
}

// Like 点赞指定笔记，如果已点赞则直接返回
func (a *LikeAction) Like(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, true)
}

// Unlike 取消点赞指定笔记，如果未点赞则直接返回
func (a *LikeAction) Unlike(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, false)
}

func (a *LikeAction) perform(ctx context.Context, feedID, xsecToken string, targetLiked bool) error {
	actionType := actionLike
	if !targetLiked {
		actionType = actionUnlike
	}

	page, err := a.preparePage(ctx, actionType, feedID, xsecToken)
	if err != nil {
		return err
	}

	liked, _, err := a.getInteractState(page, feedID)
	if err != nil {
		slog.Warn("failed to read interact state", "error", err)
		return a.toggleLike(page, feedID, targetLiked, actionType)
	}

	if targetLiked && liked {
		slog.Info("feed already liked, skip clicking", "feedID", feedID)
		return nil
	}
	if !targetLiked && !liked {
		slog.Info("feed not liked yet, skip clicking", "feedID", feedID)
		return nil
	}

	return a.toggleLike(page, feedID, targetLiked, actionType)
}

func (a *LikeAction) toggleLike(page browser.Page, feedID string, targetLiked bool, actionType interactActionType) error {
	a.performClick(page, SelectorLikeButton)
	if err := polling.SleepDelay(a.polling, "wait_3000ms"); err != nil {
		return err
	}

	liked, _, err := a.getInteractState(page, feedID)
	if err != nil {
		slog.Warn("验证状态失败", "actionType", actionType, "error", err)
		return nil
	}
	if liked == targetLiked {
		slog.Info("feed 操作成功", "feedID", feedID, "actionType", actionType)
		return nil
	}

	slog.Warn("feed 操作可能未成功，状态未变化，尝试再次点击", "feedID", feedID, "actionType", actionType)
	a.performClick(page, SelectorLikeButton)
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return err
	}

	liked, _, err = a.getInteractState(page, feedID)
	if err != nil {
		slog.Warn("第二次验证状态失败", "actionType", actionType, "error", err)
		return nil
	}
	if liked == targetLiked {
		slog.Info("feed 第二次点击操作成功", "feedID", feedID, "actionType", actionType)
		return nil
	}

	return nil
}

// FavoriteAction 负责处理收藏相关交互
type FavoriteAction struct {
	*interactAction
}

func NewFavoriteAction(page browser.Page, pollingModule polling.Module) *FavoriteAction {
	return &FavoriteAction{interactAction: newInteractAction(page, pollingModule)}
}

// Favorite 收藏指定笔记，如果已收藏则直接返回
func (a *FavoriteAction) Favorite(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, true)
}

// Unfavorite 取消收藏指定笔记，如果未收藏则直接返回
func (a *FavoriteAction) Unfavorite(ctx context.Context, feedID, xsecToken string) error {
	return a.perform(ctx, feedID, xsecToken, false)
}

func (a *FavoriteAction) perform(ctx context.Context, feedID, xsecToken string, targetCollected bool) error {
	actionType := actionFavorite
	if !targetCollected {
		actionType = actionUnfavorite
	}

	page, err := a.preparePage(ctx, actionType, feedID, xsecToken)
	if err != nil {
		return err
	}

	_, collected, err := a.getInteractState(page, feedID)
	if err != nil {
		slog.Warn("failed to read interact state", "error", err)
		return a.toggleFavorite(page, feedID, targetCollected, actionType)
	}

	if targetCollected && collected {
		slog.Info("feed already favorited, skip clicking", "feedID", feedID)
		return nil
	}
	if !targetCollected && !collected {
		slog.Info("feed not favorited yet, skip clicking", "feedID", feedID)
		return nil
	}

	return a.toggleFavorite(page, feedID, targetCollected, actionType)
}

func (a *FavoriteAction) toggleFavorite(page browser.Page, feedID string, targetCollected bool, actionType interactActionType) error {
	a.performClick(page, SelectorCollectButton)
	if err := polling.SleepDelay(a.polling, "wait_3000ms"); err != nil {
		return err
	}

	_, collected, err := a.getInteractState(page, feedID)
	if err != nil {
		slog.Warn("验证状态失败", "actionType", actionType, "error", err)
		return nil
	}
	if collected == targetCollected {
		slog.Info("feed 操作成功", "feedID", feedID, "actionType", actionType)
		return nil
	}

	slog.Warn("feed 操作可能未成功，状态未变化，尝试再次点击", "feedID", feedID, "actionType", actionType)
	a.performClick(page, SelectorCollectButton)
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return err
	}

	_, collected, err = a.getInteractState(page, feedID)
	if err != nil {
		slog.Warn("第二次验证状态失败", "actionType", actionType, "error", err)
		return nil
	}
	if collected == targetCollected {
		slog.Info("feed 第二次点击操作成功", "feedID", feedID, "actionType", actionType)
		return nil
	}

	return nil
}

// getInteractState 从页面读取笔记的点赞/收藏状态（优先用DOM，fallback到__INITIAL_STATE__）
func (a *interactAction) getInteractState(page browser.Page, feedID string) (liked bool, collected bool, err error) {
	result, evalErr := page.Eval(`() => {
		const likeBtn = document.querySelector('.like-wrapper .like-lottie, .like-wrapper');
		const collectBtn = document.querySelector('.collect-wrapper .collect-icon, .collect-wrapper');

		let liked = false;
		let collected = false;

		if (likeBtn) {
			liked = likeBtn.classList.contains('active') ||
				likeBtn.classList.contains('liked') ||
				likeBtn.closest('.like-wrapper')?.classList.contains('active') ||
				likeBtn.getAttribute('data-active') === 'true';
		}

		if (collectBtn) {
			collected = collectBtn.classList.contains('active') ||
				collectBtn.classList.contains('collected') ||
				collectBtn.closest('.collect-wrapper')?.classList.contains('active') ||
				collectBtn.getAttribute('data-active') === 'true';
		}

		if (!liked && !collected) {
			if (window.__INITIAL_STATE__ &&
				window.__INITIAL_STATE__.note &&
				window.__INITIAL_STATE__.note.noteDetailMap) {
				const noteDetailMap = window.__INITIAL_STATE__.note.noteDetailMap;
				const keys = Object.keys(noteDetailMap);
				if (keys.length > 0) {
					const detail = noteDetailMap[keys[0]];
					if (detail && detail.note && detail.note.interactInfo) {
						liked = detail.note.interactInfo.liked || false;
						collected = detail.note.interactInfo.collected || false;
					}
				}
			}
		}

		return JSON.stringify({liked: liked, collected: collected});
	}`)
	if evalErr != nil {
		return false, false, errors.Wrap(evalErr, "eval interactState failed")
	}

	resultStr, ok := result.(string)
	if !ok || resultStr == "" {
		return false, false, myerrors.ErrNoFeedDetail
	}

	var interactInfo struct {
		Liked     bool `json:"liked"`
		Collected bool `json:"collected"`
	}
	if err := json.Unmarshal([]byte(resultStr), &interactInfo); err != nil {
		return false, false, errors.Wrap(err, "unmarshal interactInfo failed")
	}

	return interactInfo.Liked, interactInfo.Collected, nil
}

// ===== Delete =====

// DeleteAction 删除操作
type DeleteAction struct {
	page    browser.Page
	polling polling.Module
}

// NewDeleteAction 创建删除操作实例
func NewDeleteAction(page browser.Page, pollingModule polling.Module) (*DeleteAction, error) {
	return &DeleteAction{page: page, polling: pollingModule}, nil
}

// DeleteFeed 删除自己的笔记
func (d *DeleteAction) DeleteFeed(ctx context.Context, feedID, xsecToken string) error {
	timeout, err := d.polling.Delay("wait_60000ms")
	if err != nil {
		return err
	}
	page := d.page.WithContext(ctx).WithTimeout(timeout)

	url := makeFeedDetailURL(feedID, xsecToken)
	slog.Info("打开 feed 详情页进行删除", "url", url)

	if err := page.Goto(url); err != nil {
		return fmt.Errorf("导航到详情页失败: %w", err)
	}
	waitStable, err := d.polling.Delay("wait_1000ms")
	if err != nil {
		return err
	}
	if err := page.WaitDOMStable(waitStable, 0.1); err != nil {
		slog.Warn("等待DOM稳定超时，继续执行", "error", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_2000ms"); err != nil {
		return err
	}

	if err := checkPageAccessible(page, d.polling); err != nil {
		return err
	}

	moreBtn, err := d.findMoreButton(page)
	if err != nil {
		if sErr := page.Screenshot("/tmp/delete_debug.png"); sErr == nil {
			slog.Info("调试截图已保存", "path", "/tmp/delete_debug.png")
		}
		if classes, eErr := page.Eval(`() => [...document.querySelectorAll('*')].filter(e => e.className && typeof e.className === 'string').map(e => e.className).filter(c => /more|operate|action|btn|button|ellipsis|dot/i.test(c)).slice(0, 30).join('\n')`); eErr == nil {
			slog.Info("页面相关class名", "classes", classes)
		}
		return fmt.Errorf("未找到更多按钮: %w", err)
	}

	slog.Info("点击更多按钮...")
	if err := moreBtn.Click(); err != nil {
		return fmt.Errorf("点击更多按钮失败: %w", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
		return err
	}

	deleteBtn, err := d.findDeleteButton(page)
	if err != nil {
		return fmt.Errorf("未找到删除按钮: %w", err)
	}

	slog.Info("点击删除按钮...")
	if err := deleteBtn.Click(); err != nil {
		return fmt.Errorf("点击删除按钮失败: %w", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
		return err
	}

	confirmBtn, err := d.findConfirmButton(page)
	if err != nil {
		return fmt.Errorf("未找到确认按钮: %w", err)
	}

	slog.Info("点击确认删除...")
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("点击确认按钮失败: %w", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_2000ms"); err != nil {
		return err
	}

	slog.Info("笔记删除成功", "feedID", feedID)
	return nil
}

// DeleteComment 删除自己的评论
func (d *DeleteAction) DeleteComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	timeout, err := d.polling.Delay("wait_300000ms")
	if err != nil {
		return err
	}
	page := d.page.WithContext(ctx).WithTimeout(timeout)

	url := makeFeedDetailURL(feedID, xsecToken)
	slog.Info("打开 feed 详情页进行删除评论", "url", url)

	if err := page.Goto(url); err != nil {
		return fmt.Errorf("导航到详情页失败: %w", err)
	}
	waitStable, err := d.polling.Delay("wait_1000ms")
	if err != nil {
		return err
	}
	if err := page.WaitDOMStable(waitStable, 0.1); err != nil {
		slog.Warn("等待DOM稳定超时，继续执行", "error", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_2000ms"); err != nil {
		return err
	}

	if err := checkPageAccessible(page, d.polling); err != nil {
		return err
	}
	if err := polling.SleepDelay(d.polling, "wait_2000ms"); err != nil {
		return err
	}

	commentEl, err := findCommentElement(page, d.polling, commentID, userID)
	if err != nil {
		return fmt.Errorf("无法找到评论: %w", err)
	}

	slog.Info("滚动到评论位置...")
	if err := commentEl.ScrollIntoView(); err != nil {
		return fmt.Errorf("滚动到评论位置失败: %w", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
		return err
	}

	moreBtn, err := d.findCommentMoreButton(commentEl)
	if err != nil {
		return fmt.Errorf("未找到评论更多按钮: %w", err)
	}

	slog.Info("点击评论更多按钮...")
	if err := moreBtn.Click(); err != nil {
		return fmt.Errorf("点击更多按钮失败: %w", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
		return err
	}

	deleteBtn, err := d.findDeleteButton(page)
	if err != nil {
		return fmt.Errorf("未找到删除按钮: %w", err)
	}

	slog.Info("点击删除按钮...")
	if err := deleteBtn.Click(); err != nil {
		return fmt.Errorf("点击删除按钮失败: %w", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
		return err
	}

	confirmBtn, err := d.findConfirmButton(page)
	if err != nil {
		slog.Warn("未找到确认按钮，可能已直接删除", "error", err)
		return nil
	}

	slog.Info("点击确认删除...")
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("点击确认按钮失败: %w", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_2000ms"); err != nil {
		return err
	}

	slog.Info("评论删除成功")
	return nil
}

// findMoreButton 查找更多按钮（三个点）
func (d *DeleteAction) findMoreButton(page browser.Page) (browser.Element, error) {
	selectors := []string{
		".info-right-area-more-container",
		".more-button",
		".operate-button",
		"button[aria-label*='更多']",
		"[class*='more-container']",
		"[class*='more']",
	}
	for _, sel := range selectors {
		timeout, err := d.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			slog.Info("找到更多按钮", "selector", sel)
			return elem, nil
		}
	}
	return nil, fmt.Errorf("所有选择器都失败")
}

// findCommentMoreButton 查找评论的更多按钮
func (d *DeleteAction) findCommentMoreButton(commentEl browser.Element) (browser.Element, error) {
	selectors := []string{
		".more",
		"[class*='more']",
		".operate",
	}
	for _, sel := range selectors {
		elem, err := commentEl.Element(sel)
		if err == nil && elem != nil {
			slog.Info("找到评论更多按钮", "selector", sel)
			return elem, nil
		}
	}
	return nil, fmt.Errorf("所有选择器都失败")
}

// findDeleteButton 查找删除按钮
func (d *DeleteAction) findDeleteButton(page browser.Page) (browser.Element, error) {
	selectors := []string{
		"button:has-text('删除')",
		"[class*='delete']",
	}
	for _, sel := range selectors {
		timeout, err := d.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			slog.Info("找到删除按钮", "selector", sel)
			return elem, nil
		}
	}
	buttons, err := page.Elements("button")
	if err == nil {
		for _, btn := range buttons {
			text, _ := btn.Text()
			if text == "删除" {
				slog.Info("通过文本找到删除按钮")
				return btn, nil
			}
		}
	}
	return nil, fmt.Errorf("所有选择器都失败")
}

// findConfirmButton 查找确认按钮
func (d *DeleteAction) findConfirmButton(page browser.Page) (browser.Element, error) {
	selectors := []string{
		"button:has-text('确认')",
		"button:has-text('确定')",
		"[class*='confirm']",
	}
	for _, sel := range selectors {
		timeout, err := d.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			slog.Info("找到确认按钮", "selector", sel)
			return elem, nil
		}
	}
	buttons, err := page.Elements("button")
	if err == nil {
		for _, btn := range buttons {
			text, _ := btn.Text()
			if text == "确认" || text == "确定" {
				slog.Info("通过文本找到确认按钮")
				return btn, nil
			}
		}
	}
	return nil, fmt.Errorf("所有选择器都失败")
}

// ===== Share =====

// ShareAction 分享操作
type ShareAction struct {
	page    browser.Page
	polling polling.Module
}

// NewShareAction 创建分享操作实例
func NewShareAction(page browser.Page, pollingModule polling.Module) (*ShareAction, error) {
	return &ShareAction{page: page, polling: pollingModule}, nil
}

// ShareFeed 分享笔记，获取分享链接
func (s *ShareAction) ShareFeed(ctx context.Context, feedID, xsecToken string) (string, error) {
	timeout, err := s.polling.Delay("wait_60000ms")
	if err != nil {
		return "", err
	}
	page := s.page.WithContext(ctx).WithTimeout(timeout)

	url := makeFeedDetailURL(feedID, xsecToken)
	slog.Info("打开 feed 详情页进行分享", "url", url)

	if err := page.Goto(url); err != nil {
		return "", fmt.Errorf("导航失败: %w", err)
	}
	waitStable, err := s.polling.Delay("wait_1000ms")
	if err != nil {
		return "", err
	}
	if err := page.WaitDOMStable(waitStable, 0.1); err != nil {
		return "", fmt.Errorf("等待 DOM 稳定失败: %w", err)
	}
	if err := polling.SleepDelay(s.polling, "wait_2000ms"); err != nil {
		return "", err
	}

	if err := checkPageAccessible(page, s.polling); err != nil {
		return "", err
	}

	shareBtn, err := s.findShareButton(page)
	if err != nil {
		return "", fmt.Errorf("未找到分享按钮: %w", err)
	}

	slog.Info("滚动到分享按钮...")
	if err := shareBtn.ScrollIntoView(); err != nil {
		slog.Warn("滚动失败", "error", err)
	}
	if err := polling.SleepDelay(s.polling, "wait_500ms"); err != nil {
		return "", err
	}

	slog.Info("点击分享按钮...")
	if err := shareBtn.Click(); err != nil {
		slog.Warn("点击失败，尝试使用 JS 点击", "error", err)
		_, err = page.Eval(`() => {
			const buttons = Array.from(document.querySelectorAll('button, [class*="share"]'));
			const shareBtn = buttons.find(btn =>
				btn.textContent.includes('分享') ||
				btn.className.includes('share')
			);
			if (shareBtn) {
				shareBtn.click();
				return true;
			}
			return false;
		}`)
		if err != nil {
			return "", fmt.Errorf("无法点击分享按钮: %w", err)
		}
	}
	if err := polling.SleepDelay(s.polling, "wait_2000ms"); err != nil {
		return "", err
	}

	copyLinkBtn, err := s.findCopyLinkButton(page)
	if err != nil {
		slog.Warn("未找到复制链接按钮，尝试直接获取链接", "error", err)
		return page.URL(), nil
	}

	slog.Info("点击复制链接按钮...")
	if err := copyLinkBtn.Click(); err != nil {
		slog.Warn("点击复制链接失败", "error", err)
	}
	if err := polling.SleepDelay(s.polling, "wait_1000ms"); err != nil {
		return "", err
	}

	shareLink, err := s.getShareLinkFromClipboard(page)
	if err != nil {
		slog.Warn("从剪贴板获取链接失败，使用当前URL", "error", err)
		return page.URL(), nil
	}

	slog.Info("成功获取分享链接", "shareLink", shareLink)
	return shareLink, nil
}

// findShareButton 查找分享按钮
func (s *ShareAction) findShareButton(page browser.Page) (browser.Element, error) {
	selectors := []string{
		".share-wrapper",
		"[class*='share']",
		".interact-container .share",
		"button[aria-label*='分享']",
	}
	for _, sel := range selectors {
		timeout, err := s.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			slog.Info("找到分享按钮", "selector", sel)
			return elem, nil
		}
	}
	buttons, err := page.Elements("button")
	if err == nil {
		for _, btn := range buttons {
			text, _ := btn.Text()
			if text == "分享" {
				slog.Info("通过文本找到分享按钮")
				return btn, nil
			}
		}
	}
	return nil, fmt.Errorf("所有选择器都失败")
}

// findCopyLinkButton 查找复制链接按钮
func (s *ShareAction) findCopyLinkButton(page browser.Page) (browser.Element, error) {
	selectors := []string{
		"button:has-text('复制链接')",
		"[class*='copy']",
		"button[aria-label*='复制']",
	}
	for _, sel := range selectors {
		timeout, err := s.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			slog.Info("找到复制链接按钮", "selector", sel)
			return elem, nil
		}
	}
	buttons, err := page.Elements("button")
	if err == nil {
		for _, btn := range buttons {
			text, _ := btn.Text()
			if text == "复制链接" || text == "复制" {
				slog.Info("通过文本找到复制链接按钮")
				return btn, nil
			}
		}
	}
	return nil, fmt.Errorf("所有选择器都失败")
}

// getShareLinkFromClipboard 从剪贴板获取分享链接
func (s *ShareAction) getShareLinkFromClipboard(page browser.Page) (string, error) {
	result, err := page.Eval(`async () => {
		try {
			const text = await navigator.clipboard.readText();
			return text;
		} catch (err) {
			return "";
		}
	}`)
	if err != nil {
		return "", fmt.Errorf("读取剪贴板失败: %w", err)
	}
	text, ok := result.(string)
	if !ok || text == "" {
		return "", fmt.Errorf("剪贴板为空")
	}
	return text, nil
}
