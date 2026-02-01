package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/errors"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/polling"
)

// ========== 配置常量 ==========
const (
	defaultMaxAttempts     = 500
	stagnantLimit          = 20
	minScrollDelta         = 10
	maxClickPerRound       = 3
	stagnantCheckThreshold = 2 // 达到目标后需要停滞几次才确认
	largeScrollTrigger     = 5 // 停滞多少次后触发大滚动
	buttonClickInterval    = 3 // 每隔多少次尝试点击一次按钮
	finalSprintPushCount   = 15
)

// 延迟时间配置改为配置驱动（见 polling.interaction.delays）

// ========== 数据结构 ==========

type CommentLoadConfig struct {
	ClickMoreReplies    bool
	MaxRepliesThreshold int
	MaxCommentItems     int
	ScrollSpeed         string
}

func DefaultCommentLoadConfig() CommentLoadConfig {
	return CommentLoadConfig{
		ClickMoreReplies:    false,
		MaxRepliesThreshold: 10,
		MaxCommentItems:     0,
		ScrollSpeed:         "normal",
	}
}

type FeedDetailAction struct {
	page    browser.Page
	polling polling.Module
}

func NewFeedDetailAction(page browser.Page, pollingModule polling.Module) (*FeedDetailAction, error) {
	return &FeedDetailAction{page: page, polling: pollingModule}, nil
}

// ========== 主要业务逻辑 ==========

func (f *FeedDetailAction) GetFeedDetail(ctx context.Context, feedID, xsecToken string, loadAllComments bool, config CommentLoadConfig) (*FeedDetailResponse, error) {
	return f.GetFeedDetailWithConfig(ctx, feedID, xsecToken, loadAllComments, config)
}

