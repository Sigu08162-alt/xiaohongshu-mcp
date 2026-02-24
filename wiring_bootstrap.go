package main

import (
	apppublish "github.com/vmxmy/xiaohongshu-mcp/internal/app/publish"
	"github.com/vmxmy/xiaohongshu-mcp/internal/interfaces/wiring"
)

// buildPublishUsecase delegates to the wiring package.
func buildPublishUsecase(cfg interface{}, selectors map[string]string, headless bool) (*apppublish.Usecase, error) {
	return wiring.LoadPublishUsecase(headless)
}

func loadPublishUsecase(headless bool) (*apppublish.Usecase, error) {
	return wiring.LoadPublishUsecase(headless)
}

func initPublishUsecase(headless bool) *apppublish.Usecase {
	return wiring.InitPublishUsecase(headless)
}
