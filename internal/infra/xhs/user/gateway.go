// Package user implements the ports.UserGateway interface.
package user

import (
	"context"
	"fmt"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// Gateway implements ports.UserGateway.
type Gateway struct {
	pageFactory func() (browser.Page, func(), error)
	polling     polling.Module
}

// New creates a new user Gateway.
func New(pageFactory func() (browser.Page, func(), error), pollingModule polling.Module) *Gateway {
	return &Gateway{pageFactory: pageFactory, polling: pollingModule}
}

func (g *Gateway) withPage(ctx context.Context, fn func(browser.Page) error) error {
	page, cleanup, err := g.pageFactory()
	if err != nil {
		return fmt.Errorf("user gateway: create page: %w", err)
	}
	defer cleanup()
	return fn(page.WithContext(ctx))
}

// FollowUser follows or unfollows a user.
func (g *Gateway) FollowUser(ctx context.Context, userID, xsecToken string, unfollow bool) error {
	return g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewFollowAction(page, g.polling)
		if err != nil {
			return err
		}
		if unfollow {
			return action.Unfollow(ctx, userID, xsecToken)
		}
		return action.Follow(ctx, userID, xsecToken)
	})
}

// GetUserProfile returns a user's profile page data.
func (g *Gateway) GetUserProfile(ctx context.Context, userID, xsecToken string) (*xiaohongshu.UserProfileResponse, error) {
	var result *xiaohongshu.UserProfileResponse
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewUserProfileAction(page, g.polling)
		if err != nil {
			return err
		}
		result, err = action.UserProfile(ctx, userID, xsecToken)
		return err
	})
	return result, err
}

// GetMyProfile returns the current logged-in user's profile via sidebar.
func (g *Gateway) GetMyProfile(ctx context.Context) (*xiaohongshu.UserProfileResponse, error) {
	return g.GetMyProfileTab(ctx, "note")
}

func (g *Gateway) GetMyProfileTab(ctx context.Context, tab string) (*xiaohongshu.UserProfileResponse, error) {
	var result *xiaohongshu.UserProfileResponse
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewUserProfileAction(page, g.polling)
		if err != nil {
			return err
		}
		result, err = action.GetMyProfileTabViaSidebar(ctx, tab)
		return err
	})
	return result, err
}

// GetMyStats returns the current user's stats (fans, follows, likes, notes).
func (g *Gateway) GetMyStats(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewDataAction(page, g.polling)
		if err != nil {
			return err
		}
		stats, err := action.GetMyStats(ctx)
		if err != nil {
			return err
		}
		// Convert struct to map for port interface compatibility
		result = map[string]any{
			"follower_count": stats.FollowerCount,
			"follow_count":   stats.FollowCount,
			"liked_count":    stats.LikedCount,
			"note_count":     stats.NoteCount,
			"collect_count":  stats.CollectCount,
		}
		return nil
	})
	return result, err
}
