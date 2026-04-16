# DEX 服务 — 去中心化交易所

## 1. 概述

DEX 服务处理所有与 **去中心化交易所** 相关的数据，包括：
- 交易对管理（哪些代币可以互换）
- 成交记录（谁在什么时候买卖了什么）
- 储备量/价格快照（交易池里有多少流动性）
- 流动性事件（谁添加/移除了流动性）

> **类比理解**：想象交易所后台的"行情系统"。它不负责执行交易（交易在链上完成），但负责记录和展示所有交易数据。

---

## 2. 数据来源 (Kafka Consumer)

### 监听的事件：

| 链上事件 | 触发时机 | 消费后写入 |
|---------|---------|-----------|
| **PairCreated** | 工厂合约创建新交易对 | `dex_pairs` 表 |
| **Swap** | 用户执行代币兑换 | `dex_trades` 表 |
| **Sync** | 每笔交易后储备量变化 | `dex_pair_snapshots` 表 |
| **Mint** | 用户添加流动性 | `dex_liquidity_events` 表 |
| **Burn** | 用户移除流动性 | `dex_liquidity_events` 表 |

### Swap 事件处理流程：

```
Kafka 消息到达
    ↓
解析 SwapPayload { sender, to, amount0In, amount1In, amount0Out, amount1Out }
    ↓
幂等检查 → SELECT COUNT(*) WHERE tx_hash = ? AND log_index = ?
    ↓ (不存在)
判断方向: amount0In > 0 → "sell" (卖 token0 换 token1)
          amount1In > 0 → "buy"  (卖 token1 换 token0)
    ↓
INSERT INTO dex_trades (...)
    ↓
清除 Redis 缓存（交易对列表、overview）
```

### Sync 事件处理：
每次 Swap 后合约都会 emit Sync(reserve0, reserve1)，我们记录这个快照用于展示实时价格。

---

## 3. 对外接口 (Server/Logic)

| RPC 方法 | 功能 | 缓存 |
|----------|------|------|
| `GetPairs` | 分页查询交易对列表 | Redis 10s |
| `GetTrades` | 查询成交记录（支持按交易对/用户筛选） | 无 |
| `GetOverview` | DEX 概览（总 TVL、24h 交易量、交易对数） | Redis 15s |
| `GetPairDetail` | 交易对详情（7d 交易量、费率等） | 无 |

### GetPairs 查询逻辑：
```
1. 尝试 Redis 缓存命中
2. 查 dex_pairs 表 → 分页列表
3. 对每个交易对，查最新的 dex_pair_snapshots → 获取 reserve0/1、price
4. 查 dex_trades 聚合 24h 交易量
5. 组装返回数据
6. 写回 Redis (TTL 10s)
```

---

## 4. 数据库表

| 表 | 字段要点 | 索引 |
|----|---------|------|
| `dex_pairs` | pair_address, token0/1_address, token0/1_symbol, fee_bps, is_active | UK(chain_id, pair_address) |
| `dex_trades` | pair_address, trader, direction, amount0/1_in/out, price, tx_hash | UK(chain_id, tx_hash, log_index) |
| `dex_pair_snapshots` | pair_address, reserve0/1, price0/1, tvl_usd, snapshot_time | IDX(pair, time) |
| `dex_liquidity_events` | pair_address, user, event_type(mint/burn), amount0/1 | UK(chain_id, tx_hash, log_index) |
| `dex_liquidity_positions` | pair_address, user, lp_balance, share_ratio | UK(chain_id, pair, user) |
| `dex_pair_stats_daily` | pair_address, date, volume_usd, trade_count, tvl_usd | UK(chain_id, pair, date) |
