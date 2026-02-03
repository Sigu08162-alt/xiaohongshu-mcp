package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

// DataAction 数据获取操作
type DataAction struct {
	page    browser.Page
	polling polling.Module
}

// NewDataAction 创建数据获取操作实例
func NewDataAction(page browser.Page, pollingModule polling.Module) (*DataAction, error) {
	return &DataAction{page: page, polling: pollingModule}, nil
}

// UserStats 用户统计数据
type UserStats struct {
	// 基础数据
	FollowerCount int `json:"follower_count"` // 粉丝数
	FollowCount   int `json:"follow_count"`   // 关注数
	LikedCount    int `json:"liked_count"`    // 获赞与收藏
	NoteCount     int `json:"note_count"`     // 笔记数
	CollectCount  int `json:"collect_count"`  // 收藏数

	// 创作者中心数据（近7日）
	ExposureCount       int     `json:"exposure_count,omitempty"`        // 曝光数
	ViewCount           int     `json:"view_count,omitempty"`            // 观看数
	CoverClickRate      float64 `json:"cover_click_rate,omitempty"`      // 封面点击率
	VideoCompleteRate   float64 `json:"video_complete_rate,omitempty"`   // 视频完播率
	LikeCount7d         int     `json:"like_count_7d,omitempty"`         // 点赞数（7日）
	CommentCount7d      int     `json:"comment_count_7d,omitempty"`      // 评论数（7日）
	CollectCount7d      int     `json:"collect_count_7d,omitempty"`      // 收藏数（7日）
	ShareCount7d        int     `json:"share_count_7d,omitempty"`        // 分享数（7日）
	NetFollowerGrowth   int     `json:"net_follower_growth,omitempty"`   // 净涨粉
	NewFollowerCount    int     `json:"new_follower_count,omitempty"`    // 新增关注
	UnfollowCount       int     `json:"unfollow_count,omitempty"`        // 取消关注
	ProfileVisitorCount int     `json:"profile_visitor_count,omitempty"` // 主页访客
}

// FanAnalytics 粉丝分析数据
type FanAnalytics struct {
	Overview     FanOverview     `json:"overview"`
	Demographics FanDemographics `json:"demographics"`
	ActiveFans   []ActiveFan     `json:"active_fans"`
}

// FanOverview 粉丝概览
type FanOverview struct {
	TotalFans int `json:"total_fans"` // 总粉丝数
	NewFans   int `json:"new_fans"`   // 新增粉丝数
	LostFans  int `json:"lost_fans"`  // 流失粉丝数
}

// FanDemographics 粉丝画像
type FanDemographics struct {
	Gender    map[string]int `json:"gender"`    // 性别分布 {"male": 59, "female": 41}
	Interests []string       `json:"interests"` // 兴趣分布
}

// ActiveFan 活跃粉丝
type ActiveFan struct {
	Nickname     string `json:"nickname"`     // 昵称
	Interactions int    `json:"interactions"` // 互动次数
}

// ContentAnalytics 内容分析数据
type ContentAnalytics struct {
	Notes []NoteMetrics `json:"notes"`
}

// NoteMetrics 笔记指标
type NoteMetrics struct {
	FeedID          string  `json:"feed_id"`           // 笔记ID
	XsecToken       string  `json:"xsec_token"`        // 访问令牌
	Title           string  `json:"title"`             // 标题
	PublishTime     string  `json:"publish_time"`      // 发布时间
	Exposure        int     `json:"exposure"`          // 曝光数
	Views           int     `json:"views"`             // 观看数
	ClickRate       float64 `json:"click_rate"`        // 点击率
	Likes           int     `json:"likes"`             // 点赞数
	Comments        int     `json:"comments"`          // 评论数
	Collects        int     `json:"collects"`          // 收藏数
	FollowerGrowth  int     `json:"follower_growth"`   // 涨粉数
	Shares          int     `json:"shares"`            // 分享数
	AvgViewDuration string  `json:"avg_view_duration"` // 人均观看时长
	FullScreen      int     `json:"full_screen"`       // 满屏
	Status          string  `json:"status"`            // 状态
}

// FollowerUser 粉丝/关注用户信息
type FollowerUser struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Desc     string `json:"desc"`
}