func (f *FeedDetailAction) GetFeedDetailWithConfig(ctx context.Context, feedID, xsecToken string, loadAllComments bool, config CommentLoadConfig) (*FeedDetailResponse, error) {
	timeout, err := f.polling.Delay("wait_600000ms")
	if err != nil {
		return nil, err
	}
	page := f.page.WithContext(ctx).WithTimeout(timeout)
	url := makeFeedDetailURL(feedID, xsecToken)

	logrus.Infof("打开 feed 详情页: %s", url)
	logrus.Infof("配置: 点击更多=%v, 回复阈值=%d, 最大评论数=%d, 滚动速度=%s",
		config.ClickMoreReplies, config.MaxRepliesThreshold, config.MaxCommentItems, config.ScrollSpeed)

	// 使用retry-go处理页面导航
	retryDelay, err := f.polling.Delay("wait_500ms")
	if err != nil {
		return nil, err
	}
	retryJitter, err := f.polling.Delay("wait_1000ms")
	if err != nil {
		return nil, err
	}
	err = retry.Do(
		func() error {
			if err := page.Goto(url); err != nil {
				return err
			}
			if err := page.WaitLoad(); err != nil {
				return err
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.MaxJitter(retryJitter),
		retry.OnRetry(func(n uint, err error) {
			logrus.Debugf("页面导航重试 #%d: %v", n, err)
		}),
	)
	if err != nil {
		logrus.Errorf("页面导航失败: %v", err)
		return nil, err
	}

	// 等待 __INITIAL_STATE__ 中的笔记数据加载，而不是等待 DOM 稳定
	// Feed 详情页有动态内容（评论加载、实时更新、推荐内容），DOM 可能永远不会稳定
	maxWait, err := f.polling.Timeout()
	if err != nil {
		return nil, err
	}
	checkInterval, err := f.polling.Interval()
	if err != nil {
		return nil, err
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

	// 额外等待500ms确保笔记数据完全加载
	if err := sleepRandom(f.polling, "read_time_min_ms", "read_time_max_ms"); err != nil {
		return nil, err
	}

	if err := checkPageAccessible(page, f.polling); err != nil {
		return nil, err
	}

	if loadAllComments {
		if err := f.loadAllCommentsWithConfig(page, config); err != nil {
			logrus.Warnf("加载全部评论失败: %v", err)
		}
	}

	return f.extractFeedDetail(page, feedID)
}

// ========== 评论加载器 ==========

type commentLoader struct {
	page    browser.Page
	config  CommentLoadConfig
	stats   *loadStats
	state   *loadState
	polling polling.Module
}

type loadStats struct {
	totalClicked int
	totalSkipped int
	attempts     int
}

type loadState struct {
	lastCount      int
	lastScrollTop  int
	stagnantChecks int
}

func (f *FeedDetailAction) loadAllCommentsWithConfig(page browser.Page, config CommentLoadConfig) error {
	loader := &commentLoader{
		page:    page,
		config:  config,
		stats:   &loadStats{},
		state:   &loadState{},
		polling: f.polling,
	}

	return loader.load()
}

func (cl *commentLoader) load() error {
	maxAttempts := cl.calculateMaxAttempts()
	scrollInterval, err := getScrollInterval(cl.polling, cl.config.ScrollSpeed)
	if err != nil {
		return err
	}

	logrus.Info("开始加载评论...")
	if err := scrollToCommentsArea(cl.page, cl.polling); err != nil {
		return err
	}
	if err := sleepRandom(cl.polling, "human_delay_min_ms", "human_delay_max_ms"); err != nil {
		return err
	}

	// 检查是否没有评论
	if cl.checkNoComments() {
		return nil
	}

	for cl.stats.attempts = 0; cl.stats.attempts < maxAttempts; cl.stats.attempts++ {
		logrus.Debugf("=== 尝试 %d/%d ===", cl.stats.attempts+1, maxAttempts)

		if cl.checkComplete() {
			return nil
		}

		if cl.shouldClickButtons() {
			cl.clickButtonsWithRetry()
		}

		currentCount := getCommentCount(cl.page, cl.polling)
		cl.updateState(currentCount)

		if cl.shouldStopAtTarget(currentCount) {
			return nil
		}

		if err := cl.performScroll(); err != nil {
			return err
		}
		cl.handleStagnation()

		time.Sleep(scrollInterval)
	}

	cl.performFinalSprint()
	return nil
}

func (cl *commentLoader) calculateMaxAttempts() int {
	if cl.config.MaxCommentItems > 0 {
		return cl.config.MaxCommentItems * 3
	}
	return defaultMaxAttempts
}

func (cl *commentLoader) checkNoComments() bool {
	if checkNoCommentsArea(cl.page, cl.polling) {
		logrus.Infof("✓ 检测到无评论区域（这是一片荒地），跳过加载")
		return true
	}
	return false
}

func (cl *commentLoader) checkComplete() bool {
	if checkEndContainer(cl.page, cl.polling) {
		currentCount := getCommentCount(cl.page, cl.polling)
		logrus.Infof("✓ 检测到 'THE END' 元素，已滑动到底部")
		if err := sleepRandom(cl.polling, "human_delay_min_ms", "human_delay_max_ms"); err != nil {
			logrus.Warnf("随机等待失败: %v", err)
		}
		logrus.Infof("✓ 加载完成: %d 条评论, 尝试次数: %d, 点击: %d, 跳过: %d",
			currentCount, cl.stats.attempts+1, cl.stats.totalClicked, cl.stats.totalSkipped)
		return true
	}
	return false
}

func (cl *commentLoader) shouldClickButtons() bool {
	return cl.config.ClickMoreReplies && cl.stats.attempts%buttonClickInterval == 0
}

func (cl *commentLoader) clickButtonsWithRetry() {
	clicked, skipped := clickShowMoreButtonsSmart(cl.page, cl.polling, cl.config.MaxRepliesThreshold)
	if clicked > 0 || skipped > 0 {
		cl.stats.totalClicked += clicked
		cl.stats.totalSkipped += skipped
		logrus.Infof("点击'更多': %d 个, 跳过: %d 个, 累计点击: %d, 累计跳过: %d",
			clicked, skipped, cl.stats.totalClicked, cl.stats.totalSkipped)

		if err := sleepRandom(cl.polling, "read_time_min_ms", "read_time_max_ms"); err != nil {
			logrus.Warnf("随机等待失败: %v", err)
		}

		// 重试一轮
		clicked2, skipped2 := clickShowMoreButtonsSmart(cl.page, cl.polling, cl.config.MaxRepliesThreshold)
		if clicked2 > 0 || skipped2 > 0 {
			cl.stats.totalClicked += clicked2
			cl.stats.totalSkipped += skipped2
			logrus.Infof("第 2 轮: 点击 %d, 跳过 %d", clicked2, skipped2)
			if err := sleepRandom(cl.polling, "short_read_min_ms", "short_read_max_ms"); err != nil {
				logrus.Warnf("随机等待失败: %v", err)
			}
		}
	}
}

func (cl *commentLoader) updateState(currentCount int) {
	totalCount := getTotalCommentCount(cl.page, cl.polling)
	logrus.Debugf("当前评论: %d, 目标: %d", currentCount, totalCount)

	if currentCount != cl.state.lastCount {
		logrus.Infof("✓ 评论增加: %d -> %d (+%d)",
			cl.state.lastCount, currentCount, currentCount-cl.state.lastCount)
		cl.state.lastCount = currentCount
		cl.state.stagnantChecks = 0
	} else {
		cl.state.stagnantChecks++
		if cl.state.stagnantChecks%5 == 0 {
			logrus.Debugf("评论停滞 %d 次", cl.state.stagnantChecks)
		}
	}
}

func (cl *commentLoader) shouldStopAtTarget(currentCount int) bool {
	// 如果未设置最大评论数，或者还未达到目标，继续加载
	if cl.config.MaxCommentItems <= 0 {
		return false
	}

	// 如果已达到或超过目标评论数，立即停止
	if currentCount >= cl.config.MaxCommentItems {
		logrus.Infof("✓ 已达到目标评论数: %d/%d, 停止加载",
			currentCount, cl.config.MaxCommentItems)
		return true
	}

	return false
}

func (cl *commentLoader) performScroll() error {
	currentCount := getCommentCount(cl.page, cl.polling)
	if currentCount > 0 {
		_ = scrollToLastComment(cl.page, cl.polling)
		if err := sleepRandom(cl.polling, "post_scroll_min_ms", "post_scroll_max_ms"); err != nil {
			return err
		}
	}

	largeMode := cl.state.stagnantChecks >= largeScrollTrigger
	pushCount := 1
	if largeMode {
		pushCount = 3 + rand.Intn(3)
	}

	_, scrollDelta, currentScrollTop := humanScroll(cl.page, cl.polling, cl.config.ScrollSpeed, largeMode, pushCount)

	if scrollDelta < minScrollDelta || currentScrollTop == cl.state.lastScrollTop {
		cl.state.stagnantChecks++
		if cl.state.stagnantChecks%5 == 0 {
			logrus.Debugf("滚动停滞 %d 次", cl.state.stagnantChecks)
		}
	} else {
		cl.state.stagnantChecks = 0
		cl.state.lastScrollTop = currentScrollTop
	}
	return nil
}

func (cl *commentLoader) handleStagnation() {
	if cl.state.stagnantChecks >= stagnantLimit {
		logrus.Infof("停滞过多，尝试大冲刺...")
		humanScroll(cl.page, cl.polling, cl.config.ScrollSpeed, true, 10)
		cl.state.stagnantChecks = 0

		if checkEndContainer(cl.page, cl.polling) {
			currentCount := getCommentCount(cl.page, cl.polling)
			logrus.Infof("✓ 到达底部，评论数: %d", currentCount)
		}
	}
}

func (cl *commentLoader) performFinalSprint() {
	logrus.Infof("达到最大尝试次数，最后冲刺...")
	humanScroll(cl.page, cl.polling, cl.config.ScrollSpeed, true, finalSprintPushCount)

	currentCount := getCommentCount(cl.page, cl.polling)
	hasEnd := checkEndContainer(cl.page, cl.polling)
	logrus.Infof("✓ 加载结束: %d 条评论, 点击: %d, 跳过: %d, 到达底部: %v",
		currentCount, cl.stats.totalClicked, cl.stats.totalSkipped, hasEnd)
}

// ========== 工具函数 ==========

func sleepRandom(pollingModule polling.Module, minKey, maxKey string) error {
	minDelay, err := pollingModule.Delay(minKey)
	if err != nil {
		return err
	}
	maxDelay, err := pollingModule.Delay(maxKey)
	if err != nil {
		return err
	}
	if maxDelay <= minDelay {
		time.Sleep(minDelay)
		return nil
	}
	gap := maxDelay - minDelay
	delay := minDelay + time.Duration(rand.Int63n(int64(gap)))
	time.Sleep(delay)
	return nil
}

func getScrollInterval(pollingModule polling.Module, speed string) (time.Duration, error) {
	var minKey, maxKey string
	switch speed {
	case "slow":
		minKey, maxKey = "scroll_slow_min_ms", "scroll_slow_max_ms"
	case "fast":
		minKey, maxKey = "scroll_fast_min_ms", "scroll_fast_max_ms"
	default: // normal
		minKey, maxKey = "scroll_normal_min_ms", "scroll_normal_max_ms"
	}
	minDelay, err := pollingModule.Delay(minKey)
	if err != nil {
		return 0, err
	}
	maxDelay, err := pollingModule.Delay(maxKey)
	if err != nil {
		return 0, err
	}
	if maxDelay <= minDelay {
		return minDelay, nil
	}
	gap := maxDelay - minDelay
	return minDelay + time.Duration(rand.Int63n(int64(gap))), nil
}

// ========== 按钮点击 ==========

func clickShowMoreButtonsSmart(page browser.Page, pollingModule polling.Module, maxRepliesThreshold int) (clicked, skipped int) {
	elements, err := page.Elements(".show-more")
	if err != nil {
		return 0, 0
	}

	replyCountRegex := regexp.MustCompile(`展开\s*(\d+)\s*条回复`)
	maxClick := maxClickPerRound + rand.Intn(maxClickPerRound)
	clickedInRound := 0

	for _, el := range elements {
		if clickedInRound >= maxClick {
			break
		}

		if !isElementClickable(el) {
			continue
		}

		text, err := el.Text()
		if err != nil {
			continue
		}

		if shouldSkipButton(text, maxRepliesThreshold, replyCountRegex) {
			skipped++
			continue
		}

		if clickElementWithHumanBehavior(page, pollingModule, el, text) {
			clicked++
			clickedInRound++
		}
	}

	return clicked, skipped
}

func isElementClickable(el browser.Element) bool {
	visible, err := el.IsVisible()
	if err != nil || !visible {
		return false
	}

	box, err := el.BoundingBox()
	return err == nil && box != nil && box.Width > 0 && box.Height > 0
}

func shouldSkipButton(text string, threshold int, regex *regexp.Regexp) bool {
	if threshold <= 0 {
		return false
	}

	matches := regex.FindStringSubmatch(text)
	if len(matches) > 1 {
		if replyCount, err := strconv.Atoi(matches[1]); err == nil && replyCount > threshold {
			logrus.Debugf("跳过'%s'（回复数 %d > 阈值 %d）", text, replyCount, threshold)
			return true
		}
	}
	return false
}

func clickElementWithHumanBehavior(page browser.Page, pollingModule polling.Module, el browser.Element, text string) bool {
	var clickSuccess bool

	retryDelay, err := pollingModule.Delay("wait_100ms")
	if err != nil {
		logrus.Warnf("获取重试延迟失败: %v", err)
		return false
	}
	retryJitter, err := pollingModule.Delay("wait_200ms")
	if err != nil {
		logrus.Warnf("获取重试抖动失败: %v", err)
		return false
	}

	// 使用retry-go进行点击操作重试
	err = retry.Do(
		func() error {
			// 滚动到元素
			if err := el.ScrollIntoView(); err != nil {
				return err
			}

			if err := sleepRandom(pollingModule, "reaction_time_min_ms", "reaction_time_max_ms"); err != nil {
				return err
			}

			// 鼠标悬停
			if box, err := el.BoundingBox(); err == nil && box != nil {
				x := box.X + box.Width/2
				y := box.Y + box.Height/2
				if err := page.Mouse().MoveTo(x, y); err != nil {
					logrus.Debugf("鼠标移动失败: %v", err)
				}
				if err := sleepRandom(pollingModule, "hover_time_min_ms", "hover_time_max_ms"); err != nil {
					return err
				}
			}

			// 点击
			if err := el.Click(); err != nil {
				return err // 返回错误以触发重试
			}

			// 模拟人类阅读时间
			if err := sleepRandom(pollingModule, "read_time_min_ms", "read_time_max_ms"); err != nil {
				return err
			}
			clickSuccess = true
			return nil
		},
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.MaxJitter(retryJitter),
		retry.OnRetry(func(n uint, err error) {
			logrus.Debugf("点击重试 #%d: %s, 错误: %v", n, text, err)
		}),
	)

	if err != nil {
		logrus.Debugf("点击失败 '%s': %v", text, err)
		return false
	}

	if clickSuccess {
		logrus.Debugf("点击了'%s'", text)
	}

	return clickSuccess
}

// ========== 滚动相关 ==========

func humanScroll(page browser.Page, pollingModule polling.Module, speed string, largeMode bool, pushCount int) (bool, int, int) {
	beforeTop := getScrollTop(page, pollingModule)
	viewportHeight := evalIntOrDefault(page, pollingModule, `() => window.innerHeight`, 800)

	baseRatio := getScrollRatio(speed)
	if largeMode {
		baseRatio *= 2.0
	}

	scrolled := false
	actualDelta := 0
	currentScrollTop := beforeTop

	for i := 0; i < max(1, pushCount); i++ {
		scrollDelta := calculateScrollDelta(viewportHeight, baseRatio)
		_, err := page.Eval(`(delta) => { window.scrollBy(0, delta); }`, scrollDelta)
		if err != nil {
			logrus.Warnf("滚动失败: %v", err)
		}

		if err := sleepRandom(pollingModule, "scroll_wait_min_ms", "scroll_wait_max_ms"); err != nil {
			return false, 0, 0
		}

		currentScrollTop = getScrollTop(page, pollingModule)
		deltaThisTime := currentScrollTop - beforeTop
		actualDelta += deltaThisTime

		if deltaThisTime > 5 {
			scrolled = true
		}

		beforeTop = currentScrollTop

		if i < pushCount-1 {
		if err := sleepRandom(pollingModule, "human_delay_min_ms", "human_delay_max_ms"); err != nil {
			return false, 0, 0
		}
		}
	}

	if !scrolled && pushCount > 0 {
		_, err := page.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
		if err != nil {
			logrus.Warnf("滚动到底部失败: %v", err)
		}
		if err := sleepRandom(pollingModule, "post_scroll_min_ms", "post_scroll_max_ms"); err != nil {
			return false, 0, 0
		}
		currentScrollTop = getScrollTop(page, pollingModule)
		actualDelta = currentScrollTop - beforeTop + actualDelta
		scrolled = actualDelta > 5
	}

	if scrolled {
		logrus.Debugf("滚动: %d -> %d (Δ%d, large=%v, push=%d)",
			beforeTop-actualDelta, currentScrollTop, actualDelta, largeMode, pushCount)
	}

	return scrolled, actualDelta, currentScrollTop
}

func getScrollRatio(speed string) float64 {
	switch speed {
	case "slow":
		return 0.5
	case "fast":
		return 0.9
	default: // normal
		return 0.7
	}
}

func calculateScrollDelta(viewportHeight int, baseRatio float64) float64 {
	scrollDelta := float64(viewportHeight) * (baseRatio + rand.Float64()*0.2)
	if scrollDelta < 400 {
		scrollDelta = 400
	}
	return scrollDelta + float64(rand.Intn(100)-50)
}

func scrollToCommentsArea(page browser.Page, pollingModule polling.Module) error {
	logrus.Info("滚动到评论区...")

	// 先定位到评论区
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		return err
	}
	if err := page.WithTimeout(timeout).ScrollIntoView(".comments-container"); err == nil {
		// 等待滚动完成
		if err := polling.SleepDelay(pollingModule, "wait_500ms"); err != nil {
			return err
		}
	}

	// 触发一次小滚动，激活懒加载机制
	smartScroll(page, 100)
	return nil
}

// smartScroll 智能滚动：触发滚轮事件以正确触发懒加载
func smartScroll(page browser.Page, delta float64) {
	_, err := page.Eval(`(delta) => {
		// 查找滚动目标元素
		let targetElement = document.querySelector('.note-scroller')
			|| document.querySelector('.interaction-container')
			|| document.documentElement;

		// 触发滚轮事件（关键！这样才能触发懒加载）
		const wheelEvent = new WheelEvent('wheel', {
			deltaY: delta,
			deltaMode: 0, // 像素模式
			bubbles: true,
			cancelable: true,
			view: window
		});
		targetElement.dispatchEvent(wheelEvent);
	}`, delta)
	if err != nil {
		logrus.Warnf("智能滚动失败: %v", err)
	}
}

func scrollToLastComment(page browser.Page, pollingModule polling.Module) error {
	// 获取所有主评论元素
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		return err
	}
	elements, err := page.WithTimeout(timeout).Elements(".parent-comment")
	if err != nil || len(elements) == 0 {
		return nil
	}
	// 滚动到最后一个评论
	lastComment := elements[len(elements)-1]
	if err := lastComment.ScrollIntoView(); err != nil {
		logrus.Debugf("滚动到最后评论失败: %v", err)
	}
	return nil
}

// ========== DOM 查询 ==========

func getScrollTop(page browser.Page, pollingModule polling.Module) int {
	return evalIntOrDefault(page, pollingModule, `() => {
		return window.pageYOffset || document.documentElement.scrollTop || document.body.scrollTop || 0;
	}`, 0)
}

// evalIntOrDefault 执行 JS 表达式并返回整数，失败时返回默认值
func evalIntOrDefault(page browser.Page, pollingModule polling.Module, expression string, defaultVal int) int {
	var result int

	retryDelay, err := pollingModule.Delay("wait_100ms")
	if err != nil {
		logrus.Warnf("获取重试延迟失败: %v", err)
		return defaultVal
	}
	retryJitter, err := pollingModule.Delay("wait_200ms")
	if err != nil {
		logrus.Warnf("获取重试抖动失败: %v", err)
		return defaultVal
	}

	err = retry.Do(
		func() error {
			evalResult, err := page.Eval(expression)
			if err != nil {
				return err
			}

			switch v := evalResult.(type) {
			case int:
				result = v
			case int64:
				result = int(v)
			case float64:
				result = int(v)
			default:
				return fmt.Errorf("eval 结果不是数字: %T", evalResult)
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.MaxJitter(retryJitter),
		retry.OnRetry(func(n uint, err error) {
			logrus.Debugf("Eval 重试 #%d: %v", n, err)
		}),
	)

	if err != nil {
		logrus.Warnf("Eval 失败，使用默认值 %d: %v", defaultVal, err)
		return defaultVal
	}

	return result
}

func getCommentCount(page browser.Page, pollingModule polling.Module) int {
	var result int

	retryDelay, err := pollingModule.Delay("wait_100ms")
	if err != nil {
		logrus.Warnf("获取重试延迟失败: %v", err)
		return 0
	}
	retryJitter, err := pollingModule.Delay("wait_200ms")
	if err != nil {
		logrus.Warnf("获取重试抖动失败: %v", err)
		return 0
	}
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		logrus.Warnf("获取超时失败: %v", err)
		return 0
	}

	// 使用retry-go来处理可能的DOM查询失败
	err = retry.Do(
		func() error {
			// 使用 Go 获取评论元素
			elements, err := page.WithTimeout(timeout).Elements(".parent-comment")
			if err != nil {
				return err
			}
			result = len(elements)
			return nil
		},
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.MaxJitter(retryJitter),
		retry.OnRetry(func(n uint, err error) {
			logrus.Debugf("获取评论计数重试 #%d: %v", n, err)
		}),
	)

	if err != nil {
		logrus.Warnf("获取评论计数失败: %v", err)
		return 0 // 失败时返回0
	}

	return result
}

func getTotalCommentCount(page browser.Page, pollingModule polling.Module) int {
	var result int

	retryDelay, err := pollingModule.Delay("wait_100ms")
	if err != nil {
		logrus.Warnf("获取重试延迟失败: %v", err)
		return 0
	}
	retryJitter, err := pollingModule.Delay("wait_200ms")
	if err != nil {
		logrus.Warnf("获取重试抖动失败: %v", err)
		return 0
	}
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		logrus.Warnf("获取超时失败: %v", err)
		return 0
	}

	// 使用retry-go来处理可能的DOM查询失败
	err = retry.Do(
		func() error {
			// 使用 Go 获取总评论数元素
			totalEl, err := page.WithTimeout(timeout).Element(".comments-container .total")
			if err != nil {
				return err
			}

			// 获取文本内容
			text, err := totalEl.Text()
			if err != nil {
				return err
			}

			// 使用正则提取数字
			re := regexp.MustCompile(`共(\d+)条评论`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				count, err := strconv.Atoi(matches[1])
				if err != nil {
					return err
				}
				result = count
			} else {
				result = 0
			}

			return nil
		},
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.MaxJitter(retryJitter),
		retry.OnRetry(func(n uint, err error) {
			logrus.Debugf("获取总评论计数重试 #%d: %v", n, err)
		}),
	)

	if err != nil {
		logrus.Warnf("获取总评论计数失败: %v", err)
		return 0 // 失败时返回0
	}

	return result
}

func checkNoCommentsArea(page browser.Page, pollingModule polling.Module) bool {
	// 查找无评论区域
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		return false
	}
	noCommentsEl, err := page.WithTimeout(timeout).Element(".no-comments-text")
	if err != nil {
		// 未找到无评论元素，说明有评论或评论区正常
		return false
	}

	// 获取文本内容
	text, err := noCommentsEl.Text()
	if err != nil {
		return false
	}

	// 检查是否包含"这是一片荒地"等关键词
	text = strings.TrimSpace(text)
	return strings.Contains(text, "这是一片荒地")
}

