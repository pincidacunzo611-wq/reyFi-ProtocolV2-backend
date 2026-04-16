# Futures 服务 — 永续合约

## 1. 概述

Futures 服务管理 **永续合约交易** 数据。用户可以用杠杆做多或做空某种资产，而不需要实际买入资产。

> **类比理解**：期货交易所。你觉得 ETH 会涨，就开多单 10x 杠杆。你只需要 100 USDC 保证金，就能获得 1000 USDC 的多头敞口。涨了赚 10 倍利润，跌了也亏 10 倍。

---

## 2. 核心概念

| 概念 | 解释 |
|------|------|
| **多 Long** | 做多，赌价格上涨 |
| **空 Short** | 做空，赌价格下跌 |
| **杠杆 Leverage** | 放大倍数 (1x ~ 100x) |
| **保证金 Margin** | 开仓时抵押的本金 |
| **标记价格 Mark Price** | 用于计算盈亏的价格 (防操纵) |
| **资金费率 Funding Rate** | 多空双方定时互付（平衡多空比例） |
| **爆仓价 Liquidation Price** | 亏损触及保证金时的强制平仓价格 |
| **未实现盈亏 PnL** | 浮盈/浮亏 (尚未平仓) |

---

## 3. 事件消费

| 事件 | 处理逻辑 |
|------|---------|
| **PositionOpened** | 写入 `futures_positions` 表 (status=open) |
| **PositionClosed** | 更新仓位状态为 closed，计算实现盈亏 |
| **FundingSettled** | 记录资金费率到 `futures_funding_records` |
| **PositionLiquidated** | 记录清算 + 标记仓位为 liquidated |

---

## 4. 对外接口

| RPC 方法 | 功能 |
|----------|------|
| `GetMarkets` | 永续合约市场列表（标记价格、资金费率、OI） |
| `GetMarketDetail` | 市场详情 + 费率 + 保证金要求 |
| `GetPositions` | 用户当前持仓（未实现盈亏、保证金率） |
| `GetFundingHistory` | 资金费率历史记录 |
| `BuildOpenPosition` | 构造开仓交易 |
| `BuildClosePosition` | 构造平仓交易 |
| `BuildAdjustMargin` | 追加/减少保证金 |

---

## 5. 数据库表

| 表 | 说明 |
|----|------|
| `futures_markets` | 市场配置（最大杠杆、手续费率、保证金要求） |
| `futures_positions` | 用户持仓（方向、大小、入场价、保证金、盈亏、状态） |
| `futures_funding_records` | 资金费率结算记录 |
| `futures_liquidations` | 强制平仓记录 |

---

## 6. 资金费率

```
资金费率 > 0  →  多头付空头（多单太多了，鼓励做空）
资金费率 < 0  →  空头付多头（空单太多了，鼓励做多）
资金费率 ≈ 0  →  多空平衡

结算公式：
  费用 = 仓位大小 × 资金费率
  每 8 小时结算一次
```

# Options 服务 — 期权

## 1. 概述

Options 服务管理 **链上期权** 数据。期权给予持有者在特定日期以特定价格买/卖资产的"权利"（而非义务）。

> **类比理解**：保险。你花 50 USDC 买一份看跌期权（相当于保险），保证自己能在 1 个月后以 3000 USDC 的价格卖出 1 ETH。如果 ETH 跌到 2000，你可以用 3000 的价格卖出（赚 950）。如果 ETH 涨了，保险到期作废（只亏 50 的权利金）。

---

## 2. 核心概念

| 概念 | 解释 |
|------|------|
| **看涨期权 Call** | 认为价格会涨，买入做多的权利 |
| **看跌期权 Put** | 认为价格会跌，买入做空的权利 |
| **行权价 Strike** | 约定的买卖价格 |
| **权利金 Premium** | 购买期权的成本 |
| **到期日 Expiry** | 期权失效的日期 |
| **隐含波动率 IV** | 市场对价格波动的预期 |

---

## 3. 事件消费

| 事件 | 处理逻辑 |
|------|---------|
| **OptionPurchased** | 写入 `options_positions` (status=open) |
| **OptionExercised** | 更新仓位状态为 exercised，记录结算价和盈亏 |
| **OptionExpired** | 标记到期未行权的期权 |
| **SettlementExecuted** | 记录结算详情 |

---

## 4. 数据库表

| 表 | 说明 |
|----|------|
| `options_markets` | 期权市场（标的资产、结算资产） |
| `options_positions` | 用户持仓（类型、行权价、权利金、到期日、盈亏） |
| `options_settlements` | 结算/行权记录 |
| `options_vol_surfaces` | 波动率曲面数据（不同行权价×到期日的 IV） |
