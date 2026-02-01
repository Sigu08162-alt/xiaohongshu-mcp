package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xpzouying/xiaohongshu-mcp/errors"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/polling"
)

type FeedsListAction struct {
	page    browser.Page
	polling polling.Module
}

func NewFeedsListAction(page browser.Page, pollingModule polling.Module) (*FeedsListAction, error) {
	timeout, err := pollingModule.Timeout()
	if err != nil {
		return nil, err
	}
	pp := page.WithTimeout(timeout)

	if err := pp.Goto("https://www.xiaohongshu.com"); err != nil {
		panic(fmt.Sprintf("导航失败: %v", err))
	}

	// 等待 __INITIAL_STATE__ 加载，而不是等待 DOM 稳定
	// 小红书首页有动态内容（轮播、推荐刷新），DOM 永远不会稳定
	maxWait, err := pollingModule.Timeout()
	if err != nil {
		return nil, err
	}
	checkInterval, err := pollingModule.Interval()
	if err != nil {
		return nil, err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasState, err := pp.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.feed &&
			       window.__INITIAL_STATE__.feed.feeds !== undefined;
		}`)
		if err == nil && hasState == true {
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待500ms确保数据完全加载
	if err := polling.SleepDelay(pollingModule, "wait_500ms"); err != nil {
		return nil, err
	}

	return &FeedsListAction{page: pp, polling: pollingModule}, nil
}

// GetFeedsList 获取页面的 Feed 列表数据
func (f *FeedsListAction) GetFeedsList(ctx context.Context) ([]Feed, error) {
	page := f.page.WithContext(ctx)

	if err := polling.SleepDelay(f.polling, "wait_1000ms"); err != nil {
		return nil, err
	}

	resultRaw, err := page.Eval(`() => {
		if (window.__INITIAL_STATE__ &&
		    window.__INITIAL_STATE__.feed &&
		    window.__INITIAL_STATE__.feed.feeds) {
			const feeds = window.__INITIAL_STATE__.feed.feeds;
			const feedsData = feeds.value !== undefined ? feeds.value : feeds._value;
			if (feedsData) {
				return JSON.stringify(feedsData);
			}
		}
		return "";
	}`)
	if err != nil {
		return nil, fmt.Errorf("failed to eval feeds: %w", err)
	}

	result, ok := resultRaw.(string)
	if !ok || result == "" {
		return nil, errors.ErrNoFeeds
	}

	var feeds []Feed
	if err := json.Unmarshal([]byte(result), &feeds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal feeds: %w", err)
	}

	return feeds, nil
}
