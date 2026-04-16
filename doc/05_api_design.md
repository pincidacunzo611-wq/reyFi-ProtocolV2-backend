# 05 — API 设计

> 本文档定义 ReyFi Gateway 对外提供的 REST API 设计原则、接口分组和典型返回结构。

---

## 一、总体原则

### 1.1 面向前端，而不是面向合约

API 不应该简单镜像合约函数，而应该直接返回前端页面需要的数据结构。

例如前端想展示"我的总资产"，API 就应该直接返回总资产，而不是要求前端自己去调用 6 个接口再拼。

```
❌ 前端调 6 个接口自己拼总资产：
   GET /api/dex/positions
   GET /api/lending/positions
   GET /api/futures/positions
   GET /api/options/positions
   GET /api/vault/positions
   GET /api/bonds/positions

✅ 后端聚合好一次返回：
   GET /api/user/portfolio
```

### 1.2 读写分离

- **读接口（GET）**
  查询数据库和缓存，追求稳定和快。
- **写接口（POST）**
  不直接代替用户发交易，而是返回交易构造结果，交给前端钱包签名。

### 1.3 路径按业务域划分

```text
/api/{module}/{resource}

示例：
/api/dex/pairs            → DEX 交易对
/api/lending/markets       → 借贷市场
/api/futures/positions     → 期货仓位
/api/user/portfolio        → 用户总览
```

### 1.4 HTTP 方法语义

| 方法 | 用途 | 示例 |
|------|------|------|
| `GET` | 查询数据 | 获取交易对列表 |
| `POST` | 构造交易 / 提交操作 | 构造 Swap 交易参数 |
| `PUT` | 更新资源 | 更新用户偏好设置 |
| `DELETE` | 删除资源 | 删除收藏（如有） |

### 1.5 命名规范

- URL 路径全小写，使用 `-` 连接多个单词：`/api/dex/swap/build`
- 查询参数使用 camelCase：`?pageSize=20&pairAddress=0x...`
- 响应字段使用 camelCase：`{ "pairAddress": "0x..." }`
- 交易构造类接口统一后缀 `/build`：`/api/dex/swap/build`

---

## 二、统一响应格式

### 2.1 成功响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "traceId": "abc123def456"
}
```

### 2.2 分页响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [],
    "pagination": {
      "page": 1,
      "pageSize": 20,
      "total": 120,
      "totalPages": 6
    }
  }
}
```

### 2.3 错误响应

```json
{
  "code": 3001,
  "message": "交易对不存在",
  "data": null,
  "traceId": "abc123def456"
}
```

### 2.4 分页参数约定

| 参数 | 类型 | 默认值 | 说明 |
|------|------|-------|------|
| `page` | int | 1 | 页码，从 1 开始 |
| `pageSize` | int | 20 | 每页数量，最大 100 |

---

## 三、鉴权设计

### 3.1 不需要鉴权的接口（公开数据）

| 分类 | 示例接口 |
|------|---------|
| 市场行情 | `GET /api/dex/pairs`、`GET /api/dex/trades` |
| 协议概览 | `GET /api/dex/overview`、`GET /api/lending/markets` |
| 提案列表 | `GET /api/governance/proposals` |
| 金库列表 | `GET /api/vault/list` |
| 系统状态 | `GET /api/system/sync-status` |

### 3.2 需要鉴权的接口（用户私有数据）

| 分类 | 示例接口 |
|------|---------|
| 我的资产 | `GET /api/user/portfolio` |
| 我的头寸 | `GET /api/dex/positions`、`GET /api/lending/positions` |
| 我的历史 | `GET /api/user/activity` |
| 交易构造 | `POST /api/dex/swap/build` |
| 偏好设置 | `PUT /api/user/settings` |

### 3.3 钱包登录流程

