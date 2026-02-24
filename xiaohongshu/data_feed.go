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

// FollowerUser 粉丝/关注用户信息
type FollowerUser struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Desc     string `json:"desc"`
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
