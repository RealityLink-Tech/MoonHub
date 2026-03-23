# 设备配网系统 - 实现状态

**源码包**：`pkg/provisioning/` · **仓库文档索引**：[docs/README.md](../README.md)

## 文档阅读顺序（与仓库文档流一致）

1. 英文包级说明：[pkg/provisioning/docs/README.md](../../pkg/provisioning/docs/README.md) — 职责边界、源码地图、集成表  
2. 配置与安全：[pkg/provisioning/docs/CONFIG.md](../../pkg/provisioning/docs/CONFIG.md) — 环境变量、`provisioning.json`、`device.*` 键、HTTP 与前端令牌  
3. **本文** — 实现细节、API 表、前端目录、curl 示例、测试命令（中文深度参考）

## 实现概述

MoonHub 设备配网系统已成功实现，实现了"插电即用"的零配置设备体验。系统包含完整的后端 API 层和前端配网页面。

## 已完成的工作

### 1. 核心包结构

```
pkg/provisioning/
├── config.go          # 类型定义和配置 Key
├── events.go          # SSE 事件广播
├── network.go         # nmcli 网络操作封装
├── network_test.go    # WiFi 列表解析等单元测试
├── manager.go         # DeviceManager 核心实现
└── manager_test.go    # 单元测试
```

### 2. 核心功能

#### 2.1 设备模式状态机
- 六种设备模式：
  - `provisioning`: 等待初始 WiFi 配置（热点激活）
  - `connecting`: 正在连接配置的 WiFi
  - `onboarding`: 网络已连接，等待用户完成引导
  - `ready`: 设备完全就绪
  - `maintenance`: 维护模式
  - `error`: 遇到不可恢复的错误
- 运行时阶段：provisioning, connecting, onboarding, ready, degraded, maintenance, restarting, error

#### 2.2 WiFi 管理
- WiFi 网络扫描（nmcli）
- WiFi 连接（支持密码、隐藏网络）
- 已保存网络列表
- 网络遗忘功能
- 自动重连机制

#### 2.3 热点管理
- 自动创建热点配置文件
- 热点启用/禁用
- 默认热点 SSID：`MoonHub-XXXX`（XXXX 为设备名后4位）

#### 2.4 网络诊断
- 连接状态检测
- 互联网可达性测试
- DNS 解析测试
- 网络详细信息获取

#### 2.5 恢复机制
- 可配置的故障阈值
- 冷却时间机制
- 网络退化检测
- 自动回滚到配网模式
- **自动恢复触发**：连续网络故障达到阈值时自动触发恢复
- **恢复策略**：
  1. 尝试重连上次保存的 WiFi 网络
  2. 失败后启用热点作为后备
- **手动恢复触发**：支持通过 API 手动触发恢复流程

#### 2.6 SSE 实时事件推送
- 事件类型：status, action, system
- 事件级别：info, success, warning, error
- 动作阶段：started, completed, failed
- 最近事件缓存（25条）

#### 2.7 恢复出厂设置
- 清除所有已保存的 WiFi 网络
- 清除所有设备配置（device.*）
- 清除引导状态（onboarding.*）
- 启用热点并返回配网模式
- 通过 SSE 事件通知恢复进度

#### 2.8 授权码管理
- 6 位数字授权码（`crypto/rand` 均匀随机）
- 设备标识生成（YSHU-YYYY-XXX 格式）
- 授权码重新生成
- 用于 PWA 连接设备时的身份验证

### 3. 前端配网页面

#### 3.1 文件结构

```
web/frontend/src/
├── api/provisioning.ts              # API 客户端
├── hooks/use-provisioning-sse.ts    # SSE 事件订阅 Hook
├── components/provisioning/
│   ├── layout.tsx                   # 配网页面布局
│   ├── bottom-nav.tsx               # 底部四步骤导航
│   ├── wifi-selector.tsx            # WiFi 网络选择器
│   ├── auth-code.tsx                # 授权码显示组件
│   └── factory-reset-dialog.tsx     # 恢复出厂设置对话框
└── routes/provisioning/
    ├── route.tsx                    # 配网路由布局
    ├── index.tsx                    # WiFi 配置页（步骤1）
    ├── auth.tsx                     # 授权码页面（步骤2）
    ├── install.tsx                  # PWA 安装引导（步骤3）
    └── settings.tsx                 # 设备设置页（步骤4）
```