```
┌──────────┐        ┌──────────┐        ┌──────────┐
│   前端   │        │ Gateway  │        │ User RPC │
└────┬─────┘        └────┬─────┘        └────┬─────┘
     │  1. 请求 nonce     │                   │
     │──────────────────→│                   │
     │                    │  2. 生成随机 nonce │
     │                    │──────────────────→│
     │                    │←─────────────────│
     │  3. 返回 nonce     │                   │
     │←──────────────────│                   │
     │                    │                   │
     │  4. 前端用钱包签名   │                   │
     │                    │                   │
     │  5. 提交签名        │                   │
     │──────────────────→│                   │
     │                    │  6. 验证签名       │
     │                    │──────────────────→│
     │                    │  7. 签发 JWT       │
     │                    │←─────────────────│
     │  8. 返回 JWT       │                   │
     │←──────────────────│                   │
```

#### Step 1: 获取 nonce

```
GET /api/user/auth/nonce?address=0xabc...
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "address": "0xabc",
    "nonce": "Sign this message to login to ReyFi:\n\nNonce: 938271\nTimestamp: 2026-04-15T08:00:00Z",
    "expiresIn": 300
  }
}
```

#### Step 2: 提交签名

```
POST /api/user/auth/login
Content-Type: application/json

{
  "address": "0xabc",
  "message": "Sign this message to login to ReyFi:\n\nNonce: 938271\nTimestamp: 2026-04-15T08:00:00Z",
  "signature": "0xsignature..."
}
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "dGhpcyBpcyBhIHJlZnJl...",
    "expiresIn": 7200
  }
}
```

#### Step 3: 请求头携带 Token

```
GET /api/user/portfolio
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

#### Step 4: 刷新 Token

```
POST /api/user/auth/refresh
Content-Type: application/json

{
  "refreshToken": "dGhpcyBpcyBhIHJlZnJl..."
}
```

---

## 四、DEX API

### 4.1 查询交易对列表

```
GET /api/dex/pairs?page=1&pageSize=20&keyword=ETH&sortBy=volume24h&sortOrder=desc
```

查询参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20 |
| `keyword` | string | 否 | 按 token symbol 搜索 |
| `token0` | string | 否 | 按 token0 地址过滤 |
| `token1` | string | 否 | 按 token1 地址过滤 |
| `sortBy` | string | 否 | 排序字段：`volume24h` / `tvl` / `apr` |
| `sortOrder` | string | 否 | `asc` / `desc`，默认 `desc` |

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "pairAddress": "0x...",
        "token0": { "address": "0x...", "symbol": "WETH", "decimals": 18, "logoUrl": "..." },
        "token1": { "address": "0x...", "symbol": "USDC", "decimals": 6, "logoUrl": "..." },
        "price": "3250.12",
        "priceChange24h": "-2.35",
        "volume24h": "1250000.00",
        "tvl": "15000000.00",
        "feeApr": "12.5",
        "feeBps": 30
      }
    ],
    "pagination": { "page": 1, "pageSize": 20, "total": 85 }
  }
}
```

### 4.2 查询交易对详情

```
GET /api/dex/pairs/{pairAddress}
```

响应包含比列表更详细的数据：储备量、累计手续费、24h最高最低价等。

### 4.3 查询成交记录

```
GET /api/dex/trades?pairAddress=0x...&page=1&pageSize=50
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `pairAddress` | string | 否 | 按交易对过滤 |
| `address` | string | 否 | 按用户地址过滤（需鉴权） |
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 50 |

响应示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "txHash": "0x...",
        "pairAddress": "0x...",
        "traderAddress": "0x...",
        "direction": "buy",
        "amount0": "1.5",
        "amount1": "4875.18",
        "amountUsd": "4875.18",
        "price": "3250.12",
        "blockTime": "2026-04-15T08:00:00Z"
      }
    ],
    "pagination": { "page": 1, "pageSize": 50, "total": 1200 }
  }
}
```

### 4.4 查询 K 线

```
GET /api/dex/candles?pairAddress=0x...&interval=1h&from=1710000000&to=1710086400
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `pairAddress` | string | 是 | 交易对地址 |
| `interval` | string | 是 | `1m` / `5m` / `15m` / `1h` / `4h` / `1d` |
| `from` | int64 | 是 | 开始时间（Unix 秒） |
| `to` | int64 | 是 | 结束时间（Unix 秒） |

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "candles": [
      {
        "time": 1710000000,
        "open": "3240.00",
        "high": "3260.50",
        "low": "3235.00",
        "close": "3250.12",
        "volume": "125000.00"
      }
    ]
  }
}
```

