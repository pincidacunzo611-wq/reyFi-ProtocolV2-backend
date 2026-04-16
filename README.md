# ReyFi Protocol V2 - Go-Zero 后端

> **一句话总结**: 用 Go-Zero 微服务架构，为 ReyFi DeFi 协议构建一个高性能后端，负责 **监听链上事件 -> 存储到数据库 -> 通过 API 提供给前端**。

---

## 后端到底是做什么的？

你的智能合约已经部署在区块链上了，但是：

| 问题 | 为什么需要后端 |
|------|---------------|
| 链上查询太慢 | 直接从链上读取大量用户头寸和历史数据，前端会很卡 |
| 没有历史记录 | 链上更适合存最新状态，不适合直接做丰富历史查询 |
| 无法做复杂聚合 | 比如“某用户在所有模块的总资产/总收益/风险等级” |
| 前端需要聚合数据 | Dashboard 通常需要一次性拿到多个模块的汇总数据 |
| 定时任务 | 例如清算检查、费率结算、补块、重放等任务 |

**所以后端的角色是：**

```text
区块链 --(事件)--> 后端 --(API)--> 前端
                 ├─ 监听合约事件，解析并写入数据库
                 ├─ 提供 RESTful API 给前端调用
                 ├─ 运行定时任务（清算检查、费率结算等）
                 └─ 构造交易数据，让用户签名后上链
```

---

## 项目结构总览

```text
backend/
├── README.md                          # 你正在看的这个文件
├── doc/
│   ├── 01_architecture.md             # 整体架构设计
│   ├── 02_modules.md                  # 九大模块业务详解
│   ├── 03_development_guide.md        # 开发指南
│   ├── 04_database.md                 # 数据库设计
│   └── 05_api_design.md               # API 设计
├── app/                               # 微服务代码（后续开发）
│   ├── gateway/                       # API 网关
│   ├── chain-indexer/                 # 链上事件索引服务
│   ├── dex/                           # DEX 服务
│   ├── lending/                       # 借贷服务
│   ├── futures/                       # 期货服务
│   ├── options/                       # 期权服务
│   ├── vault/                         # 金库服务
│   ├── bonds/                         # 债券服务
│   ├── governance/                    # 治理服务
│   ├── user/                          # 用户/账户服务
│   └── bot/                           # 自动化机器人服务
├── pkg/                               # 跨服务公共包
│   ├── chains/                        # 链交互工具
│   ├── response/                      # 统一响应结构
│   ├── errorx/                        # 统一错误码
│   └── mathx/                         # 大数计算工具
├── scripts/                           # 脚本工具
│   └── migration/                     # 数据库迁移
├── deploy/                            # 部署配置
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml
│   └── nginx.conf
├── Makefile                           # 常用命令入口
├── go.mod
└── go.sum
```

---

## 快速开始阅读

**推荐阅读顺序：**

| 步骤 | 文档 | 你将了解 |
|------|------|---------|
| 1 | [01_architecture.md](doc/01_architecture.md) | 系统整体架构、三层设计、数据流 |
| 2 | [02_modules.md](doc/02_modules.md) | 各模块业务职责、依赖关系、关键计算 |
| 3 | [04_database.md](doc/04_database.md) | 表结构设计、索引规范、数据分层 |
| 4 | [05_api_design.md](doc/05_api_design.md) | API 规范、请求/响应示例、鉴权 |
| 5 | [03_development_guide.md](doc/03_development_guide.md) | 环境搭建、开发步骤、目录约定 |

---

## 技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| 微服务框架 | **go-zero** | API 网关 + RPC + 代码生成 |
| API 协议 | REST (HTTP) | 方便前端调用 |
| 内部通信 | gRPC (Protobuf) | 服务间高性能通信 |
| 数据库 | MySQL 8.0+ | 业务数据存储 |
| 缓存 | Redis 7.0+ | 热点缓存、限流、会话等 |
| 消息队列 | Kafka | 异步处理链上事件 |
| 链交互 | go-ethereum | 监听事件、合约调用 |
| 定时任务 | go-zero 内置 / cron | 清算检查、费率结算等 |
| 可观测性 | Prometheus + Jaeger | 指标监控与链路追踪 |
| 数据库迁移 | golang-migrate | 版本化管理表结构变更 |

---

## 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis 7.0+
- Kafka 3.0+（初期可先用 Redis Stream 代替）
- goctl
- protoc

```bash
# 安装 goctl
go install github.com/zeromicro/go-zero/tools/goctl@latest

# 验证安装
goctl --version

# 安装 protobuf 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

---

## 快速启动开发环境

```bash
# 1. 启动 MySQL + Redis
cd deploy
docker compose -f docker-compose.dev.yml up -d

# 2. 执行数据库迁移
make migrate

# 3. 启动服务
make run-gateway
make run-dex
make run-indexer
```

详细开发环境说明请查看 [03_development_guide.md](doc/03_development_guide.md)。
