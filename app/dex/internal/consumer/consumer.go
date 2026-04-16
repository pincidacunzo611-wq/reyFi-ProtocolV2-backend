// Package consumer DEX 模块的 Kafka 事件消费者
package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/reyfi/reyfi-backend/app/dex/internal/model"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// DexEventConsumer DEX 事件消费者
type DexEventConsumer struct {
	db        *sql.DB
	chainId   int64
	redis     *redis.Redis
	pairModel  *model.DexPairModel
	tradeModel *model.DexTradeModel
	snapModel  *model.DexPairSnapshotModel
}

// NewDexEventConsumer 创建 DEX 事件消费者
func NewDexEventConsumer(db *sql.DB, chainId int64, rds *redis.Redis) *DexEventConsumer {
	return &DexEventConsumer{
		db:         db,
		chainId:    chainId,
		redis:      rds,
		pairModel:  model.NewDexPairModel(db),
		tradeModel: model.NewDexTradeModel(db),
		snapModel:  model.NewDexPairSnapshotModel(db),
	}
}

// Start 启动 Kafka 消费者
func (c *DexEventConsumer) Start(ctx context.Context, brokers []string, groupId, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupId,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})
	defer reader.Close()

	logx.Infof("dex consumer started: topic=%s, group=%s", topic, groupId)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			logx.Errorf("read kafka message: %v", err)
			continue
		}

		if err := c.handleMessage(ctx, msg.Value); err != nil {
			logx.Errorf("handle message: %v", err)
		}
	}
}

// handleMessage 处理单条 Kafka 消息
func (c *DexEventConsumer) handleMessage(ctx context.Context, value []byte) error {
	event, err := chains.UnmarshalChainEvent(value)
	if err != nil {
		return err
	}

	logx.Debugf("dex consuming event: %s from block %d", event.EventName, event.BlockNumber)

	switch event.EventName {
	case chains.EventSwap:
		return c.handleSwap(ctx, event)
	case chains.EventSync:
		return c.handleSync(ctx, event)
	case chains.EventMint:
		return c.handleMint(ctx, event)
	case chains.EventBurn:
		return c.handleBurn(ctx, event)
	case chains.EventPairCreated:
		return c.handlePairCreated(ctx, event)
	default:
		logx.Debugf("unhandled dex event: %s", event.EventName)
		return nil
	}
}

// handleSwap 处理 Swap 事件
func (c *DexEventConsumer) handleSwap(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.SwapPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	// 幂等检查
	exists, err := c.tradeModel.ExistsByTxHashAndLogIndex(ctx, event.ChainId, event.TxHash, event.LogIndex)
	if err != nil {
		return err
	}
	if exists {
		logx.Infof("swap event already processed: %s:%d", event.TxHash, event.LogIndex)
		return nil
	}

	// 判断交易方向（相对于 token0）
	direction := "buy"
	if payload.Amount0In != "0" && payload.Amount0In != "" {
		direction = "sell"
	}

	// 从最新快照计算价格
	price := "0"
	amountUsd := "0"
	snap, snapErr := c.snapModel.GetLatest(ctx, event.ChainId, event.Contract)
	if snapErr == nil && snap != nil && snap.Reserve0 != "0" && snap.Reserve1 != "0" {
		r0, _ := new(big.Float).SetString(snap.Reserve0)
		r1, _ := new(big.Float).SetString(snap.Reserve1)
		if r0 != nil && r1 != nil && r0.Sign() > 0 {
			p := new(big.Float).Quo(r1, r0)
			price = p.Text('f', 18)
		}
	}

	trade := &model.DexTrade{
		ChainId:       event.ChainId,
		PairAddress:   event.Contract,
		TraderAddress: payload.To,
		SenderAddress: payload.Sender,
		Direction:     direction,
		Amount0In:     payload.Amount0In,
		Amount1In:     payload.Amount1In,
		Amount0Out:    payload.Amount0Out,
		Amount1Out:    payload.Amount1Out,
		AmountUsd:     amountUsd,
		Price:         price,
		BlockNumber:   event.BlockNumber,
		BlockTime:     event.GetBlockTime(),
		TxHash:        event.TxHash,
		LogIndex:      event.LogIndex,
	}

	if err := c.tradeModel.Insert(ctx, trade); err != nil {
		return err
	}

	// 清除缓存
	c.invalidateCache(event.Contract)

	logx.Infof("swap processed: pair=%s, dir=%s, tx=%s", event.Contract, direction, event.TxHash)
	return nil
}