### 4.5 查询用户 LP 头寸 🔒

```
GET /api/dex/positions
Authorization: Bearer <JWT>
```

### 4.6 构造 Swap 交易 🔒

```
POST /api/dex/swap/build
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "tokenIn": "0xTokenA",
  "tokenOut": "0xTokenB",
  "amountIn": "1.5",
  "slippageBps": 50,
  "receiver": "0xabc"
}
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "to": "0xRouter",
    "value": "0",
    "data": "0xabcdef...",
    "gasLimit": "250000",
    "minAmountOut": "2980.12",
    "priceImpact": "0.15",
    "route": ["0xTokenA", "0xUSDC", "0xTokenB"],
    "estimatedGasUsd": "2.50"
  }
}
```

### 4.7 DEX 概览数据

```
GET /api/dex/overview
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "totalTvl": "150000000.00",
    "totalVolume24h": "25000000.00",
    "totalFees24h": "75000.00",
    "totalPairs": 85,
    "topPairs": [ ... ]
  }
}
```

---

## 五、Lending API

### 5.1 查询借贷市场列表

```
GET /api/lending/markets
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "assetAddress": "0x...",
        "assetSymbol": "WETH",
        "totalSupply": "50000.00",
        "totalBorrow": "30000.00",
        "utilizationRate": "0.60",
        "supplyApr": "3.25",
        "borrowApr": "5.80",
        "collateralFactor": "0.75",
        "liquidationThreshold": "0.80",
        "tvlUsd": "162500000.00"
      }
    ]
  }
}
```

### 5.2 查询单个市场详情

```
GET /api/lending/markets/{assetAddress}
```

响应包含：`totalSupply` / `totalBorrow` / `utilizationRate` / `supplyApr` / `borrowApr` / `collateralFactor` / `liquidationThreshold` / 利率曲线参数 / 历史 APR 趋势。

### 5.3 查询我的借贷仓位 🔒

```
GET /api/lending/positions
```

### 5.4 查询我的健康度 🔒

```
GET /api/lending/health
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "healthFactor": "1.85",
    "totalCollateralUsd": "100000.00",
    "totalBorrowUsd": "54000.00",
    "availableBorrowUsd": "21000.00",
    "liquidationRisk": "low",
    "positions": [
      {
        "asset": "WETH",
        "supplied": "30.0",
        "borrowed": "0",
        "isCollateral": true
      },
      {
        "asset": "USDC",
        "supplied": "0",
        "borrowed": "54000.00",
        "isCollateral": false
      }
    ]
  }
}
```

### 5.5 构造存款交易 🔒

```
POST /api/lending/deposit/build
```

### 5.6 构造借款交易 🔒

```
POST /api/lending/borrow/build
```

### 5.7 构造还款交易 🔒

```
POST /api/lending/repay/build
```

---

## 六、Futures / Leverage API

### 6.1 查询市场列表

```
GET /api/futures/markets
```

### 6.2 查询市场详情

```
GET /api/futures/markets/{market}
```

响应包含：最新标记价、指数价、24h涨跌、未平仓量、最大杠杆、当前资金费率等。

### 6.3 查询用户仓位 🔒

```
GET /api/futures/positions
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "positionId": "12345",
        "market": "ETH-USD",
        "side": "long",
        "size": "10.0",
        "entryPrice": "3200.00",
        "markPrice": "3250.12",
        "margin": "6400.00",
        "leverage": "5.0",
        "unrealizedPnl": "501.20",
        "unrealizedPnlPercent": "7.83",
        "liquidationPrice": "2720.00",
        "marginRatio": "0.215"
      }
    ]
  }
}
```

### 6.4 查询仓位详情 🔒

```
GET /api/futures/positions/{positionId}
```

### 6.5 查询资金费率

```
GET /api/futures/funding?market=ETH-USD
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "currentRate": "0.0001",
    "nextSettlement": "2026-04-15T16:00:00Z",
    "countdown": 3600,
    "history": [
      { "rate": "0.00012", "time": "2026-04-15T08:00:00Z" },
      { "rate": "-0.00005", "time": "2026-04-15T00:00:00Z" }
    ]
  }
}
```