#### 3.2 Service Worker 离线支持

```
web/frontend/public/
├── sw.js                            # Service Worker
├── site.webmanifest                 # PWA 配置（配网范围）
└── fonts/
    ├── manrope/
    │   ├── manrope.css              # Manrope 字体 CSS
    │   └── Manrope-VariableFont_wght.woff2  # 字体文件（需下载）
    └── material-symbols/
        ├── material-symbols.css     # Material Symbols CSS
        └── MaterialSymbolsOutlined.woff2    # 字体文件（需下载）
```

Service Worker 功能：
- 缓存配网页面静态资源
- 离线时从缓存提供服务
- API 调用不走缓存（实时性要求）

#### 3.3 用户流程

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  WiFi 配置页    │ ──► │  授权码页面     │ ──► │  PWA 安装引导   │ ──► │  设备设置页     │
│  /provisioning  │     │  /provisioning/ │     │  /provisioning/ │     │  /provisioning/ │
│                 │     │  auth           │     │  install        │     │  settings       │
│ - 扫描网络      │     │ - 显示6位授权码 │     │ - iOS 指引      │     │ - 设备状态      │
│ - 选择网络      │     │ - 复制按钮      │     │ - Android 指引  │     │ - 网络恢复      │
│ - 输入密码      │     │ - 设备标识      │     │ - 安装按钮      │     │ - 恢复出厂设置  │
│ - 开始配网      │     │ - 下一步按钮    │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘     └─────────────────┘
```

#### 3.4 设计系统
- 主色调：`#506070`（深蓝灰）
- 月晕光晕效果：`box-shadow: 0 0 60px 10px rgba(212, 228, 247, 0.4)`
- 玻璃态背景：`backdrop-filter: blur(12px)`
- Material Symbols 图标
- Manrope 字体

### 4. 集成点

#### 4.1 web/backend/api/provisioning.go
- HTTP API 处理器
- SSE 事件流处理
- 请求验证和响应格式化
- 授权码端点处理

#### 4.2 web/backend/api/router.go
- 添加 `provisioning` 字段
- 添加 `SetProvisioningHandler()` 方法
- 自动注册配网路由

#### 4.3 web/backend/main.go
- 环境变量控制启用/禁用
- **在 `RegisterRoutes` 之前**调用 `SetProvisioningHandler`，确保 `/api/provisioning/*` 已挂载
- 设备管理器初始化与后台监控循环
- 中间件顺序：**先** `IPAllowlist`（若配置了 CIDR），**再** `ProvisioningAuth`；后者在非空 `MOONHUB_PROVISIONING_TOKEN` 下校验 `Authorization: Bearer …` 或 `X-MoonHub-Provisioning-Token`
- 若 `MOONHUB_PROVISIONING_ENABLED=1` 且对外监听（`-public` 或 launcher 配置）但未设置 token，启动时打印告警

#### 4.4 环境变量与安全

| 变量 | 说明 |
|------|------|
| `MOONHUB_PROVISIONING_ENABLED` | 设为 `1` 时启用配网管理器与 API |
| `MOONHUB_ALLOW_SYSTEM_CONTROL` | 设为 `1` 时允许通过 API 触发真实 `systemctl` 重启等 |
| `MOONHUB_PROVISIONING_TOKEN` | 非空时，所有 `/api/provisioning/*` 请求须携带与之一致的 Bearer 或 `X-MoonHub-Provisioning-Token`；**强烈建议在公网/局域网对外暴露监听时设置** |

**设备端 `provisioning.json`**：由 `JSONConfigStore` 写入，文件权限 `0600`，避免同机其他用户读取热点 PSK、授权码等。

**前端令牌（浏览器）**：

