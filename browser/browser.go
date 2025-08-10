package browser

import (
	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
)

var (
	browser *headless_browser.Browser
)

func Init(headless bool) error {

	opts := []headless_browser.Option{
		headless_browser.WithHeadless(headless),
	}

	// 加载 cookies
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)

	if data, err := cookieLoader.LoadCookies(); err == nil {
		opts = append(opts, headless_browser.WithCookies(string(data)))
		logrus.Debugf("loaded cookies from filesuccessfully")
	} else {
		logrus.Warnf("failed to load cookies: %v", err)
	}

	browser = headless_browser.New(opts...)

	return nil
}

func NewPage() *rod.Page {
	return browser.NewPage()
}

func Close() {
	browser.Close()
}
