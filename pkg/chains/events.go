// Package chains 定义链上事件的统一消息结构。
// 所有模块的事件都使用同一结构，方便 Kafka 传输和消费端解析。
package chains

import (
	"encoding/json"
	"fmt"
	"time"
)

// ChainEvent 统一链上事件消息结构
// 这是 Indexer 产出的标准消息格式，所有业务 RPC 服务消费此结构
type ChainEvent struct {
	ChainId       int64             `json:"chainId"`
	Module        string            `json:"module"`        // 所属模块: dex / lending / futures / options / vault / bonds / governance / user
	EventName     string            `json:"eventName"`     // 事件名: Swap / Mint / Burn / Deposit / Withdraw 等
	Contract      string            `json:"contract"`      // 合约地址
	TxHash        string            `json:"txHash"`        // 交易哈希
	TxIndex       int               `json:"txIndex"`       // 交易在区块中的索引
	LogIndex      int               `json:"logIndex"`      // 日志在交易中的索引
	BlockNumber   int64             `json:"blockNumber"`   // 区块号
	BlockHash     string            `json:"blockHash"`     // 区块哈希
	BlockTime     int64             `json:"blockTime"`     // 区块时间 (Unix 秒)
	Status        EventStatus       `json:"status"`        // 确认状态
	Confirmations int64             `json:"confirmations"` // 确认块数
	Payload       json.RawMessage   `json:"payload"`       // 解析后的事件参数 (JSON)
	Topic0        string            `json:"topic0"`        // 事件签名哈希
}

// EventStatus 事件确认状态
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusReverted  EventStatus = "reverted"
)

// KafkaTopics 定义各模块的 Kafka 主题
const (
	TopicDexEvents        = "reyfi.dex.events"
	TopicLendingEvents    = "reyfi.lending.events"
	TopicFuturesEvents    = "reyfi.futures.events"
	TopicOptionsEvents    = "reyfi.options.events"
	TopicVaultEvents      = "reyfi.vault.events"
	TopicBondsEvents      = "reyfi.bonds.events"
	TopicGovernanceEvents = "reyfi.governance.events"
	TopicUserEvents       = "reyfi.user.events"
)

// ModuleTopicMap 模块名到 Kafka 主题的映射
var ModuleTopicMap = map[string]string{
	"dex":        TopicDexEvents,
	"lending":    TopicLendingEvents,
	"futures":    TopicFuturesEvents,
	"options":    TopicOptionsEvents,
	"vault":      TopicVaultEvents,
	"bonds":      TopicBondsEvents,
	"governance": TopicGovernanceEvents,
	"user":       TopicUserEvents,
}

// DEX 事件名常量
const (
	EventPairCreated = "PairCreated"
	EventMint        = "Mint"
	EventBurn        = "Burn"
	EventSwap        = "Swap"
	EventSync        = "Sync"
	EventRewardPaid  = "RewardPaid"
)

// Lending 事件名常量
const (
	EventDeposit         = "Deposit"
	EventWithdraw        = "Withdraw"
	EventBorrow          = "Borrow"
	EventRepay           = "Repay"
	EventLiquidate       = "Liquidate"
	EventCreditDelegated = "CreditDelegated"
)

// Futures 事件名常量
const (
	EventPositionOpened  = "PositionOpened"
	EventPositionChanged = "PositionChanged"
	EventPositionClosed  = "PositionClosed"
	EventFundingSettled  = "FundingSettled"
	EventLiquidated      = "Liquidated"
)

// Options 事件名常量
const (
	EventOptionPurchased    = "OptionPurchased"
	EventOptionExercised    = "OptionExercised"
	EventOptionExpired      = "OptionExpired"
	EventSettlementExecuted = "SettlementExecuted"
)

// Vault 事件名常量
const (
	EventVaultCreated   = "VaultCreated"
	EventStrategyUpdated = "StrategyUpdated"
	EventHarvest        = "Harvest"
)

// Bonds 事件名常量
const (
	EventBondCreated   = "BondCreated"
	EventBondPurchased = "BondPurchased"
	EventBondClaimed   = "BondClaimed"
	EventBondRedeemed  = "BondRedeemed"
)

// Governance 事件名常量
const (
	EventProposalCreated  = "ProposalCreated"
	EventVoteCast         = "VoteCast"
	EventProposalQueued   = "ProposalQueued"
	EventProposalExecuted = "ProposalExecuted"
	EventGaugeVoted       = "GaugeVoted"
	EventLockCreated      = "LockCreated"
	EventLockExtended     = "LockExtended"
)

// GetBlockTime 将 BlockTime 转换为 time.Time
func (e *ChainEvent) GetBlockTime() time.Time {
	return time.Unix(e.BlockTime, 0).UTC()
}

// UniqueKey 生成事件唯一标识，用于幂等判断
func (e *ChainEvent) UniqueKey() string {
	return fmt.Sprintf("%d:%s:%d", e.ChainId, e.TxHash, e.LogIndex)
}

