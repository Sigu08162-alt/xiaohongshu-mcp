package publish

import (
	"errors"
	"testing"
	"time"
)

type fakeSelectorProbe struct {
	callFn func(expression string, args ...interface{}) (interface{}, error)
}

func (f *fakeSelectorProbe) Eval(expression string, args ...interface{}) (interface{}, error) {
	if f.callFn == nil {
		return nil, errors.New("callFn is nil")
	}
	return f.callFn(expression, args...)
}

func TestWaitForSelectorAttached_ImmediateSuccess(t *testing.T) {
	probe := &fakeSelectorProbe{callFn: func(expression string, args ...interface{}) (interface{}, error) {
		return true, nil
	}}

	err := waitForSelectorAttached(probe, `input[type="file"]`, 20*time.Millisecond, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestWaitForSelectorAttached_RetryUntilSuccess(t *testing.T) {
	calls := 0
	probe := &fakeSelectorProbe{callFn: func(expression string, args ...interface{}) (interface{}, error) {
		calls++
		if calls < 3 {
			return false, nil
		}
		return true, nil
	}}

	err := waitForSelectorAttached(probe, `input[type="file"]`, 40*time.Millisecond, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 probe calls, got %d", calls)
	}
}

func TestWaitForSelectorAttached_Timeout(t *testing.T) {
	probe := &fakeSelectorProbe{callFn: func(expression string, args ...interface{}) (interface{}, error) {
		return false, nil
	}}

	err := waitForSelectorAttached(probe, `input[type="file"]`, 8*time.Millisecond, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
