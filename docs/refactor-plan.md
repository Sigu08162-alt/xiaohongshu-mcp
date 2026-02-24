# Refactor Plan: Clean Architecture Optimization

## Context
Project: xiaohongshu-mcp
Path: /home/dev/.nanobot/workspace/github/xiaohongshu-mcp
Language: Go

## Phase 1: Unify Architecture Boundaries

### Task 1.1 — Supplement Port Interfaces
**File**: `internal/app/ports/ports.go`

Add missing port interfaces:
- `FeedGateway` — ListFeeds, GetFeedDetail, DeleteFeed, ShareFeed
- `InteractionGateway` — LikeFeed, UnlikeFeed, PostComment, DeleteComment, LikeComment, ReplyComment, FavoriteFeed
- `UserGateway` — FollowUser, UnfollowUser, GetUserProfile, GetMyStats
- `AnalyticsGateway` — GetContentAnalytics, GetFanAnalytics
- `LoginGateway` — CheckLoginStatus, GetLoginQRCode

Each interface method must accept `context.Context` as first param.

---

### Task 1.2 — Migrate xiaohongshu/ Actions to Infra Gateways
**Scope**: `xiaohongshu/` package → `internal/infra/xhs/`

For each action file in `xiaohongshu/`:
- `feeds.go` → `internal/infra/xhs/feed/gateway.go`
- `like.go` → `internal/infra/xhs/interaction/gateway.go`
- `follow.go` → `internal/infra/xhs/user/gateway.go`
- `comment.go` → `internal/infra/xhs/interaction/gateway.go`
- `data.go` → `internal/infra/xhs/analytics/gateway.go`
- `login.go` → `internal/infra/xhs/login/gateway.go`

Each gateway must implement the corresponding Port interface from Task 1.1.
Keep `xiaohongshu/` package as thin wrappers or delete after migration.

---

### Task 1.3 — Move wiring_bootstrap.go to internal/interfaces/wiring/
**Files**:
- `wiring_bootstrap.go` (main package) → `internal/interfaces/wiring/bootstrap.go`
- `main.go` should only call `wiring.Bootstrap()` and start servers

Move these functions:
- `buildPublishUsecase`
- `loadPublishUsecase`
- `extractPublishSelectorsFromCollected`
- All YAML parsing and selector extraction logic

---

## Phase 2: App Layer Usecases

### Task 2.1 — Create Feed Usecase
**File**: `internal/app/feed/usecase.go`

Implement:
- `ListFeeds(ctx) ([]Feed, error)`
- `GetFeedDetail(ctx, feedID, xsecToken) (*FeedDetail, error)`
- `DeleteFeed(ctx, feedID, xsecToken) error`
- `ShareFeed(ctx, feedID, xsecToken) (string, error)`

Inject `FeedGateway` port via constructor.

---

### Task 2.2 — Create Interaction Usecase
**File**: `internal/app/interaction/usecase.go`

Implement:
- `LikeFeed / UnlikeFeed`
- `FavoriteFeed / UnfavoriteFeed`
- `PostComment / DeleteComment / LikeComment / ReplyComment`

Inject `InteractionGateway` port via constructor.

---

### Task 2.3 — Create User & Analytics Usecases
**Files**:
- `internal/app/user/usecase.go`
- `internal/app/analytics/usecase.go`

User: FollowUser, UnfollowUser, GetUserProfile, GetMyStats
Analytics: GetContentAnalytics, GetFanAnalytics

---

## Phase 3: Domain & Infrastructure Quality

### Task 3.1 — Enrich Domain Layer
**File**: `internal/domain/publish/content.go`

- Make `ImageContent` and `VideoContent` self-validating (add `Validate()` methods)
- Extract `Tags` as value object with dedup logic
- Extract `ScheduleTime` as value object with range validation (1h~14d)
- Move validation rules from `service.go` and `mcp_handlers.go` into domain

---

### Task 3.2 — Split gateway.go into focused files
**Scope**: `internal/infra/xhs/publish/`

Split `gateway.go` (800+ lines) into:
- `gateway.go` — orchestration only (~100 lines)
- `uploader.go` — image/video upload logic
- `form_filler.go` — title/content/tags form filling
- `location.go` — location marker setting
- `waiter.go` — polling and wait-for-selector helpers

---

### Task 3.3 — Fix withBrowserPage: inject engine, add context
**Files**: `service.go`, `browser_factory.go`

- Convert `withBrowserPage` from package-level function to `XiaohongshuService` method
- Add `context.Context` parameter
- Inject browser engine factory via `XiaohongshuService` constructor
- Add browser instance reuse (singleton with mutex)

---

### Task 3.4 — Replace time.Sleep with WaitForSelector
**Scope**: `internal/infra/xhs/publish/gateway.go` and `xiaohongshu/` package

- Replace all `time.Sleep(N * time.Second)` with `page.WaitForSelector(selector, timeout)`
- Where no selector is available, use `page.WaitForLoadState`
- Keep only necessary minimal sleeps (e.g., anti-bot jitter: 200-500ms random)

---

### Task 3.5 — Unify logging to slog, fix handler field semantics
**Scope**: entire project

- Replace all `logrus` usage with `log/slog`
- Fix `handleFollowUser` returning `FeedID: userID` → use correct field
- Remove dead code: `processImages` deprecated function in `service.go`
- Fix `findPublishPage` anonymous struct → named type `CollectedPage`

---

## Phase 4: Resilience & Safety

### Task 4.1 — Add rate limiting / anti-detection jitter
**File**: `internal/infra/xhs/ratelimit/limiter.go` (new)

- Create `RateLimiter` with configurable min/max delay
- Apply to all interaction operations (like, comment, follow)
- Use `rand.Intn` jitter: 500ms~1500ms between operations
- Add per-operation cooldown config via environment variable

---

### Task 4.2 — Update User-Agent to current Chrome
**File**: `internal/infra/browser/playwright/engine.go`

- Read UA from `BROWSER_USER_AGENT` env var
- Default to current Chrome 132 UA string
- Add UA rotation list (optional, configurable)

---