### 6.6 构造开仓交易 🔒

```
POST /api/futures/open/build

{
  "market": "ETH-USD",
  "side": "long",
  "margin": "1000",
  "leverage": "5",
  "acceptablePrice": "3300"
}
```

### 6.7 构造平仓交易 🔒

```
POST /api/futures/close/build

{
  "positionId": "12345",
  "acceptablePrice": "3200"
}
```

---

## 七、Options API

### 7.1 查询期权市场

```
GET /api/options/markets
```

### 7.2 查询报价

```
POST /api/options/quote

{
  "underlying": "ETH",
  "strikePrice": "3200",
  "expiry": 1715000000,
  "optionType": "put",
  "size": "1"
}
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "premium": "85.50",
    "premiumUsd": "85.50",
    "iv": "0.65",
    "delta": "-0.35",
    "gamma": "0.012",
    "theta": "-4.20",
    "vega": "5.80",
    "breakEvenPrice": "3114.50",
    "maxLoss": "85.50",
    "maxProfit": "3114.50"
  }
}
```

### 7.3 查询我的期权仓位 🔒

```
GET /api/options/positions
```

### 7.4 构造购买交易 🔒

```
POST /api/options/buy/build
```

### 7.5 构造行权交易 🔒

```
POST /api/options/exercise/build
```

---

## 八、Vault API

### 8.1 查询金库列表

```
GET /api/vault/list
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "vaultAddress": "0x...",
        "name": "ETH Lending Yield",
        "strategyType": "lending",
        "asset": { "address": "0x...", "symbol": "WETH" },
        "tvl": "5000000.00",
        "nav": "1.0523",
        "apr7d": "8.25",
        "apr30d": "7.80",
        "maxDrawdown": "0.5",
        "isActive": true
      }
    ]
  }
}
```

### 8.2 查询金库详情

```
GET /api/vault/{vaultAddress}
```

### 8.3 查询金库净值和收益曲线

```
GET /api/vault/{vaultAddress}/performance?period=30d
```

| 参数 | 说明 |
|------|------|
| `period` | `7d` / `30d` / `90d` / `1y` / `all` |

### 8.4 查询我的金库持仓 🔒

```
GET /api/vault/positions
```

### 8.5 构造申购交易 🔒

```
POST /api/vault/deposit/build
```

### 8.6 构造赎回交易 🔒

```
POST /api/vault/withdraw/build
```

---

## 九、Bonds API

### 9.1 查询债券市场列表

```
GET /api/bonds/markets
```

### 9.2 查询债券市场详情

```
GET /api/bonds/markets/{marketAddress}
```

响应包含：折价率、归属期、剩余容量、当前 ROI 等。

### 9.3 查询我的债券仓位 🔒

```
GET /api/bonds/positions
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "bondNftId": "42",
        "marketAddress": "0x...",
        "paidToken": "USDC",
        "paidAmount": "1000.00",
        "payoutToken": "REY",
        "payoutAmount": "1200.00",
        "claimedAmount": "600.00",
        "claimableAmount": "150.00",
        "vestingProgress": "0.625",
        "vestingEnd": "2026-05-15T00:00:00Z"
      }
    ]
  }
}
```

### 9.4 构造购买债券交易 🔒

```
POST /api/bonds/purchase/build
```

### 9.5 构造兑付交易 🔒

```
POST /api/bonds/redeem/build
```

---

## 十、Governance API

### 10.1 查询提案列表

```
GET /api/governance/proposals?status=active&page=1&pageSize=10
```

### 10.2 查询提案详情

```
GET /api/governance/proposals/{proposalId}
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "proposalId": "1",
    "proposer": "0x...",
    "title": "增加 ETH/USDC 池子激励权重",
    "description": "...",
    "status": "active",
    "forVotes": "1250000.00",
    "againstVotes": "320000.00",
    "abstainVotes": "50000.00",
    "quorum": "2000000.00",
    "quorumReached": false,
    "startTime": "2026-04-10T00:00:00Z",
    "endTime": "2026-04-17T00:00:00Z",
    "countdown": 172800,
    "participationRate": "0.12"
  }
}
```

### 10.3 查询投票记录

