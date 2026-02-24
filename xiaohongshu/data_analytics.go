package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

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

		for _, note := range pageData.Notes {
			if len(allNotes) >= limit {
				break
			}
			allNotes = append(allNotes, note)
		}

		logrus.Infof("第 %d 页提取 %d 条笔记，累计 %d 条", pageNum, len(pageData.Notes), len(allNotes))

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
	columnMap := map[SortField]int{
		SortByExposure:       2,
		SortByViews:          3,
		SortByClickRate:      4,
		SortByLikes:          5,
		SortByComments:       6,
		SortByCollects:       7,
		SortByFollowerGrowth: 8,
		SortByShares:         9,
		SortByDuration:       10,
		SortByBarrage:        11,
	}

	columnIndex, ok := columnMap[sortBy]
	if !ok {
		return fmt.Errorf("不支持的排序字段: %s", sortBy)
	}

	logrus.Infof("按 %s %s 排序", sortBy, sortOrder)

	clickCount := 1
	if sortOrder == SortDesc {
		clickCount = 2
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

	var publishTime string
	if postTime, ok := data["post_time"].(float64); ok {
		t := time.Unix(int64(postTime/1000), 0)
		publishTime = t.Format("2006-01-02 15:04")
	}

	avgDuration := ""
	if viewTimeAvg := getInt("view_time_avg"); viewTimeAvg > 0 {
		avgDuration = fmt.Sprintf("%ds", viewTimeAvg)
	}

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
		Status:          "normal",
	}
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