- **不要**把 `MOONHUB_PROVISIONING_TOKEN` 写进 Vite 构建环境变量或仓库。
- 当接口返回 401 时，配网布局会提示输入令牌；用户输入后保存到 **`sessionStorage`** 键名 `moonhub.provisioningToken`，并刷新页面。
- 若已配置服务端 token，SSE 使用带 `Authorization` 的 `fetch` 流式读取（标准 `EventSource` 无法自定义头）。

### 5. API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/provisioning/status` | 获取设备状态 |
| GET | `/api/provisioning/events` | SSE 事件流 |
| GET | `/api/provisioning/networks` | 扫描 WiFi 网络 |
| GET | `/api/provisioning/networks/saved` | 已保存网络列表 |
| POST | `/api/provisioning/network/connect` | 连接 WiFi |
| POST | `/api/provisioning/network/forget` | 遗忘网络 |
| GET | `/api/provisioning/network/diagnostics` | 网络诊断 |
| POST | `/api/provisioning/network/test` | 测试互联网连接 |
| POST | `/api/provisioning/hotspot/enable` | 启用热点 |
| POST | `/api/provisioning/hotspot/disable` | 禁用热点 |
| POST | `/api/provisioning/restart` | 请求重启 |
| POST | `/api/provisioning/rollback` | 回滚到配网模式 |
| GET | `/api/provisioning/auth-code` | 获取授权码 |
| POST | `/api/provisioning/auth-code/regenerate` | 重新生成授权码 |
| POST | `/api/provisioning/recovery/trigger` | 手动触发网络恢复 |
| POST | `/api/provisioning/factory-reset` | 恢复出厂设置 |

### 6. 配置选项

#### 环境变量
```bash
MOONHUB_PROVISIONING_ENABLED=1    # 启用配网系统
MOONHUB_ALLOW_SYSTEM_CONTROL=1    # 允许系统控制（重启等）
```

#### 配置存储（provisioning.json）
```json
{
  "device.mode": "provisioning",
  "device.network.provisioned": false,
  "device.network.lastSsid": "",
  "device.network.hotspotSsid": "MoonHub-XXXX",
  "device.network.hotspotProfile": "MoonHub Hotspot",
  "device.network.hotspotPassword": "MoonHub-XXXX-wifi",
  "device.network.interfaceName": "wlan0",
  "device.network.apEnabled": true,
  "device.auth.code": "882966",
  "device.auth.deviceId": "YSHU-2024-OX2",
  "device.recovery.enabled": true,
  "device.recovery.failureThreshold": 3,
  "device.recovery.cooldownMs": 120000
}
```

### 7. systemd 服务

```ini
[Unit]
Description=MoonHub AI Assistant
After=network-online.target NetworkManager.service
Wants=network-online.target

[Service]
Type=simple
Environment=MOONHUB_PROVISIONING_ENABLED=1
Environment=MOONHUB_ALLOW_SYSTEM_CONTROL=1
ExecStart=/usr/local/bin/moonhub-web -public /etc/moonhub/config.json
Restart=always

[Install]
WantedBy=multi-user.target
```

## 使用示例

### 获取设备状态

```bash
curl http://localhost:18800/api/provisioning/status
```

响应：
```json
{
  "status": {
    "mode": "provisioning",
    "hostname": "moonhub-device",
    "network": {
      "provisioned": false,
      "connected": false,
      "hotspotSsid": "MoonHub-VICE",
      "apEnabled": true
    },
    "runtime": {
      "phase": "provisioning",
      "health": "ok",
      "summary": "Waiting for initial WiFi provisioning."
    }
  }
}
```

### 扫描 WiFi 网络

```bash
curl http://localhost:18800/api/provisioning/networks
```

### 连接 WiFi

```bash
curl -X POST http://localhost:18800/api/provisioning/network/connect \
  -H "Content-Type: application/json" \
  -d '{"ssid": "MyNetwork", "password": "mypassword"}'
```

### 获取授权码

```bash
curl http://localhost:18800/api/provisioning/auth-code
```

响应：
```json
{
  "code": "882966",
  "deviceId": "YSHU-2024-OX2"
}
```

### SSE 事件流

```javascript
const eventSource = new EventSource('/api/provisioning/events');
eventSource.addEventListener('device', (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data);
});
```

### 访问配网页面

```
http://localhost:18800/provisioning
```

