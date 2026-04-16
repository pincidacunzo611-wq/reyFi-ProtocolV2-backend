// Package indexer 实现链上事件日志解析器。
package indexer

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/svc"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
)

// EventParser 链上事件日志解析器
type EventParser struct {
	svcCtx          *svc.ServiceContext
	eventSignatures map[common.Hash]eventMeta
	contractModules map[common.Address]string
}

type eventMeta struct {
	Module    string
	EventName string
}

// NewEventParser 创建事件解析器
func NewEventParser(svcCtx *svc.ServiceContext) *EventParser {
	p := &EventParser{
		svcCtx:          svcCtx,
		eventSignatures: make(map[common.Hash]eventMeta),
		contractModules: make(map[common.Address]string),
	}
	p.registerEvents()
	p.registerContracts()
	return p
}

func (p *EventParser) registerEvents() {
	// DEX
	p.addEvent("PairCreated(address,address,address,uint256)", "dex", chains.EventPairCreated)
	p.addEvent("Mint(address,uint256,uint256)", "dex", chains.EventMint)
	p.addEvent("Burn(address,uint256,uint256,address)", "dex", chains.EventBurn)
	p.addEvent("Swap(address,uint256,uint256,uint256,uint256,address)", "dex", chains.EventSwap)
	p.addEvent("Sync(uint112,uint112)", "dex", chains.EventSync)

	// Lending
	p.addEvent("Deposit(address,address,uint256,uint16)", "lending", chains.EventDeposit)
	p.addEvent("Withdraw(address,address,address,uint256)", "lending", chains.EventWithdraw)
	p.addEvent("Borrow(address,address,uint256,uint256,uint16)", "lending", chains.EventBorrow)
	p.addEvent("Repay(address,address,address,uint256)", "lending", chains.EventRepay)
	p.addEvent("LiquidationCall(address,address,address,uint256,uint256,address,bool)", "lending", chains.EventLiquidate)

	// Futures
	p.addEvent("PositionOpened(address,uint256,bool,uint256,uint256,uint256)", "futures", chains.EventPositionOpened)
	p.addEvent("PositionClosed(address,uint256,uint256,int256)", "futures", chains.EventPositionClosed)
	p.addEvent("FundingSettled(uint256,int256,int256)", "futures", chains.EventFundingSettled)
	p.addEvent("PositionLiquidated(address,uint256,address,uint256)", "futures", chains.EventLiquidated)

	// Options
	p.addEvent("OptionPurchased(address,uint256,bool,uint256,uint256,uint256,uint256)", "options", chains.EventOptionPurchased)
	p.addEvent("OptionExercised(address,uint256,uint256)", "options", chains.EventOptionExercised)
	p.addEvent("OptionExpired(uint256)", "options", chains.EventOptionExpired)
	p.addEvent("SettlementExecuted(uint256,uint256)", "options", chains.EventSettlementExecuted)

	// Vault
	p.addEvent("VaultCreated(address,address,string)", "vault", chains.EventVaultCreated)
	p.addEvent("StrategyUpdated(address,uint256)", "vault", chains.EventStrategyUpdated)
	p.addEvent("Harvest(address,uint256)", "vault", chains.EventHarvest)

	// Bonds
	p.addEvent("BondCreated(uint256,address,address,uint256)", "bonds", chains.EventBondCreated)
	p.addEvent("BondPurchased(uint256,address,uint256,uint256)", "bonds", chains.EventBondPurchased)
	p.addEvent("BondRedeemed(uint256,address,uint256)", "bonds", chains.EventBondRedeemed)

	// Governance
	p.addEvent("ProposalCreated(uint256,address,address[],uint256[],string[],bytes[],uint256,uint256,string)", "governance", chains.EventProposalCreated)
	p.addEvent("VoteCast(address,uint256,uint8,uint256,string)", "governance", chains.EventVoteCast)
	p.addEvent("ProposalExecuted(uint256)", "governance", chains.EventProposalExecuted)
	p.addEvent("LockCreated(address,uint256,uint256)", "governance", chains.EventLockCreated)

	// User
	p.addEvent("Transfer(address,address,uint256)", "user", "Transfer")

	logx.Infof("registered %d event signatures", len(p.eventSignatures))
}

func (p *EventParser) registerContracts() {
	mapping := map[string]string{
		"Factory":        "dex",
		"Router":         "dex",
		"LendingPool":    "lending",
		"PerpetualMarket": "futures",
		"OptionsMarket":  "options",
		"VaultFactory":   "vault",
		"BondFactory":    "bonds",
		"Governor":       "governance",
	}
	for name, module := range mapping {
		if addr, ok := p.svcCtx.Chain.GetContract(name); ok {
			p.contractModules[addr] = module
		}
	}
}

