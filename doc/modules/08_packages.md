# 公共包 (pkg/) — 详细说明

## 1. pkg/chains/events.go — 统一事件消息

这是整个项目最核心的数据结构，所有链上事件都通过它在 Kafka 中传输。

### ChainEvent 结构体

```go
type ChainEvent struct {
    ChainId     int64           // 链 ID (31337=本地, 1=以太坊主网)
    Module      string          // 模块名: "dex" / "lending" / "futures" ...
    EventName   string          // 事件名: "Swap" / "Deposit" / "PositionOpened" ...
    Contract    string          // 产生事件的合约地址
    TxHash      string          // 交易哈希（唯一标识一笔链上交易）
    TxIndex     int             // 交易在区块中的位置
    LogIndex    int             // 事件在交易中的位置
    BlockNumber int64           // 区块号
    BlockHash   string          // 区块哈希
    BlockTime   int64           // 区块时间 (Unix 秒)
    Status      EventStatus     // pending → confirmed → reverted
    Payload     json.RawMessage // 解析后的事件参数（每种事件不同）
    Topic0      string          // 事件签名哈希（用于识别事件类型）
}
```

### 幂等 Key

```go
func (e *ChainEvent) UniqueKey() string {
    return fmt.Sprintf("%d:%s:%d", e.ChainId, e.TxHash, e.LogIndex)
}
// 例: "31337:0xabc123:0"
// 同一条链上同一笔交易的同一个事件日志，全局唯一
```

### Payload 类型清单

| 事件 | Payload 结构 | 关键字段 |
|------|-------------|---------|
| Swap | SwapPayload | sender, to, amount0In/Out, amount1In/Out |
| Sync | SyncPayload | reserve0, reserve1 |
| Mint | MintPayload | sender, amount0, amount1 |
| Burn | BurnPayload | sender, to, amount0, amount1 |
| PairCreated | PairCreatedPayload | token0, token1, pairAddress |
| Deposit | DepositPayload | user, asset, amount |
| Withdraw | WithdrawPayload | user, asset, amount, to |
| Borrow | BorrowPayload | user, asset, amount, borrowRate |
| Repay | RepayPayload | user, repayer, asset, amount |
| Liquidate | LiquidatePayload | liquidator, borrower, collateral/debt amounts |
| PositionOpened | PositionOpenedPayload | user, positionId, side, size, leverage |
| OptionPurchased | OptionPurchasedPayload | buyer, optionType, strikePrice, premium |
| BondPurchased | BondPurchasedPayload | bondId, buyer, payment/payout amounts |
| BondRedeemed | BondRedeemedPayload | bondId, user, amount |
| ProposalCreated | ProposalCreatedPayload | proposalId, proposer, description |
| VoteCast | VoteCastPayload | voter, proposalId, support, weight |
| LockCreated | LockCreatedPayload | user, amount, unlockTime |

---

## 2. pkg/chains/client.go — ETH 客户端

封装 go-ethereum 的 `ethclient`，提供以下能力：

```go
type ChainClient struct {
    client    *ethclient.Client
    chainId   int64
    contracts map[string]common.Address  // 合约名 → 地址映射
}

// 核心方法:
LatestBlockNumber()         // 查最新区块号
HeaderByNumber(blockNum)    // 获取区块头
FilterLogs(query)           // 过滤事件日志 (核心方法)
GetAllContractAddresses()   // 返回所有监听的合约地址列表
GetContract(name)           // 按名称查合约地址
```

---

## 3. pkg/response/response.go — 统一响应

所有 HTTP 响应统一使用此包：

```go
// 成功
response.Success(ctx, w, data)
// → { "code": 0, "message": "ok", "data": {...}, "traceId": "..." }

// 错误
response.Error(ctx, w, err)
// → { "code": 1001, "message": "参数错误", "traceId": "..." }

// 分页
response.SuccessWithPage(ctx, w, list, total, page, pageSize)
```

---

## 4. pkg/errorx/errorx.go — 错误码

```
错误码分段:
  1000-1999  通用错误 (参数错误、未登录、无权限)
  2000-2999  DEX 模块
  3000-3999  Lending 模块
  4000-4999  Futures 模块
  5000-5999  Options 模块
  6000-6999  Vault 模块
  7000-7999  Bonds 模块
  8000-8999  Governance 模块
  9000-9999  系统/内部错误
```

使用方式：
```go
return errorx.New(errorx.CodeSignatureInvalid, "签名验证失败")
```

---

## 5. pkg/mathx/bigint.go — 大数工具

DeFi 中所有金额都是 uint256（最大 2^256），不能用 float64。

```go
// 链上金额(wei) → 可读金额
FormatAmount("1000000000000000000", 18) → "1.0"

// 计算价格
CalcPrice(reserve0, reserve1, decimals0, decimals1) → "3000.5"

// 百分比计算
CalcPercentChange(oldValue, newValue) → "2.35"
```

---

## 6. pkg/middleware/auth.go — JWT 中间件

```go
// 创建中间件
authMiddleware := middleware.AuthMiddleware(jwtSecret)

// 保护路由
Handler: authMiddleware(myHandler)

// 在 Handler 中获取当前用户
walletAddress := middleware.GetWalletAddress(r.Context())

// JWT Claims 内容:
{
    "userId": 42,
    "walletAddress": "0xABC...",
    "exp": 1713254400       // 过期时间
}
```

---

## 7. 各模块 Kafka Topic

| Topic | 生产者 | 消费者 |
|-------|--------|--------|
| `reyfi.dex.events` | Chain Indexer | DEX Service |
| `reyfi.lending.events` | Chain Indexer | Lending Service |
| `reyfi.futures.events` | Chain Indexer | Futures Service |
| `reyfi.options.events` | Chain Indexer | Options Service |
| `reyfi.vault.events` | Chain Indexer | Vault Service |
| `reyfi.bonds.events` | Chain Indexer | Bonds Service |
| `reyfi.governance.events` | Chain Indexer | Governance Service |
| `reyfi.user.events` | Chain Indexer | User Service |

每个 Topic 的消息格式都是 `ChainEvent` JSON。消费者通过 `event.EventName` 区分不同类型的事件。
