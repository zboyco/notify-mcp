# Notify MCP 📢

一个基于 Model Context Protocol (MCP) 的 通知工具，允许 AI 发送任务完成通知。

## ✨ 功能特性

- 🤖 通过 MCP 协议与 AI 无缝集成
- 📱 向指定的 Telegram 聊天发送任务完成通知
- 🖥️ 支持多种通知渠道（Telegram、操作系统通知）
- ⚙️ 简单的配置管理
- 🚀 轻量级 Go 实现
- 🔒 安全的配置存储
- 📝 可自定义通知文案与任务标题

## 📋 系统要求


- 有效的 Telegram Bot Token
- Telegram Chat ID
- 使用操作系统通知功能时需运行在 macOS 或 Windows（Linux 暂不支持该渠道）

## 🛠️ 安装
> Go 1.23.0 或更高版本

### 使用 go install（推荐）

```bash
go install github.com/zboyco/notify-mcp/cmd/notify-mcp@latest
```

安装完成后，可执行文件将位于 `$GOPATH/bin` 或 `$HOME/go/bin` 目录下。请确保该目录已添加到您的 `PATH` 环境变量中。

### 从源码构建

```bash
git clone https://github.com/zboyco/notify-mcp.git
cd notify-mcp
go build -o notify-mcp cmd/notify-mcp/main.go
```

## ⚙️ 配置

在调用 `mcp notify` 之前，至少需要配置一种通知方式（Telegram 或操作系统通知）。可多次运行 `./notify-mcp config` 为不同渠道添加或移除配置。

### 1. 创建 Telegram Bot（可选）

1. 在 Telegram 中与 [@BotFather](https://t.me/botfather) 对话
2. 发送 `/newbot` 命令创建新机器人
3. 按照提示设置机器人名称和用户名
4. 获取 Bot Token

### 2. 获取 Chat ID（可选）

1. 将您的机器人添加到目标聊天（或私聊机器人）
2. 发送任意消息给机器人
3. 访问 `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
4. 在返回的 JSON 中找到 `chat.id` 值

### 3. 配置 Telegram 通知

```bash
# 配置 Telegram 参数
./notify-mcp config \
  --method telegram \
  --chat-id YOUR_CHAT_ID \
  --token YOUR_BOT_TOKEN \
  --api-url https://api.telegram.org # 可选，默认使用官方 API 地址
```

> 如果未提供 `--api-url`，程序会自动使用官方地址 `https://api.telegram.org`。

### 4. 配置操作系统通知

```bash
# macOS / Windows 原生通知
./notify-mcp config --method os
```

macOS 使用 Notification Center，Windows 使用 Toast 通知。Linux 暂未实现操作系统通知。

### 5. 自定义通知文案

```bash
./notify-mcp config --message "即将输出总结，请注意查收 ✅"
```

文案会附在通知末尾，未设置时会使用默认文案“即将进行汇报，请注意查看...”。

### 6. 查看或移除配置

```bash
# 查看当前启用的渠道及通知文案
./notify-mcp config

# 移除指定渠道
./notify-mcp config --method telegram --remove
```

查看命令会以 JSON 形式输出所有已配置渠道。移除时无须重复提供 Token/Chat ID，只需指定 `--method` 并附带 `--remove`。若配置被清空，工具将在启动时提示重新配置。

```bash
./notify-mcp config
```

## 🚀 使用方法

### 启动 MCP 服务

```bash
./notify-mcp
```

### 在 Claude Code 中使用

在您的 `.claude/CLAUDE.md` 中添加以下配置：

```yaml
mcpServers:
  notify-mcp:
    command: /path/to/notify-mcp
```

现在您可以在 Claude Code 中使用 `mcp notify` 工具来发送通知。


> `taskName` 会出现在通知正文中，配合配置文件中的默认文案可以快速区分不同的自动化任务。

### 参考提示词
```
- 需求不明确时，**必须** 总结汇报，禁止自作主张
- 在有多个方案的时候，**必须** 总结汇报，禁止自作主张
- 在有方案/策略需要更新时，**必须** 总结汇报，禁止自作主张
- 任何任务完成后，**必须** 总结汇报
- **当且仅当** 在`总结汇报`回应显示 **前**，调用 `mcp notify` 通知用户，流程如下：[执行任务] -> [通知用户] -> [总结汇报]
```


## 📁 项目结构

```
notify-mcp/
├── cmd/notify-mcp/          # 主程序入口
│   └── main.go
├── internal/
│   ├── config/             # 配置管理
│   │   └── config.go
│   ├── mcp/                # MCP 服务器实现
│   │   └── server.go
│   └── telegram/           # Telegram 客户端
│       └── client.go
├── go.mod
├── go.sum
└── README.md
```

## 🔧 命令行选项

### 主命令

```bash
./notify-mcp [flags]
```

- `-h, --help` - 显示帮助信息

### 配置命令

```bash
./notify-mcp config [flags]
```

- `--api-url <url>` - Telegram API 基础地址（可选，默认 `https://api.telegram.org`）
- `--chat-id <id>` - Telegram Chat ID
- `--token <token>` - Telegram Bot Token
- `--method <method>` - 要配置或移除的渠道（`telegram` / `os`）
- `--message <text>` - 自定义通知正文（附加在任务信息后）
- `--remove` - 移除指定渠道
- `-h, --help` - 显示配置命令帮助

## 📍 配置文件位置

配置文件存储在用户配置目录中：

- **macOS**: `~/Library/Application Support/notify-mcp/config.json`
- **Linux**: `~/.config/notify-mcp/config.json`
- **Windows**: `%APPDATA%\notify-mcp\config.json`

## 🔄 工作流程

1. Claude Code 执行任务
2. 任务完成前调用 `mcp notify` 工具
3. MCP 服务器向指定 Telegram 聊天发送通知
4. 用户收到通知后查看详细汇报

## 🛡️ 安全说明

- 配置文件权限设置为 `0600`，仅所有者可读写
- Bot Token 和 Chat ID 仅存储在本地配置文件中


## 🔗 相关链接

- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Telegram Bot API](https://core.telegram.org/bots/api)
