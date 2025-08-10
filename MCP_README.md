# 小红书 MCP 服务使用说明

本服务已集成 Model Context Protocol (MCP) 支持，通过 HTTP JSON-RPC 协议提供服务。

## 服务端点

- **HTTP API**: `http://localhost:18060/api/v1/*`
- **MCP 协议**: `http://localhost:18060/mcp`

## 可用的 MCP 工具

使用 Claude Code CLI 添加 HTTP 端点：

```bash
# 添加HTTP类型的MCP服务器
claude mcp add --transport http xiaohongshu-mcp http://localhost:18060/mcp
```