func checkEndContainer(page browser.Page, pollingModule polling.Module) bool {
	var result bool

	retryDelay, err := pollingModule.Delay("wait_100ms")
	if err != nil {
		logrus.Warnf("获取重试延迟失败: %v", err)
		return false
	}
	retryJitter, err := pollingModule.Delay("wait_200ms")
	if err != nil {
		logrus.Warnf("获取重试抖动失败: %v", err)
		return false
	}
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		logrus.Warnf("获取超时失败: %v", err)
		return false
	}

	// 使用retry-go来处理可能的DOM查询失败
	err = retry.Do(
		func() error {
			// 使用 Go 查找结束容器
			endEl, err := page.WithTimeout(timeout).Element(".end-container")
			if err != nil {
				// 未找到元素，说明未到底部
				result = false
				return nil
			}

			// 获取文本内容
			text, err := endEl.Text()
			if err != nil {
				result = false
				return nil
			}

			// 转换为大写并检查
			textUpper := strings.ToUpper(strings.TrimSpace(text))
			result = strings.Contains(textUpper, "THE END") || strings.Contains(textUpper, "THEEND")
			return nil
		},
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.MaxJitter(retryJitter),
		retry.OnRetry(func(n uint, err error) {
			logrus.Debugf("检查结束容器重试 #%d: %v", n, err)
		}),
	)

	if err != nil {
		logrus.Warnf("检查结束容器失败: %v", err)
		return false // 失败时返回false
	}

	return result
}

