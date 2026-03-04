package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStatusPage struct {
	values map[string]bool
	errs   map[string]error
}

func (f *fakeStatusPage) Navigate(ctx context.Context, url string) error { return nil }
func (f *fakeStatusPage) WaitLoad(ctx context.Context) error             { return nil }
func (f *fakeStatusPage) Close() error                                   { return nil }
func (f *fakeStatusPage) HasRegex(ctx context.Context, selector, jsRegex string) (bool, error) {
	return false, nil
}
func (f *fakeStatusPage) Element(ctx context.Context, selector string) (qrElement, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeStatusPage) ElementByRegex(ctx context.Context, selector, jsRegex string) (qrElement, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeStatusPage) Elements(ctx context.Context, selector string) ([]qrElement, error) {
	return nil, nil
}
func (f *fakeStatusPage) Frames(ctx context.Context) ([]qrFrame, error) { return nil, nil }

func (f *fakeStatusPage) Has(ctx context.Context, selector string) (bool, error) {
	if err, ok := f.errs[selector]; ok {
		return false, err
	}
	if v, ok := f.values[selector]; ok {
		return v, nil
	}
	return false, nil
}

func TestPlaywrightLoginSessionLoggedIn_ReturnsErrorOnSelectorFailure(t *testing.T) {
	p := &fakeStatusPage{
		values: map[string]bool{},
		errs: map[string]error{
			loginStatusSelector: errors.New("dom broken"),
		},
	}

	s := &playwrightLoginSession{
		page:  p,
		sleep: func(time.Duration) {},
	}

	_, err := s.LoggedIn(context.Background())
	if err == nil {
		t.Fatalf("expected selector error to be returned")
	}
}

func TestPlaywrightLoginSessionLoggedIn_NotLoggedWhenLoginModalVisible(t *testing.T) {
	p := &fakeStatusPage{
		values: map[string]bool{
			loginStatusSelector: true,
			".login-container":  true,
		},
		errs: map[string]error{},
	}

	s := &playwrightLoginSession{
		page:  p,
		sleep: func(time.Duration) {},
	}

	got, err := s.LoggedIn(context.Background())
	if err != nil {
		t.Fatalf("LoggedIn() error = %v", err)
	}
	if got {
		t.Fatalf("expected false when login modal is still visible")
	}
}