// GetMyStats 获取当前用户的统计数据
func (d *DataAction) GetMyStats(ctx context.Context) (*UserStats, error) {
	timeout, err := d.polling.Delay("wait_60000ms")
	if err != nil {
		return nil, err
	}
	page := d.page.WithContext(ctx).WithTimeout(timeout)

	// 导航到创作者中心页面（包含更详细的运营数据）
	logrus.Info("导航到创作者中心页面...")
	if err := page.Goto("https://creator.xiaohongshu.com/new/home?source=official"); err != nil {
		return nil, fmt.Errorf("导航失败: %w", err)
	}
	waitStable, err := d.polling.Delay("wait_1000ms")
	if err != nil {
		return nil, err
	}
	if err := page.WaitDOMStable(waitStable, 0.1); err != nil {
		logrus.Warn("等待 DOM 稳定出现问题", "error", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_5000ms"); err != nil {
		return nil, err
	}

	accountBase, err := d.fetchAccountBase(page)
	if err != nil {
		return nil, fmt.Errorf("获取账号数据失败: %w", err)
	}
	personalInfo, err := d.fetchPersonalInfo(page)
	if err != nil {
		return nil, fmt.Errorf("获取个人信息失败: %w", err)
	}
	noteDetail, err := d.fetchNoteDetail(page)
	if err != nil {
		return nil, fmt.Errorf("获取笔记详情失败: %w", err)
	}

	stats := UserStats{
		FollowerCount:       getInt(personalInfo, "fans_count"),
		FollowCount:         getInt(personalInfo, "follow_count"),
		LikedCount:          getInt(personalInfo, "faved_count"),
		NoteCount:           getInt(personalInfo, "note_count"),
		CollectCount:        getInt(personalInfo, "collect_count"),
		ExposureCount:       getNestedInt(accountBase, "thirty", "exposure_count"),
		ViewCount:           getNestedInt(noteDetail, "seven", "view_count"),
		CoverClickRate:      getNestedFloat(accountBase, "thirty", "cover_click_rate"),
		VideoCompleteRate:   getNestedFloat(accountBase, "thirty", "video_complete_rate"),
		LikeCount7d:         getNestedInt(noteDetail, "seven", "like_count"),
		CommentCount7d:      getNestedInt(noteDetail, "seven", "comment_count"),
		CollectCount7d:      getNestedInt(noteDetail, "seven", "collect_count"),
		ShareCount7d:        getNestedInt(noteDetail, "seven", "share_count"),
		NetFollowerGrowth:   getNestedInt(noteDetail, "seven", "rise_fans_count"),
		NewFollowerCount:    getNestedInt(noteDetail, "seven", "new_fans_count"),
		UnfollowCount:       getNestedInt(noteDetail, "seven", "leave_fans_count"),
		ProfileVisitorCount: getNestedInt(noteDetail, "seven", "home_view_count"),
	}

	if stats.FollowerCount == 0 &&
		stats.FollowCount == 0 &&
		stats.LikedCount == 0 &&
		stats.NoteCount == 0 &&
		stats.CollectCount == 0 {
		return nil, fmt.Errorf("获取统计数据为空，可能未登录或接口返回异常")
	}

	logrus.Infof("获取统计数据成功: %+v", stats)
	return &stats, nil
}

type apiFetchResult struct {
	Status  int               `json:"status"`
	Body    json.RawMessage   `json:"body"`
	Headers map[string]string `json:"headers"`
	HasCSRF bool              `json:"has_csrf"`
}

func (d *DataAction) fetchAccountBase(page browser.Page) (map[string]interface{}, error) {
	return d.fetchJSONWithRetry(page, "/api/galaxy/v2/creator/datacenter/account/base")
}

func (d *DataAction) fetchPersonalInfo(page browser.Page) (map[string]interface{}, error) {
	return d.fetchJSONWithRetry(page, "/api/galaxy/creator/home/personal_info")
}

func (d *DataAction) fetchNoteDetail(page browser.Page) (map[string]interface{}, error) {
	return d.fetchJSONWithRetry(page, "/api/galaxy/creator/data/note_detail_new")
}

func (d *DataAction) fetchJSONWithRetry(page browser.Page, path string) (map[string]interface{}, error) {
	timeout, err := d.polling.Timeout()
	if err != nil {
		return nil, err
	}
	interval, err := d.polling.Interval()
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	url := "https://creator.xiaohongshu.com" + path
	var lastErr error

	for time.Now().Before(deadline) {
		data, err := d.fetchJSON(page, url)
		if err == nil {
			return data, nil
		}
		lastErr = err
		time.Sleep(interval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("请求超时")
	}
	return nil, lastErr
}

func (d *DataAction) fetchJSON(page browser.Page, url string) (map[string]interface{}, error) {
	result, err := page.Eval(`async (url) => {
		try {
			const getCookie = (name) => {
				const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
				return match ? decodeURIComponent(match[1]) : '';
			};
			const getStorage = (store, name) => {
				try { return store.getItem(name) || ''; } catch (e) { return ''; }
			};
			const getMeta = (name) => {
				const el = document.querySelector('meta[name="' + name + '"]');
				return el ? (el.getAttribute('content') || '') : '';
			};
			const csrf = getCookie('csrf_token') ||
				getCookie('XSRF-TOKEN') ||
				getStorage(localStorage, 'csrf_token') ||
				getStorage(localStorage, 'XSRF-TOKEN') ||
				getStorage(sessionStorage, 'csrf_token') ||
				getStorage(sessionStorage, 'XSRF-TOKEN') ||
				getMeta('csrf-token') ||
				getMeta('x-csrf-token') ||
				getMeta('xsrf-token');
			const headers = {
				'accept': 'application/json, text/plain, */*',
				'x-requested-with': 'XMLHttpRequest'
			};
			if (csrf) {
				headers['x-csrf-token'] = csrf;
				headers['x-xsrf-token'] = csrf;
			}
			const resp = await fetch(url, {
				credentials: 'include',
				referrer: window.location.href || 'https://creator.xiaohongshu.com/new/home?source=official',
				referrerPolicy: 'strict-origin-when-cross-origin',
				headers
			});
			const text = await resp.text();
			const tracked = {};
			resp.headers.forEach((value, key) => {
				const k = key.toLowerCase();
				if (k === 'content-type' || k === 'x-request-id' || k === 'x-req-id' || k === 'x-b3-traceid' || k === 'x-trace-id') {
					tracked[k] = value;
				}
			});
			return JSON.stringify({ status: resp.status, body: text, headers: tracked, has_csrf: !!csrf });
		} catch (e) {
			return JSON.stringify({ status: 0, body: JSON.stringify({ error: String(e) }), headers: {}, has_csrf: false });
		}
	}`, url)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %w", err)
	}
	raw, ok := result.(string)
	if !ok || raw == "" {
		return nil, fmt.Errorf("接口响应为空")
	}
	var fetchResult apiFetchResult
	if err := json.Unmarshal([]byte(raw), &fetchResult); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if fetchResult.Status != 200 {
		preview := string(fetchResult.Body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		logrus.WithFields(logrus.Fields{
			"url":      url,
			"status":   fetchResult.Status,
			"body":     preview,
			"headers":  fetchResult.Headers,
			"has_csrf": fetchResult.HasCSRF,
		}).Warn("统计接口返回非200状态")
		return nil, fmt.Errorf("接口返回异常状态: %d", fetchResult.Status)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(fetchResult.Body, &payload); err != nil {
		return nil, fmt.Errorf("解析接口JSON失败: %w", err)
	}
	if data, ok := payload["data"].(map[string]interface{}); ok {
		return data, nil
	}
	return payload, nil
}

func getInt(data map[string]interface{}, key string) int {
	return int(asFloat(data, key))
}

func getNestedInt(data map[string]interface{}, key string, child string) int {
	return int(getNestedFloat(data, key, child))
}

func getNestedFloat(data map[string]interface{}, key string, child string) float64 {
	parent, ok := data[key].(map[string]interface{})
	if !ok {
		return 0
	}
	return asFloat(parent, child)
}

func asFloat(data map[string]interface{}, key string) float64 {
	value, ok := data[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if num, err := v.Float64(); err == nil {
			return num
		}
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return parsed
		}
	}
	return 0
}

// GetMyFeeds 获取自己发布的笔记列表
func (d *DataAction) GetMyFeeds(ctx context.Context, limit int, userID string) ([]Feed, error) {
	timeout, err := d.polling.Delay("wait_300000ms")
	if err != nil {
		return nil, err
	}
	page := d.page.WithContext(ctx).WithTimeout(timeout)

	// 通过侧边栏导航到个人主页
	logrus.Info("通过侧边栏导航到个人主页获取笔记...")
	navigate := NewNavigate(page, d.polling)
	if err := navigate.ToProfilePageWithUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("导航到个人主页失败: %w", err)
	}

	// 等待 __INITIAL_STATE__ 中的笔记数据加载，而不是等待 DOM 稳定
	// 个人主页有动态内容（笔记推荐、实时更新），DOM 可能永远不会稳定
	maxWait, err := d.polling.Timeout()
	if err != nil {
		return nil, err
	}
	checkInterval, err := d.polling.Interval()
	if err != nil {
		return nil, err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasNotes, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.user &&
			       window.__INITIAL_STATE__.user.notes !== undefined;
		}`)
		if err == nil && hasNotes == true {
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待500ms确保笔记数据完全加载
	if err := polling.SleepDelay(d.polling, "wait_500ms"); err != nil {
		return nil, err
	}

	// 使用JavaScript提取笔记列表
	feeds := d.extractFeedsFromPage(page, limit)

	logrus.Infof("获取笔记列表成功，共 %d 条", len(feeds))
	return feeds, nil
}

// extractFeedsFromPage 从页面提取笔记列表
func (d *DataAction) extractFeedsFromPage(page browser.Page, limit int) []Feed {
	var feeds []Feed
	lastCount := 0
	stagnantChecks := 0
	maxAttempts := 20
	titleMap := map[string]string{}
	coverMap := map[string]string{}

	if currentURL, err := page.Eval(`() => location.href`); err == nil {
		logrus.WithField("url", currentURL).Info("开始提取个人主页笔记列表")
	}
	if debugSamples, err := page.Eval(`() => {
		const links = Array.from(document.querySelectorAll('a[href*="xsec_token"]'));
		const samples = [];
		for (const link of links.slice(0, 5)) {
			const card = link.closest('article, section, li, div');
			const dataset = card ? Object.assign({}, card.dataset) : {};
			const attrs = card ? {
				'data-note-id': card.getAttribute('data-note-id'),
				'data-noteid': card.getAttribute('data-noteid'),
				'data-id': card.getAttribute('data-id')
			} : {};
			const text = card ? card.innerText.split('\\n').map(t => t.trim()).filter(Boolean).slice(0, 6) : [];
			samples.push({
				href: link.getAttribute('href') || '',
				linkText: (link.textContent || '').trim(),
				cardTag: card ? card.tagName.toLowerCase() : '',
				cardClass: card ? card.className : '',
				dataset,
				attrs,
				text
			});
		}
		return samples;
	}`); err == nil {
		logrus.WithField("samples", debugSamples).Info("个人主页笔记链接样本")
	}

	stateNotesRaw, err := page.Eval(`() => window.__INITIAL_STATE__?.user?.notes ?? null`)
	if err != nil {
		logrus.WithError(err).Debug("获取 __INITIAL_STATE__.user.notes 失败")
	} else {
		titleMap, coverMap = buildStateNoteMapsFromRaw(stateNotesRaw)
	}

	for attempt := 0; attempt < maxAttempts && len(feeds) < limit; attempt++ {
		// 使用JavaScript提取笔记
		result, err := page.Eval(`(limit) => {
			const notes = [];
			const seen = new Set();

			const normalizeText = (text) => {
				if (!text) return '';
				return text.replace(/\s+/g, ' ').trim();
			};

			const collectCandidate = (list, value) => {
				const candidate = normalizeText(value);
				if (!candidate) return;
				if (list.includes(candidate)) return;
				list.push(candidate);
			};

			const parseNumberText = (text) => {
				const match = normalizeText(text).match(/(\d+(?:\.\d+)?[万亿]?)/);
				return match ? match[1] : '';
			};

			const findMetric = (card, keywords) => {
				if (!card) return '';
				const keywordRegex = new RegExp(keywords.join('|'), 'i');
				const candidates = card.querySelectorAll('[aria-label],[title],span,div,em,i');
				for (const el of candidates) {
					const aria = el.getAttribute('aria-label') || '';
					const title = el.getAttribute('title') || '';
					const text = normalizeText(el.textContent || '');
					if (keywordRegex.test(aria) || keywordRegex.test(title) || keywordRegex.test(text)) {
						const num = parseNumberText(aria || title || text);
						if (num) return num;
					}
				}

				const text = normalizeText(card.textContent || '');
				for (const keyword of keywords) {
					const re = new RegExp('(\\d+(?:\\.\\d+)?[万亿]?)\\s*' + keyword);
					const match = text.match(re);
					if (match) return match[1];
				}
				return '';
			};

			const findPublishTime = (card) => {
				if (!card) return '';
				const timeEl = card.querySelector('[class*="time"],[class*="date"]');
				if (timeEl) {
					const text = normalizeText(timeEl.textContent || '');
					if (text) return text;
				}

				const text = normalizeText(card.textContent || '');
				const match = text.match(/(\d{4}-\d{2}-\d{2}|\d{2}-\d{2}|刚刚|昨天|前天|\d+分钟前|\d+小时前|\d+天前)/);
				return match ? match[1] : '';
			};

			// 查找所有笔记链接 (包含xsec_token的链接)
			document.querySelectorAll('a[href*="xsec_token"]').forEach(link => {
				const href = link.getAttribute('href');
				// 匹配用户主页下的笔记链接
				const match = href.match(/\/user\/profile\/(\w+)\/(\w+)\?/);
				if (match && !seen.has(match[2])) {
					seen.add(match[2]);
					const [_, userId, noteId] = match;

					// 提取标题 - 查找链接内的文本
					const titleCandidates = [];
					collectCandidate(titleCandidates, link.getAttribute('title'));
					collectCandidate(titleCandidates, link.getAttribute('aria-label'));
					const titleEl = link.querySelector('[class*="title"], [data-testid*="title"], span, div');
					if (titleEl) {
						collectCandidate(titleCandidates, titleEl.textContent);
					}
					collectCandidate(titleCandidates, link.textContent);
					const imgEl = link.querySelector('img');
					if (imgEl) {
						collectCandidate(titleCandidates, imgEl.getAttribute('alt'));
					}

					// 提取封面图
					let cover = '';
					if (imgEl) {
						cover = imgEl.src || '';
					}

					const card = link.closest('article, section, li, div');
					const cardTextLines = card
						? card.innerText.split('\n').map(t => t.trim()).filter(Boolean)
						: [];
					if (cardTextLines.length) {
						collectCandidate(titleCandidates, cardTextLines[0]);
					}
					const likedCount = findMetric(card, ['赞', 'like', 'liked']);
					const commentCount = findMetric(card, ['评论', 'comment']);
					const collectedCount = findMetric(card, ['收藏', 'collect', 'favorite']);
					const publishTime = findPublishTime(card);

					// 提取xsec_token
					const tokenMatch = href.match(/xsec_token=([^&]+)/);
					const xsecToken = tokenMatch ? tokenMatch[1] : '';

					if (notes.length < limit) {
						notes.push({
							id: noteId,
							user_id: userId,
							title_candidates: titleCandidates,
							cover: cover,
							xsec_token: xsecToken,
							liked_count: likedCount,
							comment_count: commentCount,
							collected_count: collectedCount,
							publish_time: publishTime,
							card_text_lines: cardTextLines
						});
					}
				}
			});

			return JSON.stringify(notes);
		}`, limit)

		if err != nil {
			logrus.WithError(err).Error("执行 JavaScript 失败")
			break
		}

		resultStr, ok := result.(string)
		if !ok {
			logrus.Error("JavaScript 返回类型错误")
			break
		}

		parsedFeeds, err := parseFeedsJSON(resultStr)
		if err != nil {
			logrus.WithError(err).Error("解析笔记数据失败")
			break
		}
		feeds = applyStateNoteMaps(parsedFeeds, titleMap, coverMap)

		currentCount := len(feeds)
		if currentCount != lastCount {
			logrus.Infof("加载笔记: %d -> %d", lastCount, currentCount)
			lastCount = currentCount
			stagnantChecks = 0
		} else {
			stagnantChecks++
			if stagnantChecks >= 3 {
				logrus.Info("笔记列表停滞，停止加载")
				break
			}
		}

		if len(feeds) >= limit {
			break
		}

		// 滚动到底部加载更多
		page.Eval(`() => { window.scrollBy(0, window.innerHeight); }`)
		if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
			break
		}
	}

	// 限制返回数量
	if len(feeds) > limit {
		feeds = feeds[:limit]
	}

	return feeds
}

func parseFeedsJSON(resultStr string) ([]Feed, error) {
	var extractedNotes []struct {
		ID              string   `json:"id"`
		UserID          string   `json:"user_id"`
		Title           string   `json:"title"`
		TitleCandidates []string `json:"title_candidates"`
		CardTextLines   []string `json:"card_text_lines"`
		Cover           string   `json:"cover"`
		XsecToken       string   `json:"xsec_token"`
		LikedCount      string   `json:"liked_count"`
		CommentCount    string   `json:"comment_count"`
		CollectedCount  string   `json:"collected_count"`
		PublishTime     string   `json:"publish_time"`
	}

	if err := json.Unmarshal([]byte(resultStr), &extractedNotes); err != nil {
		return nil, err
	}

	feeds := make([]Feed, 0, len(extractedNotes))
	for _, note := range extractedNotes {
		feed := Feed{
			ID:        note.ID,
			XsecToken: note.XsecToken,
		}
		displayTitle := pickTitle(note.TitleCandidates)
		if displayTitle == "" {
			displayTitle = pickTitleFromLines(note.CardTextLines)
		}
		if displayTitle == "" {
			displayTitle = note.Title
		}
		feed.NoteCard.DisplayTitle = displayTitle
		feed.NoteCard.Cover.URLDefault = note.Cover
		feed.NoteCard.User.UserID = note.UserID
		feed.NoteCard.InteractInfo.LikedCount = note.LikedCount
		feed.NoteCard.InteractInfo.CommentCount = note.CommentCount
		feed.NoteCard.InteractInfo.CollectedCount = note.CollectedCount
		feed.NoteCard.PublishTime = note.PublishTime
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

func pickTitle(candidates []string) string {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func pickTitleFromLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	first := strings.TrimSpace(lines[0])
	if first == "" {
		return ""
	}
	return first
}

type stateNote struct {
	ID    string
	Title string
	Cover string
}

func normalizeStateNotesRaw(raw any) []stateNote {
	switch value := raw.(type) {
	case nil:
		return nil
	case []any:
		return normalizeStateNotesSlice(value)
	case map[string]any:
		if list, ok := value["_value"].([]any); ok {
			return normalizeStateNotesSlice(list)
		}
		if list, ok := value["list"].([]any); ok {
			return normalizeStateNotesSlice(list)
		}
		items := make([]any, 0, len(value))
		for _, item := range value {
			items = append(items, item)
		}
		return normalizeStateNotesSlice(items)
	default:
		return nil
	}
}

func normalizeStateNotesSlice(items []any) []stateNote {
	notes := make([]stateNote, 0, len(items))
	for _, item := range items {
		noteMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		note := stateNote{
			ID:    getFirstString(noteMap, []string{"id", "noteId", "note_id"}),
			Title: getFirstString(noteMap, []string{"displayTitle", "title", "noteTitle"}),
			Cover: getCoverURL(noteMap),
		}
		if note.ID == "" {
			continue
		}
		notes = append(notes, note)
	}
	return notes
}

func getFirstString(source map[string]any, keys []string) string {
	for _, key := range keys {
		if value, ok := source[key]; ok {
			if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
				return strings.TrimSpace(str)
			}
		}
	}
	return ""
}

func getCoverURL(source map[string]any) string {
	coverValue, ok := source["cover"]
	if !ok {
		return ""
	}
	coverMap, ok := coverValue.(map[string]any)
	if !ok {
		return ""
	}
	if url := getFirstString(coverMap, []string{"urlDefault"}); url != "" {
		return url
	}
	return getFirstString(coverMap, []string{"url"})
}

func buildStateNoteMapsFromRaw(raw any) (map[string]string, map[string]string) {
	titleMap := map[string]string{}
	coverMap := map[string]string{}
	for _, note := range normalizeStateNotesRaw(raw) {
		if note.Title != "" {
			titleMap[note.ID] = note.Title
		}
		if note.Cover != "" {
			coverMap[note.ID] = note.Cover
		}
	}
	return titleMap, coverMap
}

func applyStateNoteMaps(feeds []Feed, titleMap, coverMap map[string]string) []Feed {
	for i := range feeds {
		if feeds[i].NoteCard.DisplayTitle == "" {
			if title, ok := titleMap[feeds[i].ID]; ok {
				feeds[i].NoteCard.DisplayTitle = title
			}
		}
		if feeds[i].NoteCard.Cover.URLDefault == "" {
			if cover, ok := coverMap[feeds[i].ID]; ok {
				feeds[i].NoteCard.Cover.URLDefault = cover
			}
		}
	}
	return feeds
}

// GetFanAnalytics 获取粉丝分析数据
func (d *DataAction) GetFanAnalytics(ctx context.Context, period string) (*FanAnalytics, error) {
	timeout, err := d.polling.Delay("wait_300000ms")
	if err != nil {
		return nil, err
	}
	page := d.page.WithContext(ctx).WithTimeout(timeout)

	// 导航到粉丝数据页面
	logrus.Info("导航到粉丝数据页面...")
	url := "https://creator.xiaohongshu.com/statistics/fans-data?source=official"
	if err := page.Goto(url); err != nil {
		return nil, fmt.Errorf("导航失败: %w", err)
	}
	waitStable, err := d.polling.Delay("wait_1000ms")
	if err != nil {
		return nil, err
	}
	if err := page.WaitDOMStable(waitStable, 0.1); err != nil {
		logrus.Warn("等待 DOM 稳定出现问题", "error", err)
	}
	if err := polling.SleepDelay(d.polling, "wait_5000ms"); err != nil {
		return nil, err
	}

	// 提取粉丝分析数据
	result, err := page.Eval(`() => {
		const data = {
			overview: {total_fans: 0, new_fans: 0, lost_fans: 0},
			demographics: {gender: {}, interests: []},
			active_fans: []
		};

		const text = document.body.innerText;

		// 提取粉丝概览数据
		const totalMatch = text.match(/总粉丝数\s*(\d+)/);
		if (totalMatch) data.overview.total_fans = parseInt(totalMatch[1]);

		const newMatch = text.match(/新增粉丝数\s*(\d+)/);
		if (newMatch) data.overview.new_fans = parseInt(newMatch[1]);

		const lostMatch = text.match(/流失粉丝数\s*(\d+)/);
		if (lostMatch) data.overview.lost_fans = parseInt(lostMatch[1]);

		// 提取性别分布
		const maleMatch = text.match(/男性\s*(\d+)%/);
		const femaleMatch = text.match(/女性\s*(\d+)%/);
		if (maleMatch) data.demographics.gender.male = parseInt(maleMatch[1]);
		if (femaleMatch) data.demographics.gender.female = parseInt(femaleMatch[1]);

		// 提取兴趣分布
		const interestKeywords = ['美食', '生活记录', '社科', '娱乐', '家居家装', '影视', '科技数码', '职场'];
		interestKeywords.forEach(keyword => {
			if (text.includes(keyword)) {
				data.demographics.interests.push(keyword);
			}
		});

		// 提取活跃粉丝列表
		const fanItems = document.querySelectorAll('li, .fan-item, [class*="fan"]');
		fanItems.forEach(item => {
			const itemText = item.textContent;
			const match = itemText.match(/(.+?)\s*互动\s*(\d+)\s*次/);
			if (match && data.active_fans.length < 10) {
				data.active_fans.push({
					nickname: match[1].trim(),
					interactions: parseInt(match[2])
				});
			}
		});

		return JSON.stringify(data);
	}`)
	if err != nil {
		return nil, fmt.Errorf("执行 JavaScript 失败: %w", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("JavaScript 返回类型错误")
	}

	var analytics FanAnalytics
	if err := json.Unmarshal([]byte(resultStr), &analytics); err != nil {
		return nil, fmt.Errorf("解析粉丝分析数据失败: %w", err)
	}

	logrus.Infof("获取粉丝分析数据成功")
	return &analytics, nil
}

// SortField 可排序的字段
type SortField string

const (
	SortByExposure       SortField = "exposure"        // 曝光
	SortByViews          SortField = "views"           // 观看
	SortByClickRate      SortField = "click_rate"      // 封面点击率
	SortByLikes          SortField = "likes"           // 点赞
	SortByComments       SortField = "comments"        // 评论
	SortByCollects       SortField = "collects"        // 收藏
	SortByFollowerGrowth SortField = "follower_growth" // 涨粉
	SortByShares         SortField = "shares"          // 分享
	SortByDuration       SortField = "duration"        // 人均观看时长
	SortByBarrage        SortField = "barrage"         // 弹幕
)

// SortOrder 排序方向
type SortOrder string

const (
	SortAsc  SortOrder = "asc"  // 升序
	SortDesc SortOrder = "desc" // 降序
)

// GetContentAnalytics 获取内容分析数据（支持翻页和排序）
// sortBy: 排序字段，空字符串表示不排序
// sortOrder: 排序方向，asc升序/desc降序
func (d *DataAction) GetContentAnalytics(ctx context.Context, limit int, sortBy SortField, sortOrder SortOrder) (*ContentAnalytics, error) {
	timeout, err := d.polling.Delay("wait_300000ms")
	if err != nil {
		return nil, err
	}
	page := d.page.WithContext(ctx).WithTimeout(timeout)

	// 导航到数据分析页面
	logrus.Info("导航到数据分析页面...")
	url := "https://creator.xiaohongshu.com/statistics/data-analysis?source=official"

	if err := page.Goto(url); err != nil {
		return nil, fmt.Errorf("导航失败: %w", err)
	}

	// 等待表格加载完成
	logrus.Info("等待数据表格加载...")
	waitTableJS := `() => {
		const table = document.querySelector('table tbody tr');
		return table !== null;
	}`

	// 最多等待30秒，每500ms检查一次
	maxWaitTime, err := d.polling.Timeout()
	if err != nil {
		return nil, err
	}
	checkInterval, err := d.polling.Interval()
	if err != nil {
		return nil, err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		hasTable, err := page.Eval(waitTableJS)
		if err == nil && hasTable == true {
			logrus.Info("表格加载成功")
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待1秒，确保数据完全渲染
	if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
		return nil, err
	}

	// 如果指定了排序字段，先进行排序
	if sortBy != "" {
		if err := d.applySorting(page, sortBy, sortOrder); err != nil {
			logrus.Warnf("应用排序失败: %v，继续使用默认顺序", err)
		}
	}

	logrus.Info("从表格提取内容分析数据")

	allNotes := []NoteMetrics{}
	pageNum := 1
	maxPages := 50 // 最多翻50页，避免无限循环

	// JavaScript提取函数
	extractJS := `() => {
		const notes = [];
		const table = document.querySelector('table');
		if (!table) {
			return JSON.stringify({notes: [], count: 0, error: '未找到表格'});
		}

		// 获取所有数据行（跳过表头）
		const rows = table.querySelectorAll('tbody tr');

		rows.forEach((row, index) => {
			const cells = row.querySelectorAll('td');
			if (cells.length < 11) return;

			// 提取笔记标题和基本信息
			const firstCell = cells[0];
			const titleElem = firstCell.querySelector('.note-title');
			const timeElem = firstCell.querySelector('.time');

			// 尝试从标题链接提取 feed_id 和 xsec_token
			let feedId = '';
			let xsecToken = '';
			const linkElem = firstCell.querySelector('a[href*="explore"]');
			if (linkElem) {
				const href = linkElem.getAttribute('href');
				// URL格式: /explore/{feed_id}?xsec_token=xxx&xsec_source=pc_user
				const match = href.match(/\/explore\/([a-f0-9]+)/);
				if (match) {
					feedId = match[1];
				}
				const urlParams = new URLSearchParams(href.split('?')[1]);
				xsecToken = urlParams.get('xsec_token') || feedId;
			}

			const title = titleElem ? titleElem.textContent.trim() : '';
			const publishTime = timeElem ? timeElem.textContent.trim() : '';

			// 提取数字数据
			const getNumber = (cellIndex) => {
				if (cellIndex >= cells.length) return 0;
				const cell = cells[cellIndex];
				const cellDiv = cell.querySelector('.d-table__cell');
				const text = cellDiv ? cellDiv.textContent.trim() : cell.textContent.trim();
				if (text === '-' || text === '' || text === '—') return 0;
				const match = text.match(/[\d,]+/);
				if (match) {
					return parseInt(match[0].replace(/,/g, ''));
				}
				return 0;
			};

			const getRate = (cellIndex) => {
				if (cellIndex >= cells.length) return 0;
				const cell = cells[cellIndex];
				const text = cell.textContent.trim();
				const match = text.match(/([\d.]+)%%/);
				return match ? parseFloat(match[1]) : 0;
			};

			const getDuration = (cellIndex) => {
				if (cellIndex >= cells.length) return '';
				const cell = cells[cellIndex];
				const text = cell.textContent.trim();
				return text !== '-' && text !== '' ? text : '';
			};

			notes.push({
				feed_id: feedId,
				xsec_token: xsecToken,
				title: title,
				publish_time: publishTime,
				exposure: getNumber(1),
				views: getNumber(2),
				click_rate: getRate(3),
				likes: getNumber(4),
				comments: getNumber(5),
				collects: getNumber(6),
				follower_growth: getNumber(7),
				shares: getNumber(8),
				avg_view_duration: getDuration(9),
				full_screen: getNumber(10)
			});
		});

		return JSON.stringify({notes: notes, count: notes.length});
	}`

	for pageNum <= maxPages && len(allNotes) < limit {
		// 提取当前页数据
		result, err := page.Eval(extractJS)
		if err != nil {
			logrus.Warnf("第 %d 页提取数据失败: %v", pageNum, err)
			break
		}

		resultStr, ok := result.(string)
		if !ok {
			logrus.Warnf("第 %d 页提取数据类型错误", pageNum)
			break
		}

		var pageData struct {
			Notes []NoteMetrics `json:"notes"`
			Count int           `json:"count"`
		}
		if err := json.Unmarshal([]byte(resultStr), &pageData); err != nil {
			logrus.Warnf("第 %d 页解析数据失败: %v", pageNum, err)
			break
		}

		if len(pageData.Notes) == 0 {
			logrus.Infof("第 %d 页没有数据，停止翻页", pageNum)
			break
		}

		// 添加到结果集
		for _, note := range pageData.Notes {
			if len(allNotes) >= limit {
				break
			}
			allNotes = append(allNotes, note)
		}

		logrus.Infof("第 %d 页提取 %d 条笔记，累计 %d 条", pageNum, len(pageData.Notes), len(allNotes))

		// 如果已经达到限制，退出
		if len(allNotes) >= limit {
			break
		}

		// 检查是否有下一页
		hasNext, err := page.Eval(`() => {
			const nextBtn = document.querySelector('.d-pagination-page.d-clickable:not(.disabled):last-of-type');
			return nextBtn !== null && !nextBtn.classList.contains('disabled');
		}`)
		if err != nil || hasNext == false {
			logrus.Info("没有更多页面，停止翻页")
			break
		}

		// 点击下一页
		logrus.Infof("点击下一页...")
		_, clickErr := page.Eval(`() => {
			const nextBtn = document.querySelector('.d-pagination-page.d-clickable:not(.disabled):last-of-type');
			if (nextBtn) {
				nextBtn.click();
				return true;
			}
			return false;
		}`)
		if clickErr != nil {
			logrus.Warnf("点击下一页失败: %v", clickErr)
			break
		}

		// 等待页面加载
		if err := polling.SleepDelay(d.polling, "wait_2000ms"); err != nil {
			return nil, err
		}
		pageNum++
	}

	logrus.Infof("获取内容分析数据成功，共 %d 条笔记", len(allNotes))
	return &ContentAnalytics{Notes: allNotes}, nil
}

// applySorting 应用排序到页面
func (d *DataAction) applySorting(page browser.Page, sortBy SortField, sortOrder SortOrder) error {
	// 映射排序字段到表头列索引
	columnMap := map[SortField]int{
		SortByExposure:       2,  // 曝光（第2列，从1开始）
		SortByViews:          3,  // 观看
		SortByClickRate:      4,  // 封面点击率
		SortByLikes:          5,  // 点赞
		SortByComments:       6,  // 评论
		SortByCollects:       7,  // 收藏
		SortByFollowerGrowth: 8,  // 涨粉
		SortByShares:         9,  // 分享
		SortByDuration:       10, // 人均观看时长
		SortByBarrage:        11, // 弹幕
	}

	columnIndex, ok := columnMap[sortBy]
	if !ok {
		return fmt.Errorf("不支持的排序字段: %s", sortBy)
	}

	logrus.Infof("按 %s %s 排序", sortBy, sortOrder)

	// 排序图标循环模式: 未排序 → 升序(1次点击) → 降序(2次点击) → 未排序(3次点击)
	// 默认页面处于未排序状态
	clickCount := 1 // 升序需要点击1次
	if sortOrder == SortDesc {
		clickCount = 2 // 降序需要点击2次
	}

	clickScript := fmt.Sprintf(`() => {
		const header = document.querySelector('table thead th:nth-child(%d)');
		if (!header) {
			return {error: '未找到表头列'};
		}
		const sortIcon = header.querySelector('.d-table__th-cell-sort');
		if (!sortIcon) {
			return {error: '未找到排序图标'};
		}
		sortIcon.click();
		return {clicked: true};
	}`, columnIndex)

	// 执行点击
	for i := 0; i < clickCount; i++ {
		_, err := page.Eval(clickScript)
		if err != nil {
			return fmt.Errorf("点击排序图标失败(第%d次): %w", i+1, err)
		}
		if err := polling.SleepDelay(d.polling, "wait_1000ms"); err != nil {
			return err
		}
	}

	logrus.Info("排序应用成功")
	return nil
}

// convertNoteData 将API返回的笔记数据转换为NoteMetrics格式
func convertNoteData(data map[string]interface{}) NoteMetrics {
	// 安全地获取各个字段
	getInt := func(key string) int {
		if v, ok := data[key].(float64); ok {
			return int(v)
		}
		return 0
	}

	getFloat := func(key string) float64 {
		if v, ok := data[key].(float64); ok {
			return v
		}
		return 0
	}

	getString := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	// 转换发布时间
	var publishTime string
	if postTime, ok := data["post_time"].(float64); ok {
		t := time.Unix(int64(postTime/1000), 0)
		publishTime = t.Format("2006-01-02 15:04")
	}

	// 转换人均观看时长
	avgDuration := ""
	if viewTimeAvg := getInt("view_time_avg"); viewTimeAvg > 0 {
		avgDuration = fmt.Sprintf("%ds", viewTimeAvg)
	}

	// 转换点击率为百分比
	clickRate := getFloat("coverClickRate") * 100

	return NoteMetrics{
		Title:           getString("title"),
		PublishTime:     publishTime,
		Exposure:        getInt("imp_count"),
		Views:           getInt("read_count"),
		ClickRate:       clickRate,
		Likes:           getInt("like_count"),
		Comments:        getInt("comment_count"),
		Collects:        getInt("fav_count"),
		FollowerGrowth:  getInt("increase_fans_count"),
		Shares:          getInt("share_count"),
		AvgViewDuration: avgDuration,
		FullScreen:      getInt("danmaku_count"),
		Status:          "normal", // 默认正常状态
	}
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
