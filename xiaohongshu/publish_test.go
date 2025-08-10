package xiaohongshu

import (
	"context"
	"testing"

	"github.com/xpzouying/xiaohongshu-mcp/browser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublish(t *testing.T) {

	t.Skip("SKIP: 测试发布")

	_ = browser.Init(false)
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	action, err := NewPublishImageAction(page)
	require.NoError(t, err)

	err = action.Publish(context.Background(), PublishImageContent{
		Title:      "Claude移动端重大更新！随时随地高效办公",
		Content:    "Claude移动端重大更新！随时随地高效办公",
		ImagePaths: []string{"1948784311265894447"},
	})
	assert.NoError(t, err)
}
