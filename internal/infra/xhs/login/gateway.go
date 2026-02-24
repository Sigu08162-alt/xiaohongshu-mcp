// Package login implements the ports.LoginGateway interface.
package login

import (
	"context"
	"fmt"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// Gateway implements ports.LoginGateway.
type Gateway struct {
	pageFactory func() (browser.Page, func(), error)
	polling     polling.Module
}

// New creates a new login Gateway.
func New(pageFactory func() (browser.Page, func(), error), pollingModule polling.Module) *Gateway {
	return &Gateway{pageFactory: pageFactory, polling: pollingModule}
}

func (g *Gateway) withPage(ctx context.Context, fn func(browser.Page) error) error {
	page, cleanup, err := g.pageFactory()
	if err != nil {
		return fmt.Errorf("login gateway: create page: %w", err)
	}
	defer cleanup()
	return fn(page.WithContext(ctx))
}

// CheckLoginStatus returns whether the current session is logged in.
func (g *Gateway) CheckLoginStatus(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	err := g.withPage(ctx, func(page browser.Page) error {
		action := xiaohongshu.NewLogin(page, g.polling)
		loggedIn, err := action.CheckLoginStatus(ctx)
		if err != nil {
			return err
		}
		result = map[string]any{"logged_in": loggedIn}
		return nil
	})
	return result, err
}

// GetLoginQRCode returns a base64 QR code image for login.
func (g *Gateway) GetLoginQRCode(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	err := g.withPage(ctx, func(page browser.Page) error {
		action := xiaohongshu.NewLogin(page, g.polling)
		imgBase64, expired, err := action.FetchQrcodeImage(ctx)
		if err != nil {
			return err
		}
		result = map[string]any{
			"qrcode_base64": imgBase64,
			"expired":       expired,
		}
		return nil
	})
	return result, err
}
