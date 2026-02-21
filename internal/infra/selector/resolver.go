package selector

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"gopkg.in/yaml.v3"
)

// SelectorConfig 选择器配置文件顶层结构
type SelectorConfig struct {
	Version     string                       `yaml:"version"`
	LastUpdated string                       `yaml:"last_updated"`
	Description string                       `yaml:"description"`
	Elements    map[string]*ElementSelectors `yaml:",inline"`
}

// ElementSelectors 单个元素的选择器集合
type ElementSelectors struct {
	Name       string           `yaml:"name"`
	Primary    []SelectorItem   `yaml:"primary"`
	Fallback   []SelectorItem   `yaml:"fallback"`
	TextMatch  []string         `yaml:"text_match"`
	Validation *ValidationRules `yaml:"validation"`
}

// SelectorItem 单个选择器条目
type SelectorItem struct {
	Selector    string  `yaml:"selector"`
	Version     string  `yaml:"version"`
	Confidence  float64 `yaml:"confidence"`
	Description string  `yaml:"description"`
}

// ValidationRules 元素验证规则
type ValidationRules struct {
	MustBeVisible   bool              `yaml:"must_be_visible"`
	MustBeClickable bool              `yaml:"must_be_clickable"`
	MustBeEditable  bool              `yaml:"must_be_editable"`
	TextContains    []string          `yaml:"text_contains"`
	Attributes      map[string]string `yaml:"attributes"`
}

// ResolveStats 选择器解析统计
type ResolveStats struct {
	mu      sync.Mutex
	Records map[string]*ResolveRecord `json:"records"`
}

// ResolveRecord 单个元素的解析记录
type ResolveRecord struct {
	ElementName    string    `json:"element_name"`
	LastSelector   string    `json:"last_selector"`
	LastLevel      string    `json:"last_level"`
	SuccessCount   int       `json:"success_count"`
	FailCount      int       `json:"fail_count"`
	LastResolvedAt time.Time `json:"last_resolved_at"`
}

// ElementResolver 自适应选择器解析器
type ElementResolver struct {
	config *SelectorConfig
	page   browser.Page
	stats  *ResolveStats
}

// LoadSelectorConfig 从YAML文件加载选择器配置
func LoadSelectorConfig(path string) (*SelectorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取选择器配置失败: %w", err)
	}

	var config SelectorConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析选择器配置失败: %w", err)
	}

	logrus.Infof("✓ 加载选择器配置: version=%s, 元素数=%d", config.Version, len(config.Elements))
	return &config, nil
}

// NewElementResolver 创建选择器解析器
func NewElementResolver(config *SelectorConfig, page browser.Page) *ElementResolver {
	return &ElementResolver{
		config: config,
		page:   page,
		stats: &ResolveStats{
			Records: make(map[string]*ResolveRecord),
		},
	}
}

// Resolve 按优先级解析选择器，返回第一个匹配的选择器字符串
func (r *ElementResolver) Resolve(elementName string) (string, error) {
	elemConfig, exists := r.config.Elements[elementName]
	if !exists {
		return "", fmt.Errorf("未找到元素配置: %s", elementName)
	}

	logrus.Debugf("解析选择器: %s (%s)", elementName, elemConfig.Name)

	// 第一级：Primary选择器
	for _, item := range elemConfig.Primary {
		if r.selectorExists(item.Selector) {
			r.recordSuccess(elementName, item.Selector, "primary")
			logrus.Infof("✓ [%s] 命中primary: %s", elementName, item.Selector)
			return item.Selector, nil
		}
	}

	// 第二级：Fallback选择器
	for _, item := range elemConfig.Fallback {
		if r.selectorExists(item.Selector) {
			r.recordSuccess(elementName, item.Selector, "fallback")
			logrus.Warnf("⚠ [%s] 降级到fallback: %s", elementName, item.Selector)
			return item.Selector, nil
		}
	}

	// 第三级：TextMatch（生成 :has-text 选择器）
	for _, text := range elemConfig.TextMatch {
		selector := fmt.Sprintf(`button:has-text("%s")`, text)
		if r.selectorExists(selector) {
			r.recordSuccess(elementName, selector, "text_match")
			logrus.Warnf("⚠ [%s] 降级到text_match: %s", elementName, selector)
			return selector, nil
		}
	}

	// 第四级：RuntimeDiscovery（JS探测DOM）
	selector, err := r.runtimeDiscover(elementName)
	if err == nil {
		r.recordSuccess(elementName, selector, "runtime_discovery")
		logrus.Warnf("⚠ [%s] 降级到runtime_discovery: %s", elementName, selector)
		return selector, nil
	}

	r.recordFailure(elementName)
	return "", fmt.Errorf("所有选择器均失败: %s (%s)", elementName, elemConfig.Name)
}

