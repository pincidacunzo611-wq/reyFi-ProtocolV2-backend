# 03 — 开发指南

> 本文档按"先跑通、再拆分、再完善"的思路，说明 ReyFi 后端应该怎样落地开发。

---

## 一、先定目标

第一版后端不要追求一步到位，而是先完成下面 4 件事：

1. 能监听链上事件并写入数据库
2. 能通过 API 给前端返回可用数据
3. 能构造核心交易参数给前端签名
4. 能在最少服务数量下跑通一条完整链路

---

## 二、推荐目录约定

建议在 `backend/app` 下按服务拆分：

```text
backend/
├── app/
│   ├── gateway/               # API 网关
│   ├── chain-indexer/         # 链上事件索引
│   ├── dex/                   # DEX 服务
│   ├── lending/               # 借贷服务
│   ├── futures/               # 期货服务
│   ├── options/               # 期权服务
│   ├── vault/                 # 金库服务
│   ├── bonds/                 # 债券服务
│   ├── governance/            # 治理服务
│   ├── user/                  # 用户服务
│   └── bot/                   # 机器人服务
├── pkg/                       # 跨服务公共包
│   ├── chains/                # 链交互工具（ABI 绑定、客户端封装）
│   ├── response/              # 统一响应结构
│   ├── errorx/                # 统一错误码
│   ├── mathx/                 # 大数计算工具
│   └── middleware/            # 通用中间件
├── deploy/                    # 部署配置
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml # 开发环境（只启 MySQL + Redis）
│   └── nginx.conf
├── scripts/                   # 脚本工具
│   ├── migration/             # 数据库迁移
│   └── gen/                   # 代码生成脚本
├── Makefile                   # 常用命令入口
├── go.mod
├── go.sum
└── README.md
```

每个服务内部建议统一：

```text
service-name/
├── api/                  # 仅 gateway 使用
│   └── service-name.api  # API 路由定义文件
├── rpc/                  # proto 和 rpc 逻辑
│   └── service-name.proto
├── internal/
│   ├── config/           # 配置结构定义
│   │   └── config.go
│   ├── handler/          # API handler（由 goctl 生成）
│   ├── logic/            # 业务逻辑（核心代码写这里）
│   ├── svc/              # 服务上下文（依赖注入）
│   │   └── servicecontext.go
│   ├── model/            # 数据库 Model
│   ├── consumer/         # Kafka 消费者
│   ├── cron/             # 定时任务
│   └── chain/            # 合约 ABI / 事件解析
├── etc/                  # 配置文件
│   ├── service-name.yaml     # 主配置
│   └── service-name-dev.yaml # 开发环境配置
└── README.md             # 服务说明
```

---

## 三、环境准备

### 3.1 必需工具

