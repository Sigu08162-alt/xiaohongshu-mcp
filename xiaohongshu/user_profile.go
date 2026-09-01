package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

type UserProfileAction struct {
	page    browser.Page
	polling polling.Module
}

func NewUserProfileAction(page browser.Page, pollingModule polling.Module) (*UserProfileAction, error) {
	timeout, err := pollingModule.Delay("wait_60000ms")
	if err != nil {
		return nil, err
	}
	pp := page.WithTimeout(timeout)
	return &UserProfileAction{page: pp, polling: pollingModule}, nil
}

// UserProfile 获取用户基本信息及帖子
func (u *UserProfileAction) UserProfile(ctx context.Context, userID, xsecToken string) (*UserProfileResponse, error) {
	page := u.page.WithContext(ctx)

	searchURL := makeUserProfileURL(userID, xsecToken)
	if err := page.Goto(searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to user profile: %w", err)
	}
	waitStable, err := u.polling.Delay("wait_1000ms")
	if err != nil {
		return nil, err
	}
	if err := page.WaitDOMStable(waitStable, 0.1); err != nil {
		return nil, fmt.Errorf("failed to wait for page stable: %w", err)
	}

	return u.extractUserProfileData(page)
}

// extractUserProfileData 从页面中提取用户资料数据的通用方法
func (u *UserProfileAction) extractUserProfileData(page browser.Page) (*UserProfileResponse, error) {
	// 等待 __INITIAL_STATE__ 对象存在
	timeout, err := u.polling.Timeout()
	if err != nil {
		return nil, err
	}
	if err := page.WaitForFunction(`() => window.__INITIAL_STATE__ !== undefined`, timeout); err != nil {
		return nil, fmt.Errorf("failed to wait for __INITIAL_STATE__: %w", err)
	}

	// 获取用户数据：window.__INITIAL_STATE__.user.userPageData.value
	userDataRaw, err := page.Eval(`() => {
		if (window.__INITIAL_STATE__ &&
		    window.__INITIAL_STATE__.user &&
		    window.__INITIAL_STATE__.user.userPageData) {
			const userPageData = window.__INITIAL_STATE__.user.userPageData;
			const data = userPageData.value !== undefined ? userPageData.value : userPageData._value;
			if (data) {
				return JSON.stringify(data);
			}
		}
		return "";
	}`)
	if err != nil {
		return nil, fmt.Errorf("failed to eval user data: %w", err)
	}

	userDataResult, ok := userDataRaw.(string)
	if !ok || userDataResult == "" {
		return nil, fmt.Errorf("user.userPageData.value not found in __INITIAL_STATE__")
	}

	// 获取用户帖子：window.__INITIAL_STATE__.user.notes.value
	notesRaw, err := page.Eval(`() => {
		if (window.__INITIAL_STATE__ &&
		    window.__INITIAL_STATE__.user &&
		    window.__INITIAL_STATE__.user.notes) {
			const notes = window.__INITIAL_STATE__.user.notes;
			// 优先使用 value（getter），如果不存在则使用 _value（内部字段）
			const data = notes.value !== undefined ? notes.value : notes._value;
			if (data) {
				return JSON.stringify(data);
			}
		}
		return "";
	}`)
	if err != nil {
		return nil, fmt.Errorf("failed to eval notes data: %w", err)
	}

	notesResult, ok := notesRaw.(string)
	if !ok || notesResult == "" {
		return nil, fmt.Errorf("user.notes.value not found in __INITIAL_STATE__")
	}

	// 解析用户信息
	var userPageData struct {
		Interactions []UserInteractions `json:"interactions"`
		BasicInfo    UserBasicInfo      `json:"basicInfo"`
	}
	if err := json.Unmarshal([]byte(userDataResult), &userPageData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal userPageData: %w", err)
	}

	// 解析帖子数据（帖子为双重数组）
	var notesFeeds [][]Feed
	if err := json.Unmarshal([]byte(notesResult), &notesFeeds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notes: %w", err)
	}

	// 组装响应
	response := &UserProfileResponse{
		UserBasicInfo: userPageData.BasicInfo,
		Interactions:  userPageData.Interactions,
	}

	// 添加用户帖子（展平双重数组）
	for _, feeds := range notesFeeds {
		if len(feeds) != 0 {
			response.Feeds = append(response.Feeds, feeds...)
		}
	}

	return response, nil
}