// ========== 页面检查 ==========

func checkPageAccessible(page browser.Page, pollingModule polling.Module) error {
	if err := polling.SleepDelay(pollingModule, "wait_500ms"); err != nil {
		return err
	}

	// 查找错误提示容器
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		return err
	}
	wrapperEl, err := page.WithTimeout(timeout).Element(".access-wrapper, .error-wrapper, .not-found-wrapper, .blocked-wrapper")
	if err != nil {
		// 未找到错误容器，说明页面可访问
		return nil
	}

	// 获取文本内容
	text, err := wrapperEl.Text()
	if err != nil {
		// 无法获取文本，假设页面可访问
		return nil
	}

	// 检查关键词
	keywords := []string{
		"当前笔记暂时无法浏览",
		"该内容因违规已被删除",
		"该笔记已被删除",
		"内容不存在",
		"笔记不存在",
		"已失效",
		"私密笔记",
		"仅作者可见",
		"因用户设置，你无法查看",
		"因违规无法查看",
	}

	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			logrus.Warnf("笔记不可访问: %s", kw)
			return errors.NewErrFeedNotAccessible(kw)
		}
	}

	// 如果有文本但不匹配关键词，返回未知错误
	trimmedText := strings.TrimSpace(text)
	if trimmedText != "" {
		logrus.Warnf("笔记不可访问（未知原因）: %s", trimmedText)
		return errors.NewErrFeedNotAccessible(trimmedText)
	}

	return nil
}