// handleSync 处理 Sync 事件（储备量更新）
func (c *DexEventConsumer) handleSync(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.SyncPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	// 从 reserve 计算价格
	price0 := "0"
	price1 := "0"
	r0, _ := new(big.Float).SetString(payload.Reserve0)
	r1, _ := new(big.Float).SetString(payload.Reserve1)
	if r0 != nil && r1 != nil {
		if r0.Sign() > 0 {
			p := new(big.Float).Quo(r1, r0)
			price0 = p.Text('f', 18)
		}
		if r1.Sign() > 0 {
			p := new(big.Float).Quo(r0, r1)
			price1 = p.Text('f', 18)
		}
	}

	// 更新交易对快照
	snap := &model.DexPairSnapshot{
		ChainId:      event.ChainId,
		PairAddress:  event.Contract,
		Reserve0:     payload.Reserve0,
		Reserve1:     payload.Reserve1,
		Price0:       price0,
		Price1:       price1,
		TotalSupply:  "0",
		TvlUsd:       "0",
		SnapshotTime: event.GetBlockTime(),
	}

	if err := c.snapModel.Insert(ctx, snap); err != nil {
		return err
	}

	c.invalidateCache(event.Contract)

	logx.Debugf("sync processed: pair=%s, r0=%s, r1=%s", event.Contract, payload.Reserve0, payload.Reserve1)
	return nil
}

// handleMint 处理 Mint 事件（添加流动性）
func (c *DexEventConsumer) handleMint(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.MintPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	// 记录流动性事件
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO dex_liquidity_events (chain_id, pair_address, user_address, event_type,
		 amount0, amount1, lp_amount, amount_usd, block_number, block_time, tx_hash, log_index)
		 VALUES (?, ?, ?, 'mint', ?, ?, 0, 0, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id=id`,
		event.ChainId, event.Contract, payload.Sender,
		payload.Amount0, payload.Amount1,
		event.BlockNumber, event.GetBlockTime(), event.TxHash, event.LogIndex)

	if err != nil {
		return err
	}

	logx.Infof("mint processed: pair=%s, sender=%s", event.Contract, payload.Sender)
	return nil
}

// handleBurn 处理 Burn 事件（移除流动性）
func (c *DexEventConsumer) handleBurn(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.BurnPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	_, err := c.db.ExecContext(ctx,
		`INSERT INTO dex_liquidity_events (chain_id, pair_address, user_address, event_type,
		 amount0, amount1, lp_amount, amount_usd, block_number, block_time, tx_hash, log_index)
		 VALUES (?, ?, ?, 'burn', ?, ?, 0, 0, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id=id`,
		event.ChainId, event.Contract, payload.Sender,
		payload.Amount0, payload.Amount1,
		event.BlockNumber, event.GetBlockTime(), event.TxHash, event.LogIndex)

	if err != nil {
		return err
	}

	logx.Infof("burn processed: pair=%s, sender=%s", event.Contract, payload.Sender)
	return nil
}

// handlePairCreated 处理 PairCreated 事件
func (c *DexEventConsumer) handlePairCreated(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.PairCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	// 尝试从数据库查找已知的 token 元数据
	token0Symbol := ""
	token1Symbol := ""
	token0Decimals := 18
	token1Decimals := 18

	// 从已有交易对中查找 token 信息
	c.db.QueryRowContext(ctx,
		`SELECT token0_symbol, token0_decimals FROM dex_pairs 
		 WHERE chain_id = ? AND token0_address = ? AND token0_symbol != '' LIMIT 1`,
		event.ChainId, payload.Token0).Scan(&token0Symbol, &token0Decimals)
	c.db.QueryRowContext(ctx,
		`SELECT token1_symbol, token1_decimals FROM dex_pairs 
		 WHERE chain_id = ? AND token1_address = ? AND token1_symbol != '' LIMIT 1`,
		event.ChainId, payload.Token1).Scan(&token1Symbol, &token1Decimals)
	// 也反向匹配
	if token0Symbol == "" {
		c.db.QueryRowContext(ctx,
			`SELECT token0_symbol, token0_decimals FROM dex_pairs 
			 WHERE chain_id = ? AND token1_address = ? AND token0_symbol != '' LIMIT 1`,
			event.ChainId, payload.Token0).Scan(&token0Symbol, &token0Decimals)
	}
	if token1Symbol == "" {
		c.db.QueryRowContext(ctx,
			`SELECT token1_symbol, token1_decimals FROM dex_pairs 
			 WHERE chain_id = ? AND token0_address = ? AND token1_symbol != '' LIMIT 1`,
			event.ChainId, payload.Token1).Scan(&token1Symbol, &token1Decimals)
	}

	pair := &model.DexPair{
		ChainId:        event.ChainId,
		PairAddress:    payload.PairAddress,
		Token0Address:  payload.Token0,
		Token1Address:  payload.Token1,
		Token0Symbol:   token0Symbol,
		Token1Symbol:   token1Symbol,
		Token0Decimals: token0Decimals,
		Token1Decimals: token1Decimals,
		FeeBps:         30,
		CreatedBlock:   event.BlockNumber,
		IsActive:       true,
	}

	if err := c.pairModel.Upsert(ctx, pair); err != nil {
		return err
	}

	logx.Infof("pair created: %s (token0=%s, token1=%s)", payload.PairAddress, payload.Token0, payload.Token1)
	return nil
}

// invalidateCache 清除指定交易对的缓存
func (c *DexEventConsumer) invalidateCache(pairAddress string) {
	keys := []string{
		"dex:pair:" + pairAddress + ":info",
		"dex:pairs:page:1",
		"dashboard:overview",
	}
	for _, key := range keys {
		if _, err := c.redis.Del(key); err != nil {
			logx.Errorf("delete cache key %s: %v", key, err)
		}
	}
}
