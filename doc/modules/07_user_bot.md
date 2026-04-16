# User 服务 — 用户/账户

## 1. 概述

User 服务处理用户身份认证和跨模块数据聚合。

> **核心特点**：ReyFi 使用 **钱包地址** 作为用户身份，没有传统的用户名/密码。用户通过在钱包中签名一段消息来证明自己拥有某个地址。

---

## 2. 钱包登录流程详解

```
步骤 1: 获取 Nonce (防重放攻击)
┌──────┐                              ┌──────┐
│ 前端 │ → POST /api/user/auth/nonce  │ 后端 │
│      │   { address: "0xABC..." }     │      │
│      │                               │      │
│      │ ← { nonce: "Sign this...",   │      │
│      │     expiresIn: 300 }          │      │
└──────┘                              └──────┘

步骤 2: 用户在钱包中签名
┌──────┐     ┌──────────┐
│ 前端 │ ──→ │ MetaMask │  用户点击"签名"
│      │ ←── │          │  返回 signature
└──────┘     └──────────┘

步骤 3: 登录验证
┌──────┐                              ┌──────┐
│ 前端 │ → POST /api/user/auth/login  │ 后端 │
│      │   { address, message,         │      │
│      │     signature }               │      │
│      │                               │      │
│      │   后端做了什么？               │      │
│      │   1. 从 signature 恢复公钥     │      │
│      │   2. 公钥 → 地址              │      │
│      │   3. 对比地址是否一致          │      │
│      │   4. 检查 nonce 是否有效       │      │
│      │   5. 生成 JWT token            │      │
│      │                               │      │
│      │ ← { accessToken, refreshToken,│      │
│      │     expiresIn: 7200 }         │      │
└──────┘                              └──────┘
```

### 签名验证的数学原理：
```
消息 → 以太坊前缀 → Keccak256 哈希 → ecrecover(hash, sig) → 公钥 → 地址
                                                                    ↓
                                                            与请求地址对比
```

---

## 3. 投资组合聚合

`GetPortfolio` 方法聚合用户在所有模块中的资产：

```json
{
  "walletAddress": "0xABC...",
  "totalAssetUsd": "125000.50",
  "totalDebtUsd": "30000.00",
  "netValueUsd": "95000.50",
  "pnl24h": "+1250.00",
  "pnl24hPercent": "+1.33",
  "allocation": [
    { "module": "dex",        "label": "流动性池", "valueUsd": "50000", "percent": "52.6" },
    { "module": "lending",    "label": "借贷",    "valueUsd": "25000", "percent": "26.3" },
    { "module": "futures",    "label": "永续",    "valueUsd": "10000", "percent": "10.5" },
    { "module": "vault",      "label": "金库",    "valueUsd": "8000",  "percent": "8.4" },
    { "module": "bonds",      "label": "债券",    "valueUsd": "2000",  "percent": "2.1" }
  ],
  "riskSummary": {
    "lendingHealthFactor": "1.85",
    "futuresMarginRatio": "0.45",
    "riskLevel": "medium"
  }
}
```

---

## 4. 活动流

`GetActivity` 返回用户在所有模块中的操作记录（类似银行流水）：

```
2024-04-16 09:30  [dex]        swap     卖出 1 ETH 换入 3000 USDC
2024-04-16 09:25  [lending]    deposit  存入 5000 USDC 到借贷池
2024-04-16 09:20  [futures]    open     开多 ETH-USDC 10x 100 USDC
2024-04-16 09:15  [governance] vote     赞成投票 提案 #42
2024-04-16 09:10  [bonds]      purchase 购买债券 1000 USDC
```

所有 Consumer 在处理事件时，都会同时写入 `user_activity_stream` 表。

---

## 5. 数据库表

| 表 | 说明 |
|----|------|
| `users` | 用户基本信息（地址、状态、最后登录）|
| `user_nonces` | 登录 nonce 记录（防重放） |
| `user_sessions` | 会话记录（refresh_token、过期时间） |
| `user_asset_snapshots` | 每日资产快照（用于计算 PnL 趋势） |
| `user_activity_stream` | 全模块活动流（消费者写入，Gateway 读取） |

---

# Bot 服务 — 自动化机器人

## 1. 概述

Bot 服务是一个 **后台定时任务服务**，运行各种自动化维护任务。

> **它不对外提供 API**，纯后台运行，定时执行各种检查和统计任务。

---

## 2. 任务列表

| 任务 | 间隔 | 功能 |
|------|------|------|
| **清算监控** | 10s | 检查借贷/永续仓位的健康因子 |
| **价格更新** | 30s | 从 DEX 储备量同步价格 |
| **资金费率结算** | 1h | 检查永续合约资金费率 |
| **期权到期检查** | 60s | 标记已到期的期权为 expired |
| **日统计聚合** | 每天 01:00 UTC | 聚合前一天的交易量/TVL/用户资产快照 |

---

## 3. 清算监控详解

```
每 10 秒执行一次:

1. 查询借贷仓位:
   SELECT * FROM lending_user_positions
   WHERE health_factor < 1.2

2. 查询永续仓位:
   SELECT * FROM futures_positions
   WHERE status = 'open' AND margin 接近清算线

3. 将高风险仓位写入 liquidation_candidates 表

4. 如果发现危险仓位:
   - 写入 alert_records 告警表
   - Bot 可以进一步触发链上清算交易（需要配置钱包）
```

---

## 4. 日统计聚合

```
每天凌晨 1:00 UTC:

1. DEX 日统计:
   FOR EACH pair:
     SELECT SUM(amount_usd), COUNT(*) FROM dex_trades
     WHERE block_time BETWEEN yesterday AND today
     → INSERT INTO dex_pair_stats_daily

2. 用户资产快照:
   FOR EACH active user:
     INSERT INTO user_asset_snapshots
     (用于计算每日 PnL 曲线)
```

---

## 5. 任务运行记录

每个任务执行后都会写入 `job_runs` 表：
```sql
INSERT INTO job_runs (job_name, status, error_message, duration_ms, started_at, finished_at)
```

通过这张表可以监控各任务的执行情况：哪些失败了、耗时多久。

---

## 6. 数据库表

| 表 | 说明 |
|----|------|
| `job_runs` | 任务执行记录 |
| `alert_records` | 告警记录（清算预警、异常事件） |
| `liquidation_candidates` | 清算候选列表（高风险仓位） |
