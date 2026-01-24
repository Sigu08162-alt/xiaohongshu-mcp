# 私有仓库下载说明

## 重要提示

本仓库为**私有仓库**，使用 `wget` 或 `curl` **无法**下载 Release 文件。

必须使用 **gh CLI** 进行下载。

## 正确的下载方法

### 1. 安装并登录 gh CLI

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install gh

# CentOS/RHEL
sudo dnf install gh

# macOS
brew install gh

# 登录 GitHub
gh auth login
```

### 2. 下载 Release 文件

#### 下载最新版本
```bash
# 自动检测系统和架构
gh release download --repo vmxmy/xiaohongshu-mcp --pattern "xiaohongshu-mcp-linux-amd64"

# 或指定版本
gh release download v2026.01.24.2233-91ae7a1 \
  --repo vmxmy/xiaohongshu-mcp \
  --pattern "xiaohongshu-mcp-linux-amd64"
```

#### 下载所有平台版本
```bash
# 下载最新版本的所有文件
gh release download --repo vmxmy/xiaohongshu-mcp
```

### 3. 使用一键部署脚本（推荐）

```bash
# 下载部署脚本
wget https://raw.githubusercontent.com/vmxmy/xiaohongshu-mcp/main/deploy-onestop.sh
chmod +x deploy-onestop.sh

# 运行（会自动使用 gh CLI 下载）
./deploy-onestop.sh
```

## 错误示例（不要使用）

❌ **这些命令对私有仓库无效**:

```bash
# wget - 私有仓库会返回 404
wget https://github.com/vmxmy/xiaohongshu-mcp/releases/download/xxx

# curl - 私有仓库需要认证
curl -L https://github.com/vmxmy/xiaohongshu-mcp/releases/download/xxx
```

## 为什么必须用 gh CLI？

1. **认证**: gh CLI 自动处理 GitHub 认证
2. **私有访问**: 支持下载私有仓库的 Release
3. **简单**: 不需要手动管理 token

## 其他选项

如果确实需要使用 curl/wget，可以：

### 方法 1: 使用 Personal Access Token

```bash
# 创建 token: https://github.com/settings/tokens
# 需要 repo 权限

# 下载
curl -L -H "Authorization: token YOUR_TOKEN" \
  https://github.com/vmxmy/xiaohongshu-mcp/releases/download/VERSION/FILE \
  -o xiaohongshu-mcp
```

### 方法 2: 将仓库设为公开

如果不需要保密，可以在 GitHub 设置中将仓库改为 Public。

## 推荐做法

✅ **使用 deploy-onestop.sh 一键脚本**
- 自动检查 gh CLI
- 自动下载最新版本
- 自动配置和启动服务

```bash
cd /home/dev/app/xiaohongshu-mcp
wget https://raw.githubusercontent.com/vmxmy/xiaohongshu-mcp/main/deploy-onestop.sh
chmod +x deploy-onestop.sh
./deploy-onestop.sh
```