func makeUserProfileURL(userID, xsecToken string) string {
	return fmt.Sprintf("https://www.xiaohongshu.com/user/profile/%s?xsec_token=%s&xsec_source=pc_note", userID, xsecToken)
}

func (u *UserProfileAction) GetMyProfileViaSidebar(ctx context.Context) (*UserProfileResponse, error) {
	return u.GetMyProfileTabViaSidebar(ctx, "note")
}

// GetMyProfileTabViaSidebar returns the current user's notes, favorites, or liked notes.
// It only reads the profile page and never changes interaction state.
func (u *UserProfileAction) GetMyProfileTabViaSidebar(ctx context.Context, tab string) (*UserProfileResponse, error) {
	page := u.page.WithContext(ctx)
	if tab == "" {
		tab = "note"
	}
	if tab != "note" && tab != "fav" && tab != "liked" {
		return nil, fmt.Errorf("unsupported profile tab %q", tab)
	}

	// 创建导航动作
	navigate := NewNavigate(page, u.polling)

	// 通过侧边栏导航到个人主页
	if err := navigate.ToProfilePage(ctx); err != nil {
		return nil, fmt.Errorf("导航到个人主页失败: %w", err)
	}

	// 等待 __INITIAL_STATE__ 中的用户数据加载，而不是等待 DOM 稳定
	// 个人主页有动态内容（笔记推荐、实时更新），DOM 可能永远不会稳定
	maxWait, err := u.polling.Timeout()
	if err != nil {
		return nil, err
	}
	checkInterval, err := u.polling.Interval()
	if err != nil {
		return nil, err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasData, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.user &&
			       window.__INITIAL_STATE__.user.userPageData !== undefined;
		}`)
		if err == nil && hasData == true {
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待500ms确保数据完全加载
	if err := polling.SleepDelay(u.polling, "wait_500ms"); err != nil {
		return nil, err
	}

	// userPageData is created before the profile payload and tabs are rendered.
	// Wait for real profile content instead of accepting the initial empty ref.
	maxWait, err = u.polling.Timeout()
	if err != nil {
		return nil, err
	}
	checkInterval, err = u.polling.Interval()
	if err != nil {
		return nil, err
	}
	startTime = time.Now()
	for time.Since(startTime) < maxWait {
		ready, evalErr := page.Eval(`() => {
			const ref = window.__INITIAL_STATE__?.user?.userPageData;
			const data = ref?.value ?? ref?._value ?? ref?._rawValue ?? ref;
			const basic = data?.basicInfo;
			const text = document.body?.innerText || '';
			return Boolean(basic?.nickname || basic?.redId || text.includes('收藏'));
		}`)
		if evalErr == nil && ready == true {
			break
		}
		time.Sleep(checkInterval)
	}

	if tab != "note" {
		labels := []string{"收藏", "收藏夹"}
		if tab == "liked" {
			labels = []string{"赞过", "点赞"}
		}
		labelsJSON, _ := json.Marshal(labels)
		clicked, err := page.Eval(fmt.Sprintf(`() => {
			const labels = %s;
			const candidates = [...document.querySelectorAll('button, [role="tab"], .reds-tab-item, .tab-item, span')];
			const el = candidates.find(node => {
				const text = (node.textContent || '').trim();
				return labels.some(label => text === label || text.startsWith(label + ' '));
			});
			if (!el) return false;
			el.click();
			return true;
		}`, string(labelsJSON)))
		if err != nil {
			return nil, fmt.Errorf("切换个人主页标签失败: %w", err)
		}
		if clicked != true {
			return nil, fmt.Errorf("个人主页未找到%s标签", labels[0])
		}
		// The profile store is replaced asynchronously after switching tabs.
		if err := polling.SleepDelay(u.polling, "wait_1000ms"); err != nil {
			return nil, err
		}
	}

	return u.extractUserProfileData(page)
}
