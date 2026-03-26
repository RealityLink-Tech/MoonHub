# MoonHub 云目录与中继实现文档

> 状态: ✅ 核心路径已实现（独立部署服务 + 设备端客户端）  
> 更新日期: 2026-03-27

## 概述

在 **Phase 1（LAN）** 之外，提供可选的 **云目录（HTTP）+ 中继（WebSocket）** 能力：代理通过目录注册在线状态与中继 URL，对端通过 `Resolver` 在「局域网直连优先、否则云中继」策略下建立连接。

## 阅读顺序

1. [`cloud/directory/docs/README.md`](../../cloud/directory/docs/README.md) — 目录 API、签名校验、存储与缓存  
2. [`cloud/directory/docs/CONFIG.md`](../../cloud/directory/docs/CONFIG.md) — `directory-service` 参数与环境变量  
3. [`cloud/relay/docs/README.md`](../../cloud/relay/docs/README.md) — 中继鉴权与挑战应答、文本握手协议  
4. [`cloud/relay/docs/CONFIG.md`](../../cloud/relay/docs/CONFIG.md) — `relay` 命令行  
5. [`pkg/transport/docs/README.md`](../../pkg/transport/docs/README.md) — `CloudClient`、`Resolver`、`Manager`  

## 模块结构

```
cloud/directory/     # 目录 HTTP Handler、PostgreSQL、Redis 在线缓存
cmd/directory-service/
cloud/relay/         # 中继 Bridge + 目录验签
cmd/relay/
pkg/transport/       # CloudClient、Resolver（LAN > Cloud）、Agent 连接
```

## 目录服务要点

- **存储**: PostgreSQL，`agents` 表（`migration.go` 内 `MigrationSQL` 于进程启动时执行）。  
- **在线状态**: Redis，`NewRedisCache(REDIS_URL)`；TTL 默认 90s（`DefaultCacheTTL`）。  
- **安全**: 注册/心跳/删除均 Ed25519 签名；时间戳窗口 5 分钟；重复注册时用**已存储公钥**验签以防换钥。  
- **lookup**: `GET /agents/{id}` 返回 `relayEndpoint` 来自在线缓存中的 endpoint（一般为代理注册时上报的 WebSocket 地址）。

## 中继服务要点

- 升级 WebSocket 前：`Bearer agentID:base64(sig)`，sig = Ed25519 对**服务端下发的 challenge** 签名。  
- 公钥自目录 `GET /agents/{id}/pubkey` 拉取。  
- 文本命令 `CONNECT <peer>` 建立双向转发；`PING`/`PONG` 保活。

## 设备端 transport

- `CloudClient`：目录 HTTP 的注册、心跳、注销、查询、取公钥（见 `pkg/transport/cloud.go`）。  
- `Resolver.Resolve`：先查 LAN 映射，再查云目录；在线则返回 `ModeCloud` 与 `RelayEndpoint` URL（`pkg/transport/resolver.go`）。  
- `Manager.GetOrCreate`：`wsURL` 为空且已设置 `Resolver` 时自动解析（`pkg/transport/manager.go`）。

## 测试命令

```bash
go test ./cloud/directory/... ./cloud/relay/... ./pkg/transport/...
```

## 相关设计文档（仓库外可参考）

上层产品/网络设计见工作区 `docs/superpowers/specs/` 中 Phase 3 云目录与中继相关说明（若存在）。
