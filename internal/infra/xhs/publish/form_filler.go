package publish

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

// inputTags 在内容编辑器中输入标签
func inputTags(page browser.Page, tags []string, polling PollingModule) error {
	if len(tags) == 0 {
		return nil
	}

	contentElem, err := page.Element("[role=\"textbox\"]")
	if err != nil {
		return fmt.Errorf("未找到内容编辑器: %w", err)
	}

	if err := sleepDelay(polling, "tag_editor_ready_ms"); err != nil {
		return err
	}

	for i := 0; i < 20; i++ {
		contentElem.Press("ArrowDown")
		if err := sleepDelay(polling, "tag_arrow_step_ms"); err != nil {
			return err
		}
	}

	contentElem.Press("Enter")
	contentElem.Press("Enter")

	if err := sleepDelay(polling, "tag_after_enter_ms"); err != nil {
		return err
	}

	for i, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		logrus.Infof("  [%d/%d] 输入标签: #%s", i+1, len(tags), tag)
		if err := inputTag(page, contentElem, tag, polling); err != nil {
			logrus.Warnf("  ⚠️ 标签输入失败: %v", err)
		}
	}

	return nil
}

// inputTag 输入单个标签
func inputTag(page browser.Page, contentElem browser.Element, tag string, polling PollingModule) error {
	contentElem.Input("#")
	if err := sleepDelay(polling, "tag_hash_delay_ms"); err != nil {
		return err
	}

	for _, char := range tag {
		contentElem.Input(string(char))
		if err := sleepDelay(polling, "tag_char_delay_ms"); err != nil {
			return err
		}
	}

	if err := sleepDelay(polling, "tag_after_text_ms"); err != nil {
		return err
	}

	topicContainer, err := page.Element("#creator-editor-topic-container")
	if err == nil && topicContainer != nil {
		firstItem, err := topicContainer.Element(".item")
		if err == nil && firstItem != nil {
			firstItem.Click()
			logrus.Infof("    ✅ 成功点击标签联想选项")
			if err := sleepDelay(polling, "tag_suggestion_click_ms"); err != nil {
				return err
			}
		} else {
			logrus.Warnf("    ⚠️ 未找到标签联想选项，直接输入空格")
			contentElem.Input(" ")
		}
	} else {
		logrus.Warnf("    ⚠️ 未找到标签联想下拉框，直接输入空格")
		contentElem.Input(" ")
	}

	if err := sleepDelay(polling, "tag_after_tag_ms"); err != nil {
		return err
	}
	return nil
}
