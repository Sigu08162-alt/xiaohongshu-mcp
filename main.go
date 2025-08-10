package main

import (
	"flag"

	"github.com/xpzouying/xiaohongshu-mcp/browser"

	"github.com/sirupsen/logrus"
)

func main() {

	var (
		headless bool
	)
	flag.BoolVar(&headless, "headless", true, "是否无头模式")
	flag.Parse()

	if err := browser.Init(headless); err != nil {
		logrus.Fatalf("failed to init browser: %v", err)
	}
	defer browser.Close()

	if err := startServer(); err != nil {
		logrus.Fatalf("failed to run server: %v", err)
	}
}
