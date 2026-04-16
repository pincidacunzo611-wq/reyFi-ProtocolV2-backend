# 01 — 整体架构设计

> 本文档用最通俗的语言，讲清楚整个后端系统是怎么运转的。

---

## 一、一张图看懂整个系统

```
┌───────────────────────────────────────────────────────────────────────┐
│                       用户 / 前端 (Next.js)                            │
└─────────────────────────────┬─────────────────────────────────────────┘
                              │ HTTPS 请求
                              ▼
┌───────────────────────────────────────────────────────────────────────┐
│                    API Gateway (go-zero API 层)                        │
│                                                                       │
│  统一入口，所有前端请求都走这里                                           │
│  职责：路由分发 / JWT 鉴权 / 限流 / 日志 / 跨域 / 熔断                    │
│                                                                       │
│  /api/dex/*        →  DEX 服务                                        │
│  /api/lending/*    →  借贷服务                                         │
│  /api/futures/*    →  期货服务                                         │
│  /api/options/*    →  期权服务                                         │
│  /api/vault/*      →  金库服务                                         │
│  /api/bonds/*      →  债券服务                                         │
│  /api/governance/* →  治理服务                                         │
│  /api/user/*       →  用户服务                                         │
│  /api/system/*     →  系统运维（内部）                                   │
└───────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────────────┘
        │      │      │      │      │      │      │      │
        ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼
    ┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐
    │ DEX  ││Lend  ││Futur ││Option││Vault ││Bonds ││Gover ││ User │ ← RPC 服务层
    │ RPC  ││ RPC  ││ RPC  ││ RPC  ││ RPC  ││ RPC  ││ RPC  ││ RPC  │
    └──┬───┘└──┬───┘└──┬───┘└──┬───┘└──┬───┘└──┬───┘└──┬───┘└──┬───┘
       │       │       │       │       │       │       │       │
       └───────┴───────┴───┬───┴───────┴───────┴───────┴───────┘
                           │
             ┌─────────────┼─────────────┐
             ▼             ▼             ▼
        ┌─────────┐  ┌─────────┐  ┌────────────┐
        │  MySQL  │  │  Redis  │  │   Kafka    │
        │  数据库  │  │   缓存  │  │  消息队列   │
        └─────────┘  └─────────┘  └──────┬─────┘
                                         │
                                         ▼
                                  ┌────────────┐
                                  │Chain Indexer│ ← 链上事件索引器
                                  │ 监听链上事件  │
                                  │  写入 Kafka  │
                                  └──────┬─────┘
                                         │
                                         ▼
                                  ┌────────────┐
                                  │  区块链节点  │
                                  │  (ETH RPC)  │
                                  └────────────┘
```

---

## 二、三层架构详解

### 第一层：API Gateway（API 网关层）

**这是什么？**
> 你可以把它想成一个"前台接待"。所有前端的请求都先到这里，然后它帮你转接到正确的后台部门。

**它做什么？**

| 职责 | 举例 |
|------|------|
| 路由分发 | 前端请求 `/api/dex/pairs` → 转发给 DEX 服务处理 |
| 身份验证 | 检查请求头里的 JWT Token 是否有效（钱包签名登录） |
| 限流保护 | 同一个 IP 每秒最多请求 100 次，防止被恶意刷 |
| 熔断降级 | 某个下游 RPC 挂了，快速返回错误，不让整个网关卡住 |
| 日志记录 | 记录每个请求的耗时、状态码，方便排查问题 |
| 跨域处理 | 允许前端从 `app.reyfi.io` 访问 `api.reyfi.io` |
| 链路追踪 | 自动注入 Trace ID，方便追踪一个请求在多个服务间的流转 |

**go-zero 怎么实现？**
在 go-zero 里，API 网关就是一个 `.api` 文件定义路由，然后用 `goctl` 自动生成代码。

```go
// gateway.api — 定义路由规则
syntax = "v1"

// 定义请求/响应结构
type (
    GetPairsReq {
        Page     int64 `form:"page,default=1"`
        PageSize int64 `form:"pageSize,default=20"`
    }
    GetPairsResp {
        List  []PairInfo `json:"list"`
        Total int64      `json:"total"`
    }
)

// 定义路由
@server(
    prefix: /api/dex
    group:  dex
)
service gateway-api {
    @handler GetPairs
    get /pairs (GetPairsReq) returns (GetPairsResp)
}
```

运行 `goctl api go -api gateway.api -dir .` 就自动生成代码！

---

### 第二层：RPC 服务层（业务逻辑层）

**这是什么？**
> 每个模块有一个独立的 RPC 服务，就像公司里不同的"部门"。DEX 部门只管交易，借贷部门只管存取。

