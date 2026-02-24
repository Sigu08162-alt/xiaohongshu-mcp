package interaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/interaction"
	"github.com/vmxmy/xiaohongshu-mcp/internal/app/testkit"
)

func TestLikeFeed_CallsGateway(t *testing.T) {
	gw := &testkit.FakeInteractionGateway{}
	uc := &interaction.Usecase{Gateway: gw}

	err := uc.LikeFeed(context.Background(), "feed-1", "token-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.LikeCalls != 1 {
		t.Errorf("expected 1 like call, got %d", gw.LikeCalls)
	}
}

func TestLikeFeed_PropagatesError(t *testing.T) {
	gw := &testkit.FakeInteractionGateway{Err: errors.New("rate limited")}
	uc := &interaction.Usecase{Gateway: gw}

	err := uc.LikeFeed(context.Background(), "feed-1", "token-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPostComment_PassesContent(t *testing.T) {
	gw := &testkit.FakeInteractionGateway{}
	uc := &interaction.Usecase{Gateway: gw}

	err := uc.PostComment(context.Background(), "feed-1", "token-1", "好帖！")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.CommentCalls != 1 {
		t.Errorf("expected 1 comment call, got %d", gw.CommentCalls)
	}
	if gw.LastComment != "好帖！" {
		t.Errorf("expected comment '好帖！', got '%s'", gw.LastComment)
	}
}

func TestUnlikeFeed_CallsGateway(t *testing.T) {
	gw := &testkit.FakeInteractionGateway{}
	uc := &interaction.Usecase{Gateway: gw}

	err := uc.UnlikeFeed(context.Background(), "feed-1", "token-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.LikeCalls != 1 {
		t.Errorf("expected 1 unlike call, got %d", gw.LikeCalls)
	}
}
