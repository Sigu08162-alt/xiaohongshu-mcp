package main

import (
	"context"
	"errors"
	"testing"
)

type fakeLoginSession struct {
	openErr     error
	loggedInErr error
	nicknameErr error
	loggedIn    bool
	nickname    string
	openCalled  bool
	closeCalled bool
}

func (f *fakeLoginSession) Open(ctx context.Context) error {
	f.openCalled = true
	return f.openErr
}

func (f *fakeLoginSession) LoggedIn(ctx context.Context) (bool, error) {
	return f.loggedIn, f.loggedInErr
}

func (f *fakeLoginSession) QRCode(ctx context.Context) (loginQRCode, error) {
	return loginQRCode{}, nil
}

func (f *fakeLoginSession) SaveCookies() error {
	return nil
}

func (f *fakeLoginSession) Close() error {
	f.closeCalled = true
	return nil
}

func (f *fakeLoginSession) Nickname(ctx context.Context) (string, error) {
	if f.nicknameErr != nil {
		return "", f.nicknameErr
	}
	return f.nickname, nil
}

func TestCheckLoginStatus_UsesSessionNickname(t *testing.T) {
	fs := &fakeLoginSession{
		loggedIn: true,
		nickname: "真实用户",
	}

	svc := &XiaohongshuService{
		loginSessionNew: func() (loginSession, error) { return fs, nil },
	}

	got, err := svc.CheckLoginStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckLoginStatus() error = %v", err)
	}
	if !fs.openCalled {
		t.Fatalf("expected session Open to be called")
	}
	if !fs.closeCalled {
		t.Fatalf("expected session Close to be called")
	}
	if !got.IsLoggedIn {
		t.Fatalf("expected logged in true")
	}
	if got.Username != "真实用户" {
		t.Fatalf("expected username from session nickname, got %q", got.Username)
	}
}

func TestCheckLoginStatus_LoggedOut(t *testing.T) {
	fs := &fakeLoginSession{
		loggedIn: false,
	}

	svc := &XiaohongshuService{
		loginSessionNew: func() (loginSession, error) { return fs, nil },
	}

	got, err := svc.CheckLoginStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckLoginStatus() error = %v", err)
	}
	if got.IsLoggedIn {
		t.Fatalf("expected logged in false")
	}
	if got.Username != "" {
		t.Fatalf("expected empty username when logged out, got %q", got.Username)
	}
}

func TestCheckLoginStatus_NicknameUnknownTreatedAsLoggedOut(t *testing.T) {
	fs := &fakeLoginSession{
		loggedIn: true,
		nickname: "",
	}

	svc := &XiaohongshuService{
		loginSessionNew: func() (loginSession, error) { return fs, nil },
	}

	got, err := svc.CheckLoginStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckLoginStatus() error = %v", err)
	}
	if got.IsLoggedIn {
		t.Fatalf("expected logged in false when nickname is unknown")
	}
}

func TestCheckLoginStatus_NicknamePlaceholderTreatedAsLoggedOut(t *testing.T) {
	fs := &fakeLoginSession{
		loggedIn: true,
		nickname: "我",
	}

	svc := &XiaohongshuService{
		loginSessionNew: func() (loginSession, error) { return fs, nil },
	}

	got, err := svc.CheckLoginStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckLoginStatus() error = %v", err)
	}
	if got.IsLoggedIn {
		t.Fatalf("expected logged in false when nickname is placeholder")
	}
}

func TestCheckLoginStatus_PlatformTitleTreatedAsLoggedOut(t *testing.T) {
	fs := &fakeLoginSession{
		loggedIn: true,
		nickname: "小红书创作服务平台",
	}

	svc := &XiaohongshuService{
		loginSessionNew: func() (loginSession, error) { return fs, nil },
	}

	got, err := svc.CheckLoginStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckLoginStatus() error = %v", err)
	}
	if got.IsLoggedIn {
		t.Fatalf("expected logged in false when nickname is platform title")
	}
}

func TestCheckLoginStatus_OpenError(t *testing.T) {
	fs := &fakeLoginSession{
		openErr: errors.New("open failed"),
	}

	svc := &XiaohongshuService{
		loginSessionNew: func() (loginSession, error) { return fs, nil },
	}

	if _, err := svc.CheckLoginStatus(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}