**为什么要拆成多个服务？**

| 好处 | 说明 |
|------|------|
| 独立部署 | DEX 服务挂了不影响借贷服务 |
| 独立扩容 | 期货交易量大？只加期货服务器就行 |
| 代码隔离 | 每个服务只关心自己的业务，代码清晰 |
| 团队协作 | 不同人开发不同服务，互不干扰 |
| 独立发布 | 修一个模块的 bug 不需要重新部署全部服务 |

**go-zero 怎么实现？**
每个 RPC 服务用 `.proto` 文件定义接口：

```protobuf
// dex.proto
syntax = "proto3";
package dex;

option go_package = "./dex";

message GetPairsReq {
    int64 page = 1;
    int64 page_size = 2;
}

message PairInfo {
    string pair_address = 1;
    string token0 = 2;
    string token1 = 3;
    string reserve0 = 4;
    string reserve1 = 5;
    string volume_24h = 6;
}

message GetPairsResp {
    repeated PairInfo list = 1;
    int64 total = 2;
}

service DexService {
    rpc GetPairs(GetPairsReq) returns (GetPairsResp);
}
```

运行 `goctl rpc protoc dex.proto --go_out=. --go-grpc_out=. --zrpc_out=.` 自动生成代码！

---

### 第三层：Chain Indexer（链上事件索引层）

**这是什么？**
> 它是一个"实时翻译官"，时刻监听区块链上发生的事情，翻译成数据库能存储的格式。

**工作流程：**

```
区块链产生新区块
       │
       ▼
Chain Indexer 检测到新区块
       │
       ▼
解析区块中的交易日志 (Event Logs)
       │  比如检测到 Swap 事件：
       │  Swap(sender=0x123, amount0In=100, amount1Out=50, ...)
       │
       ▼
将事件数据写入 Kafka 消息队列
       │
       ▼
对应的 RPC 服务从 Kafka 消费消息
       │
       ▼
RPC 服务将数据写入 MySQL 数据库
       │
       ▼
前端通过 API 查询数据库获取数据
```

**为什么用消息队列 (Kafka)？**
> 想象一下：链上 1 秒产生 100 个事件，但数据库 1 秒只能写 50 条。如果直接写数据库，就会丢数据。Kafka 就像一个"缓冲池"，先把事件攒着，然后慢慢消化。

**Kafka 还带来的额外好处：**

| 好处 | 说明 |
|------|------|
| 解耦 | Indexer 不需要知道谁消费事件，只管发 |
| 重放 | 消费出错了，可以从某个 offset 重新消费 |
| 多消费者 | 同一事件可以被 DEX 服务和 User 服务各消费一次 |
| 削峰填谷 | 链上突然产生大量事件时不会打爆数据库 |

---

## 三、核心数据流

### 3.1 读数据（前端查询流程）

```
前端: "我要看 ETH/USDC 交易对信息"

1. 前端 → GET /api/dex/pairs?token0=ETH&token1=USDC
2. API Gateway 收到请求，验证 JWT（公开接口可跳过）
3. API Gateway 通过 gRPC 调用 DEX RPC 服务
4. DEX RPC 先查 Redis 缓存（命中就直接返回）
5. 缓存没有 → 查 MySQL 数据库
6. 把结果写入 Redis 缓存（设置 10 秒过期）
7. 返回数据给前端

耗时预期：缓存命中 < 5ms，缓存未命中 < 50ms
```

### 3.2 写数据（链上事件同步流程）

```
链上: 有人做了一笔 Swap 交易

1. Chain Indexer 监听到 Swap 事件
2. 解析事件参数：谁交易的、交易了多少、得到多少
3. 发送到 Kafka 的 "dex-swap-events" 主题
4. DEX RPC 服务消费这个消息
5. 计算交易对的新价格、24h 交易量等
6. 写入 MySQL（交易记录表、交易对状态表）
7. 删除 Redis 中该交易对的缓存（下次查询会刷新）

延迟预期：从链上事件产生到 API 可查，< 3 秒
```

### 3.3 交易构造（用户发起交易流程）

```
前端: 用户要执行一笔 Swap

1. 前端 → POST /api/dex/swap/build
   参数: { tokenIn: "ETH", tokenOut: "USDC", amountIn: "1.5" }

2. DEX 服务收到请求
3. 查询链上最新储备量，计算最优路径
4. 构造合约调用数据 (calldata)
5. 返回给前端: { to: "0xRouter地址", data: "0x交易数据...", value: "..." }

6. 前端收到后，用 MetaMask 让用户签名并发送交易
7. 交易上链后，Chain Indexer 监听到事件，更新数据库

⚠️  重要：后端只构造交易数据，不触碰用户私钥
```

