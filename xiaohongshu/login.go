package xiaohongshu

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/polling"
)

type LoginAction struct {
	page    browser.Page
	polling polling.Module
}

func NewLogin(page browser.Page, pollingModule polling.Module) *LoginAction {
	return &LoginAction{page: page, polling: pollingModule}
}

func (a *LoginAction) CheckLoginStatus(ctx context.Context) (bool, error) {
	pp := a.page.WithContext(ctx)
	if err := pp.Goto("https://www.xiaohongshu.com/explore"); err != nil {
		return false, errors.Wrap(err, "导航失败")
	}
	if err := pp.WaitLoad(); err != nil {
		return false, errors.Wrap(err, "等待页面加载失败")
	}

	if err := polling.SleepDelay(a.polling, "wait_1000ms"); err != nil {
		return false, err
	}

	exists, err := pp.Has(`.main-container .user .link-wrapper .channel`)
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	if !exists {
		return false, errors.Wrap(err, "login status element not found")
	}

	return true, nil
}

func (a *LoginAction) Login(ctx context.Context) error {
	pp := a.page.WithContext(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	if err := pp.Goto("https://www.xiaohongshu.com/explore"); err != nil {
		return errors.Wrap(err, "导航失败")
	}
	if err := pp.WaitLoad(); err != nil {
		return errors.Wrap(err, "等待页面加载失败")
	}

	// 等待一小段时间让页面完全加载
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return err
	}

	// 检查是否已经登录
	if exists, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		// 已经登录，直接返回
		return nil
	}

	// 等待扫码成功提示或者登录完成
	// 这里我们等待登录成功的元素出现，这样更简单可靠
	_, err := pp.Element(".main-container .user .link-wrapper .channel")
	if err != nil {
		return errors.Wrap(err, "等待登录元素失败")
	}

	return nil
}

func (a *LoginAction) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	pp := a.page.WithContext(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	if err := pp.Goto("https://www.xiaohongshu.com/explore"); err != nil {
		return "", false, errors.Wrap(err, "导航失败")
	}
	if err := pp.WaitLoad(); err != nil {
		return "", false, errors.Wrap(err, "等待页面加载失败")
	}

	// 等待一小段时间让页面完全加载
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return "", false, err
	}

	// 检查是否已经登录
	if exists, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		return "", true, nil
	}

	// 获取二维码图片
	elem, err := pp.Element(".login-container .qrcode-img")
	if err != nil {
		return "", false, errors.Wrap(err, "找不到二维码元素")
	}

	src, err := elem.Attribute("src")
	if err != nil {
		return "", false, errors.Wrap(err, "get qrcode src failed")
	}
	if len(src) == 0 {
		return "", false, errors.New("qrcode src is empty")
	}

	return src, false, nil
}

func (a *LoginAction) WaitForLogin(ctx context.Context) bool {
	pp := a.page.WithContext(ctx)
	interval, err := a.polling.Interval()
	if err != nil {
		return false
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			el, err := pp.Element(".main-container .user .link-wrapper .channel")
			if err == nil && el != nil {
				return true
			}
		}
	}
}
