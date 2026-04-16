# Gateway — API 网关

## 1. 概述

Gateway 是整个后端唯一暴露给前端的 HTTP 服务（端口 8888）。

> **它不包含任何业务逻辑**，只做三件事：
> 1. 接收 HTTP 请求 → 校验参数
> 2. 调用对应的 RPC 服务获取数据
> 3. 包装成统一 JSON 格式返回

---

## 2. 路由设计

### 公开接口（无需登录）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/api/dex/pairs` | 交易对列表 |
| GET | `/api/dex/pairs/:pairAddress` | 交易对详情 |
| GET | `/api/dex/pairs/:pairAddress/trades` | 成交记录 |
| GET | `/api/dex/pairs/:pairAddress/candles` | K线数据 |
| GET | `/api/dex/overview` | DEX 概览统计 |
| POST | `/api/user/auth/nonce` | 获取登录 nonce |
| POST | `/api/user/auth/login` | 钱包签名登录 |
| POST | `/api/user/auth/refresh` | 刷新 JWT token |
| GET | `/api/system/sync-status` | 链同步状态 |

### 需登录接口（JWT 鉴权）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/dex/positions` | 我的 LP 仓位 |
| POST | `/api/dex/swap/build` | 构造 Swap 交易 |
| GET | `/api/user/portfolio` | 我的资产总览 |
| GET | `/api/user/activity` | 我的操作记录 |
| PUT | `/api/user/settings` | 更新用户设置 |

---

## 3. 鉴权流程

```
1. 前端调 POST /api/user/auth/nonce { address: "0x..." }
   → 后端生成随机 nonce，存入 user_nonces 表，返回签名消息

2. 用户在钱包中签名该消息

3. 前端调 POST /api/user/auth/login { address, message, signature }
   → 后端验证签名（EIP-191），恢复出地址，与请求地址对比
   → 验证通过 → 生成 JWT (accessToken + refreshToken)

4. 后续请求携带 Header: Authorization: Bearer <accessToken>
   → 中间件解析 JWT，提取 walletAddress 注入 context
   → Handler 通过 middleware.GetWalletAddress(ctx) 获取当前用户
```

---

## 4. 统一响应格式

所有 API 返回相同的 JSON 结构：

```json
{
  "code": 0,           // 0 = 成功，非0 = 错误码
  "message": "ok",     // 人类可读的消息
  "data": { ... },     // 业务数据
  "traceId": "abc123"  // 请求追踪 ID（用于排查问题）
}
```

---

## 5. 文件结构

```
app/gateway/
├── gateway.go                      # main 入口
├── api/gateway.api                 # go-zero API 定义文件（可用 goctl 生成代码）
├── etc/gateway-dev.yaml            # 配置
└── internal/
    ├── config/config.go            # 配置结构体
    ├── svc/servicecontext.go       # 依赖注入 (Redis, RPC 客户端)
    ├── types/types.go              # 请求/响应类型定义
    └── handler/
        ├── routes.go               # ⭐ 路由注册 + 分组
        ├── dex_handler.go          # DEX 相关 handler
        └── user_handler.go         # User + System handler
```

---

## 6. routes.go 路由分组逻辑

```go
// 公开路由 — 任何人都能访问
engine.AddRoutes([]rest.Route{
    { Method: GET, Path: "/api/dex/pairs", Handler: dexGetPairsHandler(svcCtx) },
    ...
})

// 需登录路由 — 外面包了一层 authMiddleware
engine.AddRoutes([]rest.Route{
    { Method: GET, Path: "/api/dex/positions",
      Handler: authMiddleware(dexGetPositionsHandler(svcCtx)) },
    ...
})
```

`authMiddleware` 做了什么？
1. 从 `Authorization` Header 提取 JWT token
2. 验证签名和过期时间
3. 提取 `walletAddress` 放入 context
4. 如果验证失败 → 直接返回 401

---

## 7. 当前状态

**Phase 1（当前）**：Gateway Handler 返回占位数据，尚未连接到 RPC 服务。

**Phase 2（下一步）**：用 `zrpc.MustNewClient()` 初始化 RPC 客户端，Handler 中调用 RPC 获取真实数据。示例：

```go
func dexGetPairsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Phase 2: 调用 DEX RPC
        resp, err := svcCtx.DexRpc.GetPairs(r.Context(), &dex.GetPairsReq{...})
        if err != nil {
            response.Error(r.Context(), w, err)
            return
        }
        response.Success(r.Context(), w, resp)
    }
}
```