| 工具 | 版本 | 安装命令 |
|------|------|---------|
| Go | >= 1.21 | [官网下载](https://go.dev/dl/) |
| goctl | latest | `go install github.com/zeromicro/go-zero/tools/goctl@latest` |
| protoc | >= 3.19 | [protobuf releases](https://github.com/protocolbuffers/protobuf/releases) |
| protoc-gen-go | latest | `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` |
| protoc-gen-go-grpc | latest | `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest` |
| MySQL | >= 8.0 | Docker 或本地安装 |
| Redis | >= 7.0 | Docker 或本地安装 |

### 3.2 开发环境 Docker Compose

创建 `deploy/docker-compose.dev.yml`，一键启动开发依赖：

```yaml
# deploy/docker-compose.dev.yml
version: "3.8"

services:
  mysql:
    image: mysql:8.0
    container_name: reyfi-mysql
    environment:
      MYSQL_ROOT_PASSWORD: reyfi_dev_123
      MYSQL_DATABASE: reyfi
      MYSQL_CHARSET: utf8mb4
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ../scripts/migration:/docker-entrypoint-initdb.d  # 自动执行建表 SQL
    command: --default-authentication-plugin=mysql_native_password

  redis:
    image: redis:7-alpine
    container_name: reyfi-redis
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  # 初期可选，Phase 1 先用 Redis Stream 代替
  # kafka:
  #   image: bitnami/kafka:3.6
  #   container_name: reyfi-kafka
  #   environment:
  #     KAFKA_CFG_NODE_ID: 1
  #     KAFKA_CFG_PROCESS_ROLES: broker,controller
  #     KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
  #     KAFKA_CFG_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
  #     KAFKA_CFG_CONTROLLER_LISTENER_NAMES: CONTROLLER
  #   ports:
  #     - "9092:9092"

volumes:
  mysql_data:
  redis_data:
```

启动命令：

```bash
cd backend/deploy
docker compose -f docker-compose.dev.yml up -d
```

### 3.3 配置文件示例

```yaml
# app/dex/etc/dex-dev.yaml
Name: dex.rpc
ListenOn: 0.0.0.0:9001

# 数据库
DataSource: root:reyfi_dev_123@tcp(127.0.0.1:3306)/reyfi?charset=utf8mb4&parseTime=true&loc=UTC

# 缓存
Cache:
  - Host: 127.0.0.1:6379

# 链上配置
Chain:
  RpcUrl: https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY
  ChainId: 11155111
  Contracts:
    Factory: "0xFactoryAddress"
    Router: "0xRouterAddress"

# 日志
Log:
  ServiceName: dex.rpc
  Mode: console
  Level: debug

# 链路追踪（开发环境可选）
# Telemetry:
#   Name: dex.rpc
#   Endpoint: http://localhost:4318
#   Sampler: 1.0
```

---

## 四、开发顺序

### 4.1 第一步：先做基础设施

先准备：

- MySQL
- Redis
- 一个 EVM RPC 节点地址（Alchemy / Infura 免费额度即可）
- go-zero / goctl

本地开发可以先只启：

- MySQL
- Redis
- 一个简化版 Indexer

Kafka 可以先临时替换成 Redis Stream 或进程内 channel，等链路通了再切回 Kafka。

### 4.2 第二步：先做单体版链路

虽然目标是微服务，但第一阶段建议先只做 3 个服务：

1. `gateway`
2. `chain-indexer`
3. `dex`

原因很简单：

- DEX 事件最直接，最适合验证索引链路
- 前端通常最先需要行情、池子和交易对
- 可以快速验证"链上事件 -> 数据库 -> API"闭环

### 4.3 第三步：把用户总览做出来

当 DEX 打通后，再做 `user` 服务，把：

- 钱包登录
- 交易历史
- 跨模块资产总览

先建立框架，即使一开始只接入 DEX 也没关系。

### 4.4 第四步：逐步接入复杂模块

推荐顺序：

1. Lending
2. Futures / Leverage
3. Options
4. Vault
5. Bonds
6. Governance
7. Bot

---

## 五、如何新建一个 go-zero 服务

### 5.1 创建 API 服务

适用于 Gateway：

```bash
# 方式一：快速创建
goctl api new gateway

# 方式二：先写 .api 文件再生成（推荐）
goctl api go -api gateway.api -dir .
```

### 5.2 创建 RPC 服务

以 DEX 为例：

```bash
# 方式一：快速创建
goctl rpc new dex

# 方式二：基于 proto（推荐）
goctl rpc protoc dex.proto --go_out=. --go-grpc_out=. --zrpc_out=.
```

### 5.3 创建 model

```bash
# 从已有数据库表生成
goctl model mysql datasource \
  -url="root:reyfi_dev_123@tcp(127.0.0.1:3306)/reyfi" \
  -table="dex_pairs" \
  -dir="./internal/model" \
  -cache  # 加上 -cache 自动带 Redis 缓存能力
```

### 5.4 Makefile 建议

在项目根目录创建 `Makefile`，减少记忆成本：

```makefile
# Makefile

# ==================== 开发环境 ====================
.PHONY: dev-up dev-down

dev-up:                              ## 启动开发依赖 (MySQL + Redis)
	docker compose -f deploy/docker-compose.dev.yml up -d

dev-down:                            ## 停止开发依赖
	docker compose -f deploy/docker-compose.dev.yml down

# ==================== 代码生成 ====================
.PHONY: gen-api gen-rpc gen-model

gen-api:                             ## 生成 gateway API 代码
	cd app/gateway && goctl api go -api api/gateway.api -dir .

gen-rpc-dex:                         ## 生成 DEX RPC 代码
	cd app/dex && goctl rpc protoc rpc/dex.proto \
		--go_out=. --go-grpc_out=. --zrpc_out=.

gen-model-dex:                       ## 生成 DEX model 代码
	goctl model mysql datasource \
		-url="root:reyfi_dev_123@tcp(127.0.0.1:3306)/reyfi" \
		-table="dex_pairs,dex_trades,dex_pair_snapshots" \
		-dir="app/dex/internal/model" -cache

# ==================== 运行服务 ====================
.PHONY: run-gateway run-dex run-indexer

run-gateway:                         ## 启动 Gateway
	cd app/gateway && go run gateway.go -f etc/gateway-dev.yaml

run-dex:                             ## 启动 DEX RPC
	cd app/dex && go run dex.go -f etc/dex-dev.yaml

run-indexer:                         ## 启动 Chain Indexer
	cd app/chain-indexer && go run indexer.go -f etc/indexer-dev.yaml

# ==================== 数据库 ====================
.PHONY: migrate migrate-status

migrate:                             ## 执行数据库迁移
	cd scripts/migration && go run migrate.go up

migrate-status:                      ## 查看迁移状态
	cd scripts/migration && go run migrate.go status

# ==================== 测试 ====================
.PHONY: test test-cover lint

test:                                ## 运行单元测试
	go test ./... -v -count=1

test-cover:                          ## 运行测试并生成覆盖率报告
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:                                ## 代码检查
	golangci-lint run ./...

# ==================== 帮助 ====================
.PHONY: help
help:                                ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
```

使用方式：

```bash
make dev-up        # 启动 MySQL + Redis
make gen-rpc-dex   # 生成 DEX RPC 代码
make run-gateway   # 启动网关
make help          # 查看所有可用命令
```

---

## 六、建议的接口分层

### 6.1 Handler 层

只负责：

- 收请求参数
- 做基础校验
- 调 Logic
- 返回统一响应

不要把业务逻辑写在 Handler 里。

```go
// 好的 Handler 示例
func GetPairsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.GetPairsReq
        if err := httpx.Parse(r, &req); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        l := logic.NewGetPairsLogic(r.Context(), svcCtx)
        resp, err := l.GetPairs(&req)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
        } else {
            httpx.OkJsonCtx(r.Context(), w, resp)
        }
    }
}
```

### 6.2 Logic 层

负责：

- 组合业务规则
- 调 RPC / Model / Redis
- 处理鉴权、分页、过滤条件
- 组织返回结构

```go
// 好的 Logic 示例
func (l *GetPairsLogic) GetPairs(req *types.GetPairsReq) (*types.GetPairsResp, error) {
    // 1. 先查缓存
    cacheKey := fmt.Sprintf("dex:pairs:page:%d", req.Page)
    if cached, err := l.svcCtx.Redis.Get(cacheKey); err == nil {
        // 反序列化并返回
    }

    // 2. 缓存未命中，查 RPC
    rpcResp, err := l.svcCtx.DexRpc.GetPairs(l.ctx, &dex.GetPairsReq{
        Page:     req.Page,
        PageSize: req.PageSize,
    })
    if err != nil {
        return nil, err
    }

    // 3. 写缓存
    l.svcCtx.Redis.Setex(cacheKey, serialize(rpcResp), 10)

    // 4. 返回
    return convertToResp(rpcResp), nil
}
```

### 6.3 Model 层

负责：

- 对数据库做 CRUD
- 封装通用查询
- 不关心 HTTP 和业务语义

### 6.4 Consumer 层

负责：

- 订阅 Kafka 主题
- 做事件幂等
- 调用领域服务写库

```go
// 消费者示例
func (c *SwapConsumer) Consume(ctx context.Context, key, value string) error {
    var event ChainEvent
    if err := json.Unmarshal([]byte(value), &event); err != nil {
        return err
    }

    // 幂等检查：同一事件不重复处理
    exists, err := c.model.ExistsByTxHashAndLogIndex(ctx, event.TxHash, event.LogIndex)
    if err != nil {
        return err
    }
    if exists {
        logx.Infof("event already processed: %s:%d", event.TxHash, event.LogIndex)
        return nil
    }

    // 处理事件
    return c.processSwapEvent(ctx, &event)
}
```

---

## 七、事件消费的工程约束

### 7.1 必须做幂等

判断维度建议至少包含：

- `chain_id`
- `tx_hash`
- `log_index`

这三个值拼起来可以唯一识别一条事件。

建议在数据库建联合唯一索引：

```sql
ALTER TABLE dex_trades
ADD UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index);
```

### 7.2 必须支持补块

因为服务可能：

- 重启
- 宕机
- 节点短暂断连

所以要记录每个合约或每个模块的同步高度，并支持指定区块范围重扫。

```go
// 补块示例逻辑
func (i *Indexer) BackfillBlocks(ctx context.Context, from, to int64) error {
    for blockNum := from; blockNum <= to; blockNum++ {
        block, err := i.ethClient.BlockByNumber(ctx, big.NewInt(blockNum))
        if err != nil {
            return fmt.Errorf("fetch block %d: %w", blockNum, err)
        }
        if err := i.processBlock(ctx, block); err != nil {
            return fmt.Errorf("process block %d: %w", blockNum, err)
        }
    }
    return nil
}
```

### 7.3 必须支持重组回滚

做法建议：

- 事件先落原始表（`chain_raw_events`）
- 达到确认块数后再更新聚合表
- 如果发生 reorg，按区块回滚影响范围

```
正常流程：
  新事件 → chain_raw_events (status=pending)
       → 等待 N 个确认块
       → 更新 status=confirmed
       → 写入业务聚合表

重组流程：
  检测到 reorg → 找到分叉点
       → 将分叉点之后的事件标记 status=reverted
       → 回滚聚合表中对应的记录
       → 从分叉点重新扫描
```

---

## 八、数据库开发建议

### 8.1 先有原始表，再有聚合表

不要一上来就只存最终结果，建议分两类：

- 原始事件表（`chain_raw_events`、`dex_trades` 等）
- 业务聚合表（`dex_pair_stats_daily`、`user_asset_snapshots` 等）

这样后面修计算逻辑时，可以通过重放事件修复数据。

### 8.2 金额字段统一用字符串或高精度十进制

链上金额不能用浮点。

推荐：

- 数据库存 `DECIMAL(65, 18)` 或字符串
- Go 内部用 `big.Int` / `big.Rat`

```go
// ❌ 错误：浮点精度丢失
amount := 1.5e18

// ✅ 正确：用 big.Int
amount, _ := new(big.Int).SetString("1500000000000000000", 10)

// ✅ 数据库读出后转换
func decimalToBigInt(d string) *big.Int {
    bi, _ := new(big.Int).SetString(d, 10)
    return bi
}
```

### 8.3 时间统一存 UTC

前端再根据用户时区格式化。

```go
// Go 中统一使用 UTC
time.Now().UTC()

// MySQL 连接串加 loc=UTC
"root:pass@tcp(127.0.0.1:3306)/reyfi?charset=utf8mb4&parseTime=true&loc=UTC"
```

### 8.4 数据库迁移工具

推荐使用 [golang-migrate](https://github.com/golang-migrate/migrate) 管理表结构变更：

```bash
# 安装
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 创建迁移文件
migrate create -ext sql -dir scripts/migration -seq create_dex_pairs

# 执行迁移
migrate -path scripts/migration -database "mysql://root:pass@tcp(127.0.0.1:3306)/reyfi" up

# 回滚
migrate -path scripts/migration -database "mysql://root:pass@tcp(127.0.0.1:3306)/reyfi" down 1
```

迁移文件示例：

```sql
-- scripts/migration/000001_create_dex_pairs.up.sql
CREATE TABLE IF NOT EXISTS dex_pairs (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id     BIGINT          NOT NULL,
    pair_address VARCHAR(42)     NOT NULL,
    token0       VARCHAR(42)     NOT NULL,
    token1       VARCHAR(42)     NOT NULL,
    fee_bps      INT             NOT NULL DEFAULT 30,
    created_block BIGINT         NOT NULL DEFAULT 0,
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_pair (chain_id, pair_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

```sql
-- scripts/migration/000001_create_dex_pairs.down.sql
DROP TABLE IF EXISTS dex_pairs;
```

---

## 九、缓存建议

### 9.1 适合进 Redis 的内容

| 数据类型 | 缓存时长 | 缓存 Key 模式 |
|---------|---------|--------------|
| 热门交易对行情 | 5-15 秒 | `dex:pair:{address}:info` |
| 首页 Dashboard | 10-30 秒 | `dashboard:overview` |
| 用户总资产摘要 | 10-30 秒 | `user:{address}:portfolio` |
| 治理提案详情 | 30-60 秒 | `gov:proposal:{id}` |
| 金库列表 | 30-60 秒 | `vault:list` |
| 借贷市场列表 | 15-30 秒 | `lending:markets` |

### 9.2 不适合只放缓存不落库的数据

- 用户交易历史
- 清算记录
- 结算记录
- 任何需要审计和回溯的数据

### 9.3 缓存失效策略

```
事件驱动失效（推荐）：
  Swap 事件 → 消费者处理完毕 → 删除关联缓存 Key
  下次查询 → 缓存为空 → 查库 → 写缓存

主动过期兜底：
  所有缓存都设置 TTL，即使删除失败也会自然过期
```

---

## 十、统一响应和错误码

### 10.1 统一响应结构

建议 Gateway 层统一响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "traceId": "abc123"
}
```

### 10.2 统一响应工具

```go
// pkg/response/response.go
package response

import (
    "net/http"
    "github.com/zeromicro/go-zero/rest/httpx"
)

type Body struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    TraceId string      `json:"traceId,omitempty"`
}

func Success(w http.ResponseWriter, data interface{}) {
    httpx.OkJson(w, &Body{
        Code:    0,
        Message: "ok",
        Data:    data,
    })
}

func Error(w http.ResponseWriter, code int, msg string) {
    httpx.OkJson(w, &Body{
        Code:    code,
        Message: msg,
    })
}
```

### 10.3 错误码分层

| 范围 | 用途 | 示例 |
|------|------|------|
| `0` | 成功 | |
| `1000-1999` | 通用参数错误 | 1001 参数缺失、1002 分页非法 |
| `2000-2999` | 鉴权错误 | 2001 未登录、2002 签名无效、2003 Token 过期 |
| `3000-3999` | DEX 业务错误 | 3001 交易对不存在、3002 滑点超限 |
| `4000-4999` | Lending 业务错误 | 4001 市场不存在、4002 健康度不足 |
| `5000-5999` | Futures / Options 错误 | 5001 仓位不存在、5002 保证金不足 |
| `6000-6999` | Vault / Bonds 错误 | 6001 金库不存在 |
| `7000-7999` | Governance 错误 | 7001 提案不存在 |
| `9000-9999` | 系统错误 | 9001 系统繁忙、9002 服务不可用 |

---

## 十一、建议先写哪些测试

### 单元测试（优先）

| 测试对象 | 说明 | 输入/输出 |
|---------|------|----------|
| 事件解析 | 将原始 Log 解析为结构体 | 原始字节 → 事件结构 |
| 利率计算 | 验证利率模型正确性 | 利用率 → APR |
| 健康度计算 | 验证清算逻辑 | 抵押物/借款 → 健康因子 |
| 资金费率计算 | 验证费率公式 | 标记价/指数价 → 费率 |
| 期权报价计算 | 验证定价模型 | 参数 → 权利金 |
| 大数运算 | 验证精度 | 链上原始值 → 可读值 |

### 集成测试

- 索引器从区块抓事件并写库
- API 查询聚合结果
- 缓存命中与失效

### 回归测试

- 重扫区块后数据是否一致
- 同一事件重复消费是否会脏写

---

## 十二、第一阶段里程碑

### Milestone 1 — DEX 数据可查（预计 1 周）

- [x] 开发环境搭建（Docker Compose + 配置文件）
- [x] DEX 交易对可查
- [x] DEX 成交记录可查

### Milestone 2 — 用户登录可用（预计 1 周）

- [x] 用户钱包签名登录
- [x] JWT 鉴权中间件
- [x] 用户 DEX 资产总览可查

### Milestone 3 — 借贷模块上线（预计 1-2 周）

- [x] Lending 存借数据可查
- [x] 健康因子可查
- [x] 存取交易构造可用

### Milestone 4 — 期货模块上线（预计 1-2 周）

- [x] Futures 仓位可查
- [x] 清算候选列表可产出
- [x] 开平仓交易构造可用

---

## 十三、常见问题 FAQ

### Q: 本地开发需要一个真的以太坊节点吗？

**A:** 不需要。初期开发可以用 Hardhat / Anvil 本地节点，配合自己部署的测试合约。

```bash
# 启动本地节点
npx hardhat node
# 或
anvil --fork-url https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
```

### Q: 没有 Kafka，怎么开发消费者？

**A:** 先用 Redis Stream 或 Go channel 代替。定义好消息接口，后面切换实现即可。

```go
// 定义消息接口
type EventPublisher interface {
    Publish(ctx context.Context, topic string, event *ChainEvent) error
}

type EventConsumer interface {
    Subscribe(ctx context.Context, topic string, handler func(event *ChainEvent) error) error
}

// 初期用 channel 实现
type ChannelPublisher struct { ... }
// 后期切换为 Kafka 实现
type KafkaPublisher struct { ... }
```

### Q: 多个服务共用一个数据库还是各自一个？

**A:** Phase 1 共用一个数据库，按表名前缀隔离即可（`dex_*`、`lending_*`）。Phase 3+ 再考虑按服务拆库。

---

## 十四、最重要的工程原则

1. **先保证数据正确，再优化性能**
2. **先保留原始事件，再做聚合**
3. **先让最小闭环可用，再拆服务**
4. **所有链上事件消费都要幂等**
5. **所有关键计算都要可重放、可修复**
6. **接口不变、实现可换**（面向接口编程，方便替换 Kafka/Redis Stream）