---

## 四、服务划分

| 服务名 | 端口 | 职责 | 对应合约 |
|--------|------|------|---------| 
| gateway | 8888 | API 网关 + 路由 | - |
| chain-indexer | 9000 | 链上事件监听 | 所有合约 |
| dex-rpc | 9001 | DEX 交易 | ReyPair, ReyFactory, ReyRouter |
| lending-rpc | 9002 | 借贷存取 | LendingPool, RToken, DebtToken |
| futures-rpc | 9003 | 永续期货 | PerpetualMarket, MarginManager |
| options-rpc | 9004 | 期权交易 | OptionsMarket, OptionsPricing |
| vault-rpc | 9005 | 智能金库 | BaseVault, VaultFactory |
| bonds-rpc | 9006 | 协议债券 | BondFactory, BondNFT |
| governance-rpc | 9007 | DAO 治理 | Governor, GaugeController |
| user-rpc | 9008 | 用户账户 | VeREY, ReyToken |
| bot-service | 9009 | 自动化机器人 | 调用各种合约 |

### 服务间依赖关系

```
                  ┌──────────┐
                  │ Gateway  │
                  └────┬─────┘
                       │ 调用所有 RPC 服务
         ┌─────┬───────┼───────┬─────┬─────┐
         ▼     ▼       ▼       ▼     ▼     ▼
       ┌────┐┌────┐ ┌────┐ ┌────┐┌────┐┌────┐
       │DEX ││Lend│ │User│ │Futr││Opt ││Gov │
       └────┘└────┘ └──┬─┘ └────┘└────┘└────┘
                       │ 聚合查询
         ┌─────┬───────┼───────┬─────┬─────┐
         ▼     ▼       ▼       ▼     ▼     ▼
       ┌────┐┌────┐ ┌────┐ ┌────┐┌────┐┌────┐
       │DEX ││Lend│ │Vault│ │Futr││Bond││Gov │
       └────┘└────┘ └────┘ └────┘└────┘└────┘
```

> **注意**：User 服务会调用其他业务 RPC 来聚合用户总资产，这是唯一允许的跨服务 RPC 调用。

---

## 五、go-zero 核心概念速记

> 如果你没用过 go-zero，这里用最简单的方式解释核心概念。

### 5.1 API 文件 (.api)
- **是什么**: 定义 HTTP 接口的"设计图"
- **生成什么**: Handler（处理请求）、Logic（业务逻辑）、Types（数据结构）
- **类比**: 就像餐厅的菜单，告诉你有哪些接口可以调用

### 5.2 Proto 文件 (.proto)
- **是什么**: 定义 RPC 接口的"设计图"
- **生成什么**: Server（服务端）、Client（客户端）、Model（数据模型）
- **类比**: 就像部门之间的内部通讯协议

### 5.3 Model 层
- **是什么**: 操作数据库的代码层
- **生成什么**: CRUD 方法（增删改查）
- **类比**: 就像仓库管理员，负责把数据存进去、取出来

### 5.4 ServiceContext (svc)
- **是什么**: 服务上下文，集中管理依赖注入
- **它存什么**: 数据库连接、Redis 客户端、RPC 客户端、配置
- **类比**: 就像办公室的公共工具柜，所有人都从这里拿工具

### 5.5 一个请求的完整链路

```
         用户请求
            │
     ┌──────▼──────┐
     │  Middleware  │  ← JWT 验证、日志、限流
     └──────┬──────┘
            │
     ┌──────▼──────┐
     │   Handler   │  ← 接收 HTTP 请求，解析参数
     └──────┬──────┘
            │
     ┌──────▼──────┐
     │   Logic     │  ← 业务逻辑（调 RPC、查缓存、查库）
     └──────┬──────┘
            │
     ┌──────▼──────┐
     │ RPC Client  │  ← 通过 gRPC 调用对应的业务 RPC 服务
     └──────┬──────┘
            │
     ┌──────▼──────┐
     │   Model     │  ← 操作数据库
     └──────┬──────┘
            │
     ┌──────▼──────┐
     │ MySQL/Redis │  ← 数据存储
     └─────────────┘
```

---

## 六、为什么选 go-zero？