// runtimeDiscover 通过JS探测DOM查找元素
func (r *ElementResolver) runtimeDiscover(elementName string) (string, error) {
	logrus.Infof("🔍 [%s] 启动运行时DOM探测...", elementName)

	strategies := runtimeStrategies[elementName]
	if len(strategies) == 0 {
		return "", fmt.Errorf("无运行时探测策略: %s", elementName)
	}

	for _, s := range strategies {
		result, err := r.page.Eval(s.jsExpression)
		if err != nil {
			continue
		}
		if selector, ok := result.(string); ok && selector != "" {
			logrus.Infof("✓ [%s] 运行时发现: %s", elementName, selector)
			return selector, nil
		}
	}

	return "", fmt.Errorf("运行时探测未找到: %s", elementName)
}

type discoveryStrategy struct {
	jsExpression string
	description  string
}

// 运行时探测策略（按元素类型硬编码）
var runtimeStrategies = map[string][]discoveryStrategy{
	"publish_upload": {
		{
			jsExpression: `(() => {
				const el = document.querySelector('input[type="file"]');
				return el ? 'input[type="file"]' : '';
			})()`,
			description: "查找file类型input",
		},
	},
	"publish_title": {
		{
			jsExpression: `(() => {
				const inputs = document.querySelectorAll('input');
				for (const inp of inputs) {
					if (inp.placeholder && inp.placeholder.includes('标题')) {
						return 'input[placeholder*="标题"]';
					}
				}
				return '';
			})()`,
			description: "查找placeholder含'标题'的input",
		},
	},
	"publish_content": {
		{
			jsExpression: `(() => {
				const el = document.querySelector("[role='textbox']");
				if (el) return "[role='textbox']";
				const ce = document.querySelector("[contenteditable='true']");
				if (ce) return "[contenteditable='true']";
				return '';
			})()`,
			description: "查找富文本编辑器",
		},
	},
	"publish_submit": {
		{
			jsExpression: `(() => {
				const buttons = document.querySelectorAll('button');
				for (const btn of buttons) {
					const text = btn.innerText.trim();
					if (text === '发布') {
						return 'button:has-text("发布")';
					}
				}
				return '';
			})()`,
			description: "查找文本为'发布'的按钮",
		},
	},
	"publish_save_draft": {
		{
			jsExpression: `(() => {
				const buttons = document.querySelectorAll('button');
				for (const btn of buttons) {
					const text = btn.innerText.trim();
					if (text.includes('暂存')) {
						return 'button:has-text("暂存")';
					}
				}
				return '';
			})()`,
			description: "查找文本含'暂存'的按钮",
		},
	},
}

func (r *ElementResolver) selectorExists(selector string) bool {
	err := r.page.WaitForSelector(selector, 2*time.Second)
	return err == nil
}

func (r *ElementResolver) recordSuccess(elementName, selector, level string) {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()
	r.stats.Records[elementName] = &ResolveRecord{
		ElementName:    elementName,
		LastSelector:   selector,
		LastLevel:      level,
		SuccessCount:   r.getCount(elementName).SuccessCount + 1,
		FailCount:      r.getCount(elementName).FailCount,
		LastResolvedAt: time.Now(),
	}
}

func (r *ElementResolver) recordFailure(elementName string) {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()
	r.stats.Records[elementName] = &ResolveRecord{
		ElementName:    elementName,
		SuccessCount:   r.getCount(elementName).SuccessCount,
		FailCount:      r.getCount(elementName).FailCount + 1,
		LastResolvedAt: time.Now(),
	}
}

func (r *ElementResolver) getCount(elementName string) ResolveRecord {
	if rec, ok := r.stats.Records[elementName]; ok {
		return *rec
	}
	return ResolveRecord{}
}

// GetStats 获取解析统计
func (r *ElementResolver) GetStats() map[string]*ResolveRecord {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()
	result := make(map[string]*ResolveRecord)
	for k, v := range r.stats.Records {
		result[k] = v
	}
	return result
}
