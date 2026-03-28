# Built-in Plugin Index

Paths are `pkg/plugins/.../plugin.go` (package name is typically the directory name, e.g., `telegram`, `openai_compat`).

## Channel (16)

| Directory | Purpose |
|-----------|---------|
| `channels/telegram` | Telegram |
| `channels/discord` | Discord |
| `channels/slack` | Slack |
| `channels/matrix` | Matrix |
| `channels/feishu` | Feishu (Lark) |
| `channels/qq` | QQ |
| `channels/dingtalk` | DingTalk |
| `channels/line` | LINE |
| `channels/onebot` | OneBot |
| `channels/wecom` | WeCom (Enterprise WeChat) |
| `channels/wecom_app` | WeCom App |
| `channels/wecom_aibot` | WeCom AI Bot |
| `channels/irc` | IRC |
| `channels/maixcam` | MaixCam |
| `channels/whatsapp` | WhatsApp (Bridge) |
| `channels/whatsapp_native` | WhatsApp Native |

## Provider (8)

| Directory | Description |
|-----------|-------------|
| `providers/openai_compat` | OpenAI-compatible HTTP and multi-protocol |
| `providers/openai_oauth` | OpenAI OAuth scenarios |
| `providers/anthropic` | Anthropic |
| `providers/anthropic_messages` | Anthropic Messages |
| `providers/antigravity` | Antigravity |
| `providers/claude_cli` | Claude CLI |
| `providers/codex_cli` | Codex CLI |
| `providers/github_copilot` | GitHub Copilot |

## Tool (2)

| Directory | Description |
|-----------|-------------|
| `tools/web` | Web search / fetch etc. |
| `tools/message` | Message tool |

> If this file is out of sync with the repository, refer to actual directories under `pkg/plugins` and update blank imports in `helpers.go` accordingly.
