package consumer

import (
	"context"; "database/sql"; "encoding/json"; "time"
	"github.com/segmentio/kafka-go"
	"github.com/reyfi/reyfi-backend/app/bonds/internal/model"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type BondsEventConsumer struct {
	db *sql.DB; chainId int64; redis *redis.Redis
	positionModel *model.BondPositionModel; redemptionModel *model.BondRedemptionModel
}

func NewBondsEventConsumer(db *sql.DB, chainId int64, rds *redis.Redis) *BondsEventConsumer {
	return &BondsEventConsumer{db: db, chainId: chainId, redis: rds,
		positionModel: model.NewBondPositionModel(db), redemptionModel: model.NewBondRedemptionModel(db)}
}

func (c *BondsEventConsumer) Start(ctx context.Context, brokers []string, groupId, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers, GroupID: groupId, Topic: topic,
		MinBytes: 1, MaxBytes: 10e6, CommitInterval: time.Second, StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	logx.Infof("bonds consumer started: topic=%s", topic)
	for {
		select { case <-ctx.Done(): return nil; default: }
		msg, err := reader.ReadMessage(ctx)
		if err != nil { logx.Errorf("read: %v", err); continue }
		if err := c.handle(ctx, msg.Value); err != nil { logx.Errorf("handle: %v", err) }
	}
}

func (c *BondsEventConsumer) handle(ctx context.Context, value []byte) error {
	event, err := chains.UnmarshalChainEvent(value)
	if err != nil { return err }
	switch event.EventName {
	case chains.EventBondPurchased:
		return c.handlePurchased(ctx, event)
	case chains.EventBondRedeemed:
		return c.handleRedeemed(ctx, event)
	case chains.EventBondCreated:
		logx.Infof("bond market created: block=%d", event.BlockNumber)
		return nil
	default:
		return nil
	}
}

func (c *BondsEventConsumer) handlePurchased(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.BondPurchasedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil { return err }

	pos := &model.BondPosition{
		ChainId: event.ChainId, BondId: payload.BondId, UserAddress: payload.Buyer,
		MarketAddress: event.Contract, PaymentAmount: payload.PaymentAmount,
		PayoutAmount: payload.PayoutAmount, VestedAmount: "0", ClaimableAmount: "0",
		VestingStart: event.GetBlockTime(),
		VestingEnd: event.GetBlockTime().Add(30 * 24 * time.Hour), // 默认30天
		Status: "vesting",
	}
	if err := c.positionModel.Upsert(ctx, pos); err != nil { return err }

	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'bonds', 'purchase', ?, ?, ?)`,
		payload.Buyer, "购买债券 "+payload.PayoutAmount, event.TxHash, event.GetBlockTime())

	logx.Infof("bond purchased: buyer=%s, payout=%s", payload.Buyer, payload.PayoutAmount)
	return nil
}

func (c *BondsEventConsumer) handleRedeemed(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.BondRedeemedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil { return err }

	r := &model.BondRedemption{
		ChainId: event.ChainId, BondId: payload.BondId, UserAddress: payload.User,
		Amount: payload.Amount, BlockNumber: event.BlockNumber,
		BlockTime: event.GetBlockTime(), TxHash: event.TxHash, LogIndex: event.LogIndex,
	}
	if err := c.redemptionModel.Insert(ctx, r); err != nil { return err }

	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'bonds', 'redeem', ?, ?, ?)`,
		payload.User, "赎回债券 "+payload.Amount, event.TxHash, event.GetBlockTime())

	logx.Infof("bond redeemed: user=%s, amount=%s", payload.User, payload.Amount)
	return nil
}
