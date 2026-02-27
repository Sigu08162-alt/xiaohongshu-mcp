package publish

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type selectorProbe interface {
	Eval(expression string, args ...interface{}) (interface{}, error)
}

func waitForSelectorAttached(page selectorProbe, selector string, timeout, interval time.Duration) error {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return errors.New("selector is empty")
	}
	if timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if interval <= 0 {
		return errors.New("interval must be positive")
	}

	deadline := time.Now().Add(timeout)
	for {
		v, err := page.Eval(`(sel) => !!document.querySelector(sel)`, selector)
		if err == nil {
			if exists, ok := v.(bool); ok && exists {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for attached selector: %s", selector)
		}
		time.Sleep(interval)
	}
}
