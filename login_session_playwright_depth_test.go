package main

import (
	"context"
	"testing"
)

type fakeFrameCycle struct{}

func (f fakeFrameCycle) HasRegex(ctx context.Context, selector, jsRegex string) (bool, error) {
	return false, nil
}

func (f fakeFrameCycle) Element(ctx context.Context, selector string) (qrElement, error) {
	return nil, nil
}

func (f fakeFrameCycle) ElementByRegex(ctx context.Context, selector, jsRegex string) (qrElement, error) {
	return nil, nil
}

func (f fakeFrameCycle) Elements(ctx context.Context, selector string) ([]qrElement, error) {
	return nil, nil
}

func (f fakeFrameCycle) Frames(ctx context.Context) ([]qrFrame, error) {
	return []qrFrame{f}, nil
}

func TestFrameHasSecurityHint_DepthLimitStopsCycle(t *testing.T) {
	s := &playwrightLoginSession{}
	if s.frameHasSecurityHint(context.Background(), fakeFrameCycle{}, 0) {
		t.Fatalf("expected depth limit to stop cycle")
	}
}
