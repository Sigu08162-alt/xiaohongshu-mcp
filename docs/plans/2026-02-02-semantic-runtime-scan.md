# Semantic Runtime Scan Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 引入运行时语义驱动（DOM 指纹 + 可点击性驱动），提高 SPA 页面采集成功率。

**Architecture:** 新增语义配置、指纹计算、点击计划与验证回退模块；refresh_selectors 与 collect_metadata 接入语义驱动；记录语义轨迹与失败报告。

**Tech Stack:** Go、Playwright、YAML 配置

### Task 1: 语义配置结构与样例

**Files:**
- Create: `configs/semantic_scan.yaml`
- Test: `cmd/refresh_selectors/main_test.go`

**Step 1: Write the failing test**

```go
tfunc TestLoadSemanticConfig_Defaults(t *testing.T) {
    _, err := loadSemanticConfig("configs/semantic_scan.yaml")
    if err != nil {
        t.Fatalf("expected config to load, got %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/refresh_selectors -run TestLoadSemanticConfig_Defaults`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
func loadSemanticConfig(path string) (*SemanticConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg SemanticConfig
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/refresh_selectors -run TestLoadSemanticConfig_Defaults`
Expected: PASS

**Step 5: Commit**

```bash
git add configs/semantic_scan.yaml cmd/refresh_selectors/main_test.go

git commit -m "feat: add semantic scan config loader"
```

### Task 2: DOM 指纹计算

**Files:**
- Create: `pkg/semantic/fingerprint.go`
- Test: `pkg/semantic/fingerprint_test.go`

**Step 1: Write the failing test**

```go
func TestFingerprint_DeltaAboveThreshold(t *testing.T) {
    a := Fingerprint{VisibleCount: 100}
    b := Fingerprint{VisibleCount: 160}
    if !a.Delta(b).Above(0.5) {
        t.Fatalf("expected delta above threshold")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/semantic -run TestFingerprint_DeltaAboveThreshold`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
func (f Fingerprint) Delta(other Fingerprint) FingerprintDelta {
    return FingerprintDelta{VisibleRatio: float64(other.VisibleCount-f.VisibleCount) / float64(f.VisibleCount)}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/semantic -run TestFingerprint_DeltaAboveThreshold`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/semantic/fingerprint.go pkg/semantic/fingerprint_test.go

git commit -m "feat: add semantic fingerprint delta"
```

### Task 3: 点击计划与可点击性检测

**Files:**
- Create: `pkg/semantic/click_plan.go`
- Test: `pkg/semantic/click_plan_test.go`

**Step 1: Write the failing test**

```go
func TestClickPlan_OrderTargets(t *testing.T) {
    plan := BuildClickPlan([]string{"a", "b"})
    if plan[0] != "a" {
        t.Fatalf("expected first target to be a")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/semantic -run TestClickPlan_OrderTargets`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
func BuildClickPlan(targets []string) []string { return targets }
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/semantic -run TestClickPlan_OrderTargets`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/semantic/click_plan.go pkg/semantic/click_plan_test.go

git commit -m "feat: add semantic click plan"
```