```
GET /api/governance/proposals/{proposalId}/votes?page=1&pageSize=50
```

### 10.4 查询我的治理信息 🔒

```
GET /api/governance/me
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "reyBalance": "10000.00",
    "veReyBalance": "8500.00",
    "votingPower": "8500.00",
    "activeLocks": [
      {
        "lockId": "1",
        "lockedAmount": "10000.00",
        "veBalance": "8500.00",
        "lockEnd": "2027-04-15T00:00:00Z"
      }
    ],
    "claimableBribes": [
      {
        "pool": "ETH/USDC",
        "rewardToken": "REY",
        "amount": "120.00"
      }
    ]
  }
}
```

### 10.5 构造投票交易 🔒

```
POST /api/governance/vote/build
```

### 10.6 构造锁仓交易 🔒

```
POST /api/governance/lock/build
```

### 10.7 构造 gauge 投票交易 🔒

```
POST /api/governance/gauge-vote/build
```

---

## 十一、User API

### 11.1 查询我的总览 🔒

```
GET /api/user/portfolio
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "walletAddress": "0xabc",
    "totalAssetUsd": "125430.22",
    "totalDebtUsd": "21500.00",
    "netValueUsd": "103930.22",
    "pnl24h": "1820.12",
    "pnl24hPercent": "1.78",
    "allocation": [
      { "module": "dex", "label": "流动性池", "valueUsd": "12000.00", "percent": "11.5" },
      { "module": "lending", "label": "借贷", "valueUsd": "50000.00", "percent": "48.1" },
      { "module": "futures", "label": "永续", "valueUsd": "20000.00", "percent": "19.2" },
      { "module": "vault", "label": "金库", "valueUsd": "15000.00", "percent": "14.4" },
      { "module": "bonds", "label": "债券", "valueUsd": "5000.00", "percent": "4.8" },
      { "module": "governance", "label": "治理", "valueUsd": "2430.22", "percent": "2.3" }
    ],
    "riskSummary": {
      "lendingHealthFactor": "1.85",
      "futuresMarginRatio": "0.215",
      "riskLevel": "low"
    }
  }
}
```

### 11.2 查询活动流 🔒

```
GET /api/user/activity?page=1&pageSize=20&module=dex
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 12345,
        "module": "dex",
        "action": "swap",
        "summary": "用 1.5 ETH 兑换 4875.18 USDC",
        "amountUsd": "4875.18",
        "txHash": "0x...",
        "blockTime": "2026-04-15T08:00:00Z"
      }
    ],
    "pagination": { "page": 1, "pageSize": 20, "total": 350 }
  }
}
```

### 11.3 查询通知 🔒

```
GET /api/user/notifications?unreadOnly=true
```

### 11.4 更新偏好设置 🔒

```
PUT /api/user/settings

{
  "nickname": "DeFi Whale",
  "language": "zh-CN",
  "currency": "USD",
  "slippageDefault": 50
}
```

---

## 十二、系统和运维 API

> 这一组接口一般只给后台或内部使用，建议通过 IP 白名单或管理员 Token 保护。

### 12.1 查询同步状态

```
GET /api/system/sync-status
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "chainHeight": 19500000,
    "modules": [
      {
        "module": "dex",
        "lastScannedBlock": 19499980,
        "lastConfirmedBlock": 19499968,
        "lag": 20,
        "status": "running"
      },
      {
        "module": "lending",
        "lastScannedBlock": 19499975,
        "lastConfirmedBlock": 19499963,
        "lag": 25,
        "status": "running"
      }
    ]
  }
}
```

### 12.2 查询任务执行记录

```
GET /api/system/jobs?status=failed&page=1&pageSize=20
```

### 12.3 手动触发补块

```
POST /api/system/reindex

{
  "module": "dex",
  "fromBlock": 19400000,
  "toBlock": 19400100
}
```

---

## 十三、实时数据推送（WebSocket）

> 行情和仓位变化等高频数据建议通过 WebSocket 推送，减少轮询压力。

### 13.1 连接地址

```
ws://api.reyfi.io/ws
```

### 13.2 订阅消息格式

