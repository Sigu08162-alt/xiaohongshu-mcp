package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

type UserProfileAction struct {
	page browser.Page
}

func NewUserProfileAction(page browser.Page) *UserProfileAction {
	pp := page.WithTimeout(60 * time.Second)
	return &UserProfileAction{page: pp}
}

// UserProfile 获取用户基本信息及帖子
func (u *UserProfileAction) UserProfile(ctx context.Context, userID, xsecToken string) (*UserProfileResponse, error) {
	page := u.page.WithContext(ctx)

	searchURL := makeUserProfileURL(userID, xsecToken)
	if err := page.Goto(searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to user profile: %w", err)
	}
	if err := page.WaitDOMStable(time.Second, 0.1); err != nil {
		return nil, fmt.Errorf("failed to wait for page stable: %w", err)
	}

	return u.extractUserProfileData(page)
}

// extractUserProfileData 从页面中提取用户资料数据的通用方法
func (u *UserProfileAction) extractUserProfileData(page browser.Page) (*UserProfileResponse, error) {
	// 等待 __INITIAL_STATE__ 对象存在
	if err := page.WaitForFunction(`() => window.__INITIAL_STATE__ !== undefined`, 30*time.Second); err != nil {
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
	page := u.page.WithContext(ctx)

	// 创建导航动作
	navigate := NewNavigate(page)

	// 通过侧边栏导航到个人主页
	if err := navigate.ToProfilePage(ctx); err != nil {
		return nil, fmt.Errorf("导航到个人主页失败: %w", err)
	}

	// 等待 __INITIAL_STATE__ 中的用户数据加载，而不是等待 DOM 稳定
	// 个人主页有动态内容（笔记推荐、实时更新），DOM 可能永远不会稳定
	maxWait := 30 * time.Second
	checkInterval := 500 * time.Millisecond
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
	time.Sleep(500 * time.Millisecond)

	return u.extractUserProfileData(page)
}