### 触发网络恢复

```bash
curl -X POST http://localhost:18800/api/provisioning/recovery/trigger
```

响应：
```json
{
  "ok": true
}
```

### 恢复出厂设置

```bash
curl -X POST http://localhost:18800/api/provisioning/factory-reset
```

响应：
```json
{
  "ok": true
}
```

## 测试覆盖

### 单元测试
- `TestNewDeviceManager` - 设备管理器创建
- `TestNormalizeMode` - 模式规范化
- `TestSanitizeHotspotSuffix` - 热点 SSID 后缀生成
- `TestDefaultHotspotSSID` - 默认热点 SSID 生成
- `TestGetStatus` - 状态获取
- `TestListWifiNetworks` - WiFi 网络列表
- `TestEventBroadcaster` - 事件广播
- `TestConfigStore` - 配置存储
- `TestDeriveRuntimeState` - 运行时状态派生

### 运行测试

```bash
go test ./pkg/provisioning/... -v
```

## 文件清单

### 后端新建文件
- `pkg/provisioning/config.go`
- `pkg/provisioning/events.go`
- `pkg/provisioning/network.go`
- `pkg/provisioning/manager.go`
- `pkg/provisioning/manager_test.go`
- `web/backend/api/provisioning.go`
- `deployment/systemd/moonhub.service`

### 前端新建文件
- `web/frontend/src/api/provisioning.ts`
- `web/frontend/src/hooks/use-provisioning-sse.ts`
- `web/frontend/src/components/provisioning/layout.tsx`
- `web/frontend/src/components/provisioning/bottom-nav.tsx`
- `web/frontend/src/components/provisioning/wifi-selector.tsx`
- `web/frontend/src/components/provisioning/auth-code.tsx`
- `web/frontend/src/components/provisioning/factory-reset-dialog.tsx`
- `web/frontend/src/routes/provisioning/route.tsx`
- `web/frontend/src/routes/provisioning/index.tsx`
- `web/frontend/src/routes/provisioning/auth.tsx`
- `web/frontend/src/routes/provisioning/install.tsx`
- `web/frontend/src/routes/provisioning/settings.tsx`
- `web/frontend/public/sw.js`
- `web/frontend/public/fonts/manrope/manrope.css`
- `web/frontend/public/fonts/material-symbols/material-symbols.css`

### 修改文件
- `web/backend/api/router.go` - 添加 provisioning 字段和 SetProvisioningHandler 方法
- `web/backend/main.go` - 添加设备管理器初始化代码
- `web/frontend/index.html` - 添加 Material Symbols 和 Manrope 字体（支持本地字体）
- `web/frontend/src/main.tsx` - 添加 Service Worker 注册
- `web/frontend/public/site.webmanifest` - 更新 PWA 配置

## 参考实现

设计参考：
- `web/frontend/public/provisioning/wifi/index.html` - WiFi 配置页设计稿
- `web/frontend/public/provisioning/auth/index.html` - 授权码页设计稿
- `web/frontend/public/provisioning/pwa/index.html` - PWA 安装页设计稿

## 后续工作

1. **PWA 应用** - 完整的 PWA 应用（扫描设备、输入授权码、调用设备能力）
2. ~~**Service Worker** - 离线支持~~ ✅ 已完成
3. ~~**网络恢复增强** - 可选的自动恢复机制优化~~ ✅ 已完成
4. ~~**Factory Reset** - 恢复出厂设置功能~~ ✅ 已完成

## 离线支持说明

Service Worker 已实现，配网页面可在设备热点环境下离线可用。字体文件需要手动下载：

1. **Manrope 字体**
   - 下载地址：https://fonts.google.com/specimen/Manrope
   - 保存为：`web/frontend/public/fonts/manrope/Manrope-VariableFont_wght.woff2`

2. **Material Symbols 字体**
   - 下载地址：https://fonts.google.com/specimen/Material+Symbols+Outlined
   - 保存为：`web/frontend/public/fonts/material-symbols/MaterialSymbolsOutlined.woff2`

如未下载字体文件，系统会自动回退到 Google Fonts CDN（需要互联网连接）。
