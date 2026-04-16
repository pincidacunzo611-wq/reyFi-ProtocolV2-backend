# Chain Indexer — 链上事件索引器

## 1. 概述

Chain Indexer 是整个后端的 **数据源头**，它的唯一职责就是：

> **不断监听区块链上的新区块，提取出我们关心的合约事件，然后发布到 Kafka 让下游服务消费。**

它类似于一个"区块链爬虫"，24/7 不间断运行。

---

## 2. 核心流程

```
  ┌──────────┐
  │ 启动服务  │
  └────┬─────┘
       ▼
  ┌──────────────────┐
  │ 读取上次同步位置    │  ← chain_sync_cursors 表
  │ (last_scanned_block)│
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 查询链上最新区块号  │  ← eth_blockNumber
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 有新区块吗？       │
  │  否 → 休眠 N 毫秒  │
  │  是 ↓              │
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 链重组检测         │  ← 对比本地/链上 block hash
  │  发现重组 → 回滚   │
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 批量扫描区块       │  ← eth_getLogs (FilterQuery)
  │ from → to (batch) │
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 保存区块头信息     │  → chain_blocks 表
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 解析每条 Log       │  ← EventParser
  │ 识别事件类型        │
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 保存原始事件       │  → chain_raw_events 表 (数据真相)
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 发布到 Kafka       │  → 按模块分发到不同 topic
  └────┬─────────────┘
       ▼
  ┌──────────────────┐
  │ 更新同步游标       │  → chain_sync_cursors 表
  │ 继续下一批         │
  └──────────────────┘
```

---

## 3. 文件结构

```
app/chain-indexer/
├── indexer.go                          # main 入口
├── etc/indexer-dev.yaml                # 配置
└── internal/
    ├── config/config.go                # 配置结构体
    ├── svc/servicecontext.go           # 依赖注入 (DB, ChainClient, Publisher)
    ├── indexer/
    │   ├── blockscanner.go             # ⭐ 核心：区块扫描器
    │   ├── eventparser.go              # ⭐ 核心：事件解析器
    │   └── reorgdetector.go            # 链重组检测
    └── publisher/publisher.go          # Kafka 事件发布
```

---

## 4. 核心组件详解

### 4.1 BlockScanner (blockscanner.go)

**作用**：驱动整个扫描循环。

**关键参数** (indexer-dev.yaml):
```yaml
Scanner:
  StartBlock: 0        # 首次启动从哪个区块开始
  BatchSize: 100       # 每次扫描多少个区块
  PollInterval: 3000   # 无新区块时，休眠多久再查 (毫秒)
  ConfirmBlocks: 12    # 等待多少个确认块后标记事件为 confirmed
```

**核心方法**：
| 方法 | 功能 |
|------|------|
| `Start()` | 主循环：查新块 → 链重组检测 → 扫描 → 发布 → 更新游标 |
| `scanBlockRange(from, to)` | 调用 `eth_getLogs` 获取指定区块范围的事件日志 |
| `saveBlockHeaders()` | 保存区块头的 hash/parentHash 用于重组检测 |
| `saveRawEvent()` | 把事件写入 `chain_raw_events` 表（数据备份） |
| `confirmEvents()` | 经过足够确认后，将 pending 事件标记为 confirmed |
| `BackfillBlocks()` | 手动指定区块范围重新扫描（补块） |

**幂等保证**：
- `chain_raw_events` 表有 `UNIQUE KEY (chain_id, tx_hash, log_index)`
- 即使同一事件被扫描两次，`ON DUPLICATE KEY UPDATE id=id` 会忽略

### 4.2 EventParser (eventparser.go)

**作用**：把原始的 `types.Log` 解析为我们自定义的 `ChainEvent` 结构。

**事件识别原理**：
```
Solidity 事件: Swap(address sender, uint256 amount0In, uint256 amount1In, ...)

↓ 编译时自动生成

Topic0 = Keccak256("Swap(address,uint256,uint256,uint256,uint256,address)")
       = 0xd78ad95f...

↓ EventParser 维护一张映射表

Topic0 → { Module: "dex", EventName: "Swap" }
```

**目前注册的事件** (30+个):
- DEX: PairCreated, Mint, Burn, Swap, Sync
- Lending: Deposit, Withdraw, Borrow, Repay, LiquidationCall
- Futures: PositionOpened, PositionClosed, FundingSettled, PositionLiquidated
- Options: OptionPurchased, OptionExercised, OptionExpired, SettlementExecuted
- Vault: VaultCreated, Deposit, Withdraw, Harvest, StrategyUpdated
- Bonds: BondCreated, BondPurchased, BondRedeemed
- Governance: ProposalCreated, VoteCast, ProposalExecuted, LockCreated

**Payload 解析**：
对于 DEX 核心事件 (Swap, Sync, Mint, Burn, PairCreated)，有专门的类型化解析函数。
对于其他事件，使用通用解析（将 topics 和 data 原样保存为 JSON）。

### 4.3 ReorgDetector (reorgdetector.go)

**什么是链重组？**
区块链有时会出现分叉。比如你认为第 100 号区块的 hash 是 A，但链最终认定 hash 是 B。这意味着第 100 号区块被"替换"了，里面的交易可能不一样。

**检测方法**：
1. 拿本地存的 block 100 的 hash
2. 从链上查 block 100 的 hash
3. 如果不一致 → 发生重组
4. 向前逐个对比，找到"分叉起始点"

**回滚操作**：
1. 将受影响区块的事件标记为 `status = 'reverted'`
2. 删除分叉后的区块记录
3. 更新同步游标到分叉点前一个区块
4. 下一次循环会重新扫描这些区块

### 4.4 Publisher (publisher.go)

**作用**：将 ChainEvent 发布到正确的 Kafka topic。

**Topic 路由**：
```
event.Module == "dex"        → reyfi.dex.events
event.Module == "lending"    → reyfi.lending.events
event.Module == "futures"    → reyfi.futures.events
... 以此类推
```

**可靠性**：
- 同步写入模式 (`Async: false`)，确保消息不丢
- 支持按 topic 自动创建 writer，懒加载

---

## 5. 涉及的数据库表

| 表名 | 作用 |
|------|------|
| `chain_blocks` | 存储扫描过的区块头（hash, parentHash, 时间） |
| `chain_sync_cursors` | 记录每个模块扫到第几个区块了 |
| `chain_raw_events` | 所有链上事件的原始备份（数据真相来源） |

---

## 6. 配置说明

```yaml
# etc/indexer-dev.yaml
Chain:
  RpcUrl: http://127.0.0.1:8545     # 以太坊节点 RPC 地址
  ChainId: 31337                     # 链 ID (31337 = Anvil 本地)
  Contracts:                         # 需要监听的合约地址
    Factory: "0x5FbDB2315678..."
    Router: "0xe7f1725E7734..."
    LendingPool: "0x9fE46736..."

Scanner:
  StartBlock: 0      # 从第几个区块开始扫
  BatchSize: 100     # 每批扫多少区块
  PollInterval: 3000 # 查询间隔 (ms)
  ConfirmBlocks: 12  # 确认块数

KafkaBrokers:
  - 127.0.0.1:9092   # Kafka 地址
```