// Marshal 序列化为 JSON
func (e *ChainEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalChainEvent 从 JSON 反序列化
func UnmarshalChainEvent(data []byte) (*ChainEvent, error) {
	var event ChainEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("unmarshal chain event: %w", err)
	}
	return &event, nil
}

// ===== 各模块事件 Payload 结构 =====

// SwapPayload Swap 事件参数
type SwapPayload struct {
	Sender     string `json:"sender"`
	To         string `json:"to"`
	Amount0In  string `json:"amount0In"`
	Amount1In  string `json:"amount1In"`
	Amount0Out string `json:"amount0Out"`
	Amount1Out string `json:"amount1Out"`
}

// MintPayload Mint (添加流动性) 事件参数
type MintPayload struct {
	Sender  string `json:"sender"`
	Amount0 string `json:"amount0"`
	Amount1 string `json:"amount1"`
}

// BurnPayload Burn (移除流动性) 事件参数
type BurnPayload struct {
	Sender  string `json:"sender"`
	To      string `json:"to"`
	Amount0 string `json:"amount0"`
	Amount1 string `json:"amount1"`
}

// SyncPayload Sync (储备量更新) 事件参数
type SyncPayload struct {
	Reserve0 string `json:"reserve0"`
	Reserve1 string `json:"reserve1"`
}

// PairCreatedPayload PairCreated 事件参数
type PairCreatedPayload struct {
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	PairAddress string `json:"pairAddress"`
	PairIndex   string `json:"pairIndex"`
}

// DepositPayload (Lending/Vault) 存款事件参数
type DepositPayload struct {
	User    string `json:"user"`
	Asset   string `json:"asset"`
	Amount  string `json:"amount"`
	OnBehalf string `json:"onBehalf,omitempty"`
}

// WithdrawPayload 取款事件参数
type WithdrawPayload struct {
	User   string `json:"user"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
	To     string `json:"to"`
}

// BorrowPayload 借款事件参数
type BorrowPayload struct {
	User       string `json:"user"`
	Asset      string `json:"asset"`
	Amount     string `json:"amount"`
	OnBehalf   string `json:"onBehalf,omitempty"`
	BorrowRate string `json:"borrowRate"`
}

// RepayPayload 还款事件参数
type RepayPayload struct {
	User     string `json:"user"`
	Repayer  string `json:"repayer"`
	Asset    string `json:"asset"`
	Amount   string `json:"amount"`
}

// LiquidatePayload 清算事件参数
type LiquidatePayload struct {
	Liquidator       string `json:"liquidator"`
	Borrower         string `json:"borrower"`
	CollateralAsset  string `json:"collateralAsset"`
	DebtAsset        string `json:"debtAsset"`
	CollateralAmount string `json:"collateralAmount"`
	DebtAmount       string `json:"debtAmount"`
}

// PositionOpenedPayload 期货开仓事件参数
type PositionOpenedPayload struct {
	User       string `json:"user"`
	PositionId string `json:"positionId"`
	Market     string `json:"market"`
	Side       string `json:"side"` // long / short
	Size       string `json:"size"`
	EntryPrice string `json:"entryPrice"`
	Margin     string `json:"margin"`
	Leverage   string `json:"leverage"`
}

// OptionPurchasedPayload 期权购买事件参数
type OptionPurchasedPayload struct {
	Buyer       string `json:"buyer"`
	OptionId    string `json:"optionId"`
	OptionType  string `json:"optionType"` // call / put
	StrikePrice string `json:"strikePrice"`
	Premium     string `json:"premium"`
	Size        string `json:"size"`
	Expiry      int64  `json:"expiry"`
}

// BondPurchasedPayload 债券购买事件参数
type BondPurchasedPayload struct {
	BondId        string `json:"bondId"`
	Buyer         string `json:"buyer"`
	PaymentAmount string `json:"paymentAmount"`
	PayoutAmount  string `json:"payoutAmount"`
}

// BondRedeemedPayload 债券赎回事件参数
type BondRedeemedPayload struct {
	BondId string `json:"bondId"`
	User   string `json:"user"`
	Amount string `json:"amount"`
}

// ProposalCreatedPayload 治理提案创建事件参数
type ProposalCreatedPayload struct {
	ProposalId  string `json:"proposalId"`
	Proposer    string `json:"proposer"`
	Description string `json:"description"`
	StartBlock  int64  `json:"startBlock"`
	EndBlock    int64  `json:"endBlock"`
}

// VoteCastPayload 投票事件参数
type VoteCastPayload struct {
	Voter      string `json:"voter"`
	ProposalId string `json:"proposalId"`
	Support    int    `json:"support"` // 0=against, 1=for, 2=abstain
	Weight     string `json:"weight"`
	Reason     string `json:"reason"`
}

// LockCreatedPayload veToken 锁仓事件参数
type LockCreatedPayload struct {
	User       string `json:"user"`
	Amount     string `json:"amount"`
	UnlockTime int64  `json:"unlockTime"`
}