| 对比项 | go-zero | Gin + 手动搭建 |
|--------|---------|---------------|
| 代码生成 | ✅ `goctl` 一键生成 | ❌ 全部手写 |
| RPC 支持 | ✅ 内置 gRPC | ❌ 需要手动集成 |
| 服务发现 | ✅ 内置 etcd/直连 | ❌ 需要集成 Consul 等 |
| 限流熔断 | ✅ 内置 | ❌ 需要手动实现 |
| 链路追踪 | ✅ 内置 | ❌ 需要集成 |
| 超时控制 | ✅ 内置级联超时 | ❌ 需要自己传递 Context |
| 监控指标 | ✅ 内置 Prometheus | ❌ 需要手动接入 |
| 学习曲线 | ⭐⭐ 中等 | ⭐⭐⭐ 需要丰富经验 |
| 生产就绪 | ✅ 字节/腾讯大规模使用 | 取决于个人水平 |

---

## 七、可观测性设计

> 系统上线后，出了问题你怎么排查？靠三根支柱：**日志、指标、链路追踪**。

### 7.1 日志

go-zero 内置结构化日志，推荐配置：

| 环境 | 日志级别 | 输出方式 |
|------|---------|---------|
| 开发 | debug | 控制台 |
| 测试 | info | 文件 + 控制台 |
| 生产 | info | 文件 + ELK / Loki |

关键操作要记录额外字段：

```go
logx.Infow("swap event processed",
    logx.Field("pairAddress", pair),
    logx.Field("txHash", txHash),
    logx.Field("blockNumber", blockNum),
)
```

### 7.2 指标 (Metrics)

go-zero 自动暴露 Prometheus 指标：

- HTTP 请求 QPS / 延迟 / 错误率
- gRPC 调用 QPS / 延迟 / 错误率
- 数据库连接池使用率

建议自定义的业务指标：

| 指标 | 说明 |
|------|------|
| `indexer_block_height` | 索引器当前扫描高度 |
| `indexer_chain_height` | 链最新高度 |
| `indexer_lag_blocks` | 索引延迟块数 |
| `kafka_consumer_lag` | 消费者积压消息数 |
| `cache_hit_rate` | Redis 缓存命中率 |

### 7.3 链路追踪

go-zero 内置 OpenTelemetry 支持，一个请求跨多个服务时，可以用同一个 Trace ID 串起来。

```yaml
# etc/gateway.yaml
Telemetry:
  Name: gateway
  Endpoint: http://jaeger:4318
  Sampler: 1.0   # 开发环境全量采样
```

---

## 八、健康检查与优雅关闭

### 8.1 健康检查

每个服务都应提供健康检查端点，供 Kubernetes / Docker 探活：

| 检查类型 | 建议实现 |
|---------|---------|
| Liveness | 进程存活就返回 200 |
| Readiness | 检查 MySQL + Redis 连接是否正常 |
| Startup | 检查初始化（如 Kafka 消费者）是否就绪 |

### 8.2 优雅关闭

go-zero 内置优雅关闭机制：

1. 收到 SIGTERM 信号
2. 停止接收新请求
3. 等待正在处理的请求完成（默认超时 10 秒）
4. 关闭数据库连接、Kafka 消费者
5. 进程退出

---

## 九、初期开发策略（渐进式）

> 不要一上来就搞 10 个微服务！按下面的顺序一步步来。

### Phase 1: 单体起步（1-2 周）
把所有模块放在一个服务里（monolith），先跑通整个流程。

```
backend/
└── app/
    └── monolith/          # 先用单体架构
        ├── api/           # HTTP 接口
        ├── rpc/           # 内部方法（先不拆服务）
        ├── model/         # 数据库操作
        └── chain/         # 链上交互
```

**Phase 1 完成标志**：
- [x] 能启动索引器监听链上事件
- [x] 能通过 API 查到交易对列表
- [x] 能构造一笔 Swap 交易给前端签名

### Phase 2: 拆分索引器（第 3 周）
把 Chain Indexer 独立出来，因为它需要 24/7 运行。

**Phase 2 完成标志**：
- [x] Indexer 独立进程运行
- [x] 通过 Kafka / Redis Stream 与业务服务通信
- [x] 支持重启后从断点恢复

### Phase 3: 逐步拆分 RPC（第 4-8 周）
按业务模块拆分：先拆 DEX → 借贷 → 期货 → 其他。

**Phase 3 完成标志**：
- [x] 每个模块独立 RPC 服务
- [x] Gateway 统一转发
- [x] 服务间通过 gRPC 通信

### Phase 4: 完整微服务（第 9-12 周）
所有模块独立部署，加入 Kafka、监控、告警。

**Phase 4 完成标志**：
- [x] Docker Compose 一键启动全部服务
- [x] Prometheus + Grafana 监控面板
- [x] 关键告警（索引延迟、服务宕机）已配置

---

> **下一步**: 阅读 [02_modules.md](02_modules.md) 了解每个模块的业务细节。