// ========== 数据提取 ==========

func (f *FeedDetailAction) extractFeedDetail(page browser.Page, feedID string) (*FeedDetailResponse, error) {
	var result string

	// 使用retry-go来处理可能的DOM查询失败
	retryDelay, err := f.polling.Delay("wait_200ms")
	if err != nil {
		return nil, err
	}
	retryJitter, err := f.polling.Delay("wait_300ms")
	if err != nil {
		return nil, err
	}
	err = retry.Do(
		func() error {
			evalResult, err := page.Eval(`() => {
				if (window.__INITIAL_STATE__ &&
					window.__INITIAL_STATE__.note &&
					window.__INITIAL_STATE__.note.noteDetailMap) {
					const noteDetailMap = window.__INITIAL_STATE__.note.noteDetailMap;
					return JSON.stringify(noteDetailMap);
				}
				return "";
			}`)
			if err != nil {
				return err
			}

			str, ok := evalResult.(string)
			if !ok {
				return fmt.Errorf("eval 结果不是字符串")
			}

			if str != "" {
				result = str
				return nil
			}
			return fmt.Errorf("无法获取初始状态数据")
		},
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.MaxJitter(retryJitter),
		retry.OnRetry(func(n uint, err error) {
			logrus.Debugf("提取Feed详情重试 #%d: %v", n, err)
		}),
	)

	if err != nil {
		logrus.Errorf("提取Feed详情失败: %v", err)
		return nil, fmt.Errorf("提取Feed详情失败: %w", err)
	}

	if result == "" {
		return nil, errors.ErrNoFeedDetail
	}

	var noteDetailMap map[string]struct {
		Note     FeedDetail  `json:"note"`
		Comments CommentList `json:"comments"`
	}

	if err := json.Unmarshal([]byte(result), &noteDetailMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal noteDetailMap: %w", err)
	}

	noteDetail, exists := noteDetailMap[feedID]
	if !exists {
		return nil, fmt.Errorf("feed %s not found in noteDetailMap", feedID)
	}

	return &FeedDetailResponse{
		Note:     noteDetail.Note,
		Comments: noteDetail.Comments,
	}, nil
}

func makeFeedDetailURL(feedID, xsecToken string) string {
	return fmt.Sprintf("https://www.xiaohongshu.com/explore/%s?xsec_token=%s&xsec_source=pc_feed", feedID, xsecToken)
}
