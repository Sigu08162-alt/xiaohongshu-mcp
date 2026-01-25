# 部署脚本问题分析和修复

## 问题描述

用户在运行 `deploy.sh` 时遇到错误：

```bash
mv: cannot stat 'xiaohongshu-login-linux-amd64': No such file or directory
```

## 根本原因

### 1. GitHub Actions 路径忽略配置

在 `.github/workflows/release.yml` 中配置了 `paths-ignore`:

```yaml
paths-ignore:
  - 'docs/**'
  - ...其他路径
```

当提交只修改了 `docs/` 目录下的文件时，release workflow 不会被触发，导致：
- Tag 被推送到 GitHub
- 但没有对应的 release 和二进制文件

### 2. 部署脚本假设

`deploy.sh` 脚本假设 release 中一定包含两个文件：
- `xiaohongshu-mcp-linux-amd64`
- `xiaohongshu-login-linux-amd64`

但在某些情况下（如手动创建的 tag），release 可能：
- 不存在
- 或者只包含部分文件（旧版本没有 login 工具）

## 修复方案

### 方案 1: 修复��署脚本（已实施）

修改 `deploy.sh` 使其更加健壮：

```bash
# 重命名文件（移除平台后缀）
mv xiaohongshu-mcp-linux-amd64 xiaohongshu-mcp

# xiaohongshu-login 可能不存在（旧版本的 release）
if [ -f "xiaohongshu-login-linux-amd64" ]; then
    mv xiaohongshu-login-linux-amd64 xiaohongshu-login
    chmod +x xiaohongshu-login
else
    echo "警告: xiaohongshu-login 不存在，跳过（可能是旧版本）"
fi

# 设置执行权限
chmod +x xiaohongshu-mcp
```

**优点**:
- 向后兼容旧版本 release
- 不会因为缺少 login 工具而失败

### 方案 2: 手动触发 Release Build

当 tag 已创建但 release 未自动构建时，可以手动触发：

```bash
# 方法1: 使用 gh CLI
gh workflow run release.yml --repo vmxmy/xiaohongshu-mcp

# 方法2: 在 GitHub 网页操作
# 访问 Actions -> Build and Release -> Run workflow
```

### 方案 3: 修改 paths-ignore 配置

如果希望任何提交都触发 release，可以修改或移除 `paths-ignore`：

```yaml
# 选项1: 移除 docs 目录限制
paths-ignore:
  - 'README.md'
  # 移除 'docs/**'

# 选项2: 只忽略特定文档
paths-ignore:
  - 'README.md'
  - 'docs/*.md'  # 只忽略 markdown 文档
  # 但保留 docs/*.go, docs/*.json 等
```

**权衡考虑**:
- 更新文档也触发 release 可能导致过多的自动发布
- 但可以确保每个 tag 都有对应的二进制文件

## 当前状态

### 已修复
✅ `deploy.sh` 已更新，兼容旧版本和缺少 login 工具的情况
✅ 已手动触发 workflow 构建最新的 release

### 待构建
🔄 GitHub Actions 正在构建新的 release:
- Tag: `v2026.01.25.1754-625d63a`
- 以及最新提交的自动 tag

## 最佳实践建议

### 对于开发者

1. **创建 release 时确保构建完成**
   ```bash
   # 创建 tag
   git tag v1.0.0
   git push origin v1.0.0

   # 等待 Actions 完成（或手动触发）
   gh workflow run release.yml

   # 确认 release 存在
   gh release view v1.0.0
   ```

2. **验证 release 包含所需文件**
   ```bash
   gh release view v1.0.0 --json assets --jq '.assets[].name'
   ```

3. **测试部署脚本**
   ```bash
   # 在干净的环境中测试
   VERSION=v1.0.0 ./deploy.sh
   ```

### 对于用户

1. **检查 release 是否存在**
   ```bash
   gh release view --repo vmxmy/xiaohongshu-mcp
   ```

2. **如果 release 不存在或不完整**
   - 等待几分钟让 Actions 完成
   - 或联系维护者手动触发构建
   - 或使用上一个稳定版本

3. **查看构建状态**
   ```bash
   gh run list --repo vmxmy/xiaohongshu-mcp --workflow=release.yml
   ```

## 时间线

- **2026-01-25 17:54** - 创建 tag `v2026.01.25.1754-625d63a`
- **2026-01-25 17:54** - 推送 tag 到 GitHub
- **2026-01-25 17:54** - 因 `paths-ignore` 配置，workflow 未触发
- **2026-01-25 20:04** - 发现问题，修复 `deploy.sh`
- **2026-01-25 20:05** - 手动触发 workflow
- **2026-01-25 20:05+** - Actions 正在构建中...

## 相关文件

- `.github/workflows/release.yml` - Release 构建配置
- `deploy.sh` - 部署脚本
- `docs/DEPLOYMENT_ISSUE.md` - 本文档

## 总结

这是一个配置问题，不是 bug。通过以下方式已解决：
1. ✅ 修复部署脚本，使其更加健壮
2. 🔄 手动触发 workflow，生成完整的 release
3. 📝 记录问题和解决方案，避免再次发生

建议未来考虑调整 `paths-ignore` 配置，或者在创建 tag 时总是手动触发 workflow。