```json
{
  "action": "subscribe",
  "channel": "dex.trades",
  "params": {
    "pairAddress": "0x..."
  }
}
```

### 13.3 推送消息格式

```json
{
  "channel": "dex.trades",
  "data": {
    "pairAddress": "0x...",
    "direction": "buy",
    "price": "3251.20",
    "amount": "0.5",
    "amountUsd": "1625.60",
    "txHash": "0x...",
    "blockTime": "2026-04-15T08:00:05Z"
  },
  "timestamp": 1710000005
}
```

### 13.4 建议的 WebSocket 频道

| 频道 | 说明 | 推送频率 |
|------|------|---------|
| `dex.trades:{pairAddress}` | 实时成交 | 每笔交易 |
| `dex.price:{pairAddress}` | 价格变动 | 每 1-3 秒 |
| `futures.price:{market}` | 期货标记价格 | 每 1 秒 |
| `futures.positions:{address}` | 仓位变动（需鉴权） | 事件驱动 |
| `lending.rates` | 利率变动 | 每 30 秒 |
| `system.blocks` | 新区块通知 | 每个区块 |

> **Phase 1 不需要实现 WebSocket**。先用前端轮询（间隔 5-10 秒），Phase 3+ 再加 WebSocket。

---

## 十四、限流策略

| 接口类型 | 限流规则 | 说明 |
|---------|---------|------|
| 公开查询 | 60 次/分钟/IP | 市场行情等 |
| 登录后查询 | 120 次/分钟/用户 | 用户私有数据 |
| 交易构造 | 20 次/分钟/用户 | 防止频繁构造 |
| WebSocket | 10 次/秒/连接 | 订阅和取消订阅 |
| 系统管理 | 10 次/分钟/IP | 管理接口 |

超限响应：

```json
{
  "code": 1429,
  "message": "请求过于频繁，请稍后再试",
  "data": {
    "retryAfter": 30
  }
}
```

---

## 十五、状态码总表

| code | 含义 | HTTP Status |
|------|------|-------------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 分页参数非法 | 400 |
| 1429 | 请求过于频繁 | 429 |
| 2001 | 未登录 | 401 |
| 2002 | 签名无效 | 401 |
| 2003 | Token 已过期 | 401 |
| 2004 | 账号已封禁 | 403 |
| 3001 | 交易对不存在 | 404 |
| 3002 | 滑点超限 | 400 |
| 3003 | 路由不可用 | 400 |
| 4001 | 借贷市场不存在 | 404 |
| 4002 | 健康度不足 | 400 |
| 4003 | 超过可借额度 | 400 |
| 5001 | 仓位不存在 | 404 |
| 5002 | 保证金不足 | 400 |
| 5003 | 杠杆超限 | 400 |
| 6001 | 期权已到期 | 400 |
| 6002 | 不满足行权条件 | 400 |
| 7001 | 金库不存在 | 404 |
| 7002 | 金库已暂停 | 400 |
| 8001 | 提案不存在 | 404 |
| 8002 | 投票期已结束 | 400 |
| 8003 | 投票权不足 | 400 |
| 9001 | 系统繁忙 | 503 |
| 9002 | 服务不可用 | 503 |
| 9003 | 链上节点异常 | 502 |

---

## 十六、版本策略

建议在网关层预留版本：

```text
/api/v1/dex/pairs
/api/v1/lending/markets
```

当前项目还在快速迭代中，建议策略：

| 阶段 | 路径 | 说明 |
|------|------|------|
| Phase 1-2 | `/api/...` | 内部开发，不需要版本号 |
| Phase 3+ | `/api/v1/...` | 前端稳定后加版本号 |
| 破坏性变更 | `/api/v2/...` | v1 和 v2 共存过渡期 |

---

## 十七、最重要的结论

1. **查询接口返回前端视角的数据，不要让前端自己拼链上状态**
2. **写接口返回交易构造结果，不代用户托管私钥或发交易**
3. **用户总览必须由 `user` 域统一聚合，而不是每个页面自己拼**
4. **所有模块都要区分公开查询接口和登录后私有接口**（🔒 标记）
5. **响应结构统一，错误码分层，降低前端对接成本**
6. **实时数据先用轮询，后期再加 WebSocket**