func (p *EventParser) addEvent(sig, module, eventName string) {
	hash := crypto.Keccak256Hash([]byte(sig))
	p.eventSignatures[hash] = eventMeta{Module: module, EventName: eventName}
}

// ParseLog 解析原始日志为 ChainEvent
func (p *EventParser) ParseLog(log *types.Log) (*chains.ChainEvent, error) {
	if len(log.Topics) == 0 {
		return nil, nil
	}

	topic0 := log.Topics[0]
	meta, ok := p.eventSignatures[topic0]
	if !ok {
		return nil, nil
	}

	if contractModule, exists := p.contractModules[log.Address]; exists {
		meta.Module = contractModule
	}

	payload, err := p.parsePayload(meta.EventName, log)
	if err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &chains.ChainEvent{
		ChainId:     p.svcCtx.Config.Chain.ChainId,
		Module:      meta.Module,
		EventName:   meta.EventName,
		Contract:    log.Address.Hex(),
		TxHash:      log.TxHash.Hex(),
		TxIndex:     int(log.TxIndex),
		LogIndex:    int(log.Index),
		BlockNumber: int64(log.BlockNumber),
		BlockHash:   log.BlockHash.Hex(),
		Status:      chains.EventStatusPending,
		Topic0:      topic0.Hex(),
		Payload:     payloadBytes,
	}, nil
}

func (p *EventParser) parsePayload(eventName string, log *types.Log) (interface{}, error) {
	switch eventName {
	case chains.EventSwap:
		return parseSwap(log), nil
	case chains.EventSync:
		return parseSync(log), nil
	case chains.EventMint:
		return parseMint(log), nil
	case chains.EventBurn:
		return parseBurn(log), nil
	case chains.EventPairCreated:
		return parsePairCreated(log), nil
	default:
		return parseGeneric(log), nil
	}
}

func parseSwap(log *types.Log) *chains.SwapPayload {
	p := &chains.SwapPayload{}
	if len(log.Topics) >= 3 {
		p.Sender = common.BytesToAddress(log.Topics[1].Bytes()).Hex()
		p.To = common.BytesToAddress(log.Topics[2].Bytes()).Hex()
	}
	if len(log.Data) >= 128 {
		p.Amount0In = bytesToDecimal(log.Data[0:32])
		p.Amount1In = bytesToDecimal(log.Data[32:64])
		p.Amount0Out = bytesToDecimal(log.Data[64:96])
		p.Amount1Out = bytesToDecimal(log.Data[96:128])
	}
	return p
}

func parseSync(log *types.Log) *chains.SyncPayload {
	p := &chains.SyncPayload{}
	if len(log.Data) >= 64 {
		p.Reserve0 = bytesToDecimal(log.Data[0:32])
		p.Reserve1 = bytesToDecimal(log.Data[32:64])
	}
	return p
}

func parseMint(log *types.Log) *chains.MintPayload {
	p := &chains.MintPayload{}
	if len(log.Topics) >= 2 {
		p.Sender = common.BytesToAddress(log.Topics[1].Bytes()).Hex()
	}
	if len(log.Data) >= 64 {
		p.Amount0 = bytesToDecimal(log.Data[0:32])
		p.Amount1 = bytesToDecimal(log.Data[32:64])
	}
	return p
}

func parseBurn(log *types.Log) *chains.BurnPayload {
	p := &chains.BurnPayload{}
	if len(log.Topics) >= 2 {
		p.Sender = common.BytesToAddress(log.Topics[1].Bytes()).Hex()
	}
	if len(log.Data) >= 96 {
		p.Amount0 = bytesToDecimal(log.Data[0:32])
		p.Amount1 = bytesToDecimal(log.Data[32:64])
		p.To = common.BytesToAddress(log.Data[76:96]).Hex()
	}
	return p
}

func parsePairCreated(log *types.Log) *chains.PairCreatedPayload {
	p := &chains.PairCreatedPayload{}
	if len(log.Topics) >= 3 {
		p.Token0 = common.BytesToAddress(log.Topics[1].Bytes()).Hex()
		p.Token1 = common.BytesToAddress(log.Topics[2].Bytes()).Hex()
	}
	if len(log.Data) >= 64 {
		p.PairAddress = common.BytesToAddress(log.Data[12:32]).Hex()
		p.PairIndex = bytesToDecimal(log.Data[32:64])
	}
	return p
}

func parseGeneric(log *types.Log) map[string]interface{} {
	result := map[string]interface{}{
		"data": common.Bytes2Hex(log.Data),
	}
	topics := make([]string, len(log.Topics))
	for i, t := range log.Topics {
		topics[i] = t.Hex()
	}
	result["topics"] = topics
	return result
}

// bytesToDecimal 将 ABI 编码的 32 字节转为十进制字符串
func bytesToDecimal(data []byte) string {
	bi := new(big.Int).SetBytes(data)
	return bi.String()
}
