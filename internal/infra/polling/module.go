package polling

import (
	"fmt"
	"math/rand"
	"time"
)

// Module 轮询/等待配置模块。
type Module struct {
	TimeoutMs  int
	IntervalMs int
	MaxRetries int
	Delays     map[string]int
}

func (m Module) Delay(key string) (time.Duration, error) {
	if m.Delays == nil {
		return 0, fmt.Errorf("polling delays missing")
	}
	value, ok := m.Delays[key]
	if !ok || value <= 0 {
		return 0, fmt.Errorf("polling delay missing: %s", key)
	}
	return time.Duration(value) * time.Millisecond, nil
}

func (m Module) Interval() (time.Duration, error) {
	if m.IntervalMs <= 0 {
		return 0, fmt.Errorf("polling interval missing")
	}
	return time.Duration(m.IntervalMs) * time.Millisecond, nil
}

func (m Module) Timeout() (time.Duration, error) {
	if m.TimeoutMs <= 0 {
		return 0, fmt.Errorf("polling timeout missing")
	}
	return time.Duration(m.TimeoutMs) * time.Millisecond, nil
}

func SleepDelay(m Module, key string) error {
	d, err := m.Delay(key)
	if err != nil {
		return err
	}
	time.Sleep(d)
	return nil
}

func SleepRandom(m Module, minKey, maxKey string) error {
	minDelay, err := m.Delay(minKey)
	if err != nil {
		return err
	}
	maxDelay, err := m.Delay(maxKey)
	if err != nil {
		return err
	}
	if maxDelay <= minDelay {
		time.Sleep(minDelay)
		return nil
	}
	gap := maxDelay - minDelay
	if gap <= 0 {
		time.Sleep(minDelay)
		return nil
	}
	randomDelay := minDelay + time.Duration(rand.Int63n(int64(gap)))
	time.Sleep(randomDelay)
	return nil
}
