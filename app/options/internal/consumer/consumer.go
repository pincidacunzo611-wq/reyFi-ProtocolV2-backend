package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/reyfi/reyfi-backend/app/options/internal/model"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type OptionsEventConsumer struct {
	db            *sql.DB
	chainId       int64
	redis         *redis.Redis
	positionModel *model.OptionsPositionModel
	settlementModel *model.OptionsSettlementModel
}

func NewOptionsEventConsumer(db *sql.DB, chainId int64, rds *redis.Redis) *OptionsEventConsumer {
	return &OptionsEventConsumer{
		db: db, chainId: chainId, redis: rds,
		positionModel:   model.NewOptionsPositionModel(db),
		settlementModel: model.NewOptionsSettlementModel(db),
	}
}

func (c *OptionsEventConsumer) Start(ctx context.Context, brokers []string, groupId, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers, GroupID: groupId, Topic: topic,
		MinBytes: 1, MaxBytes: 10e6, CommitInterval: time.Second, StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	logx.Infof("options consumer started: topic=%s", topic)

	for {
		select { case <-ctx.Done(): return nil; default: }
		msg, err := reader.ReadMessage(ctx)
		if err != nil { logx.Errorf("read: %v", err); continue }
		if err := c.handle(ctx, msg.Value); err != nil { logx.Errorf("handle: %v", err) }
	}
}

func (c *OptionsEventConsumer) handle(ctx context.Context, value []byte) error {
	event, err := chains.UnmarshalChainEvent(value)
	if err != nil { return err }

	switch event.EventName {
	case chains.EventOptionPurchased:
		return c.handlePurchased(ctx, event)
	case chains.EventOptionExercised:
		return c.handleExercised(ctx, event)
	case chains.EventOptionExpired:
		return c.handleExpired(ctx, event)
	case chains.EventSettlementExecuted:
		return c.handleSettlement(ctx, event)
	default:
		return nil
	}
}

func (c *OptionsEventConsumer) handlePurchased(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.OptionPurchasedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil { return err }

	expiryTime := time.Unix(payload.Expiry, 0).UTC()
	pos := &model.OptionsPosition{
		ChainId: event.ChainId, OptionId: payload.OptionId,
		UserAddress: payload.Buyer, MarketAddress: event.Contract,
		UnderlyingAddress: "", StrikePrice: payload.StrikePrice,
		Premium: payload.Premium, Size: payload.Size,
		OptionType: payload.OptionType, ExpiryTime: expiryTime, Status: "open",
	}
	if err := c.positionModel.Upsert(ctx, pos); err != nil { return err }

	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'options', 'purchase', ?, ?, ?)`,
		payload.Buyer, "购买 "+payload.OptionType+" 期权 strike="+payload.StrikePrice,
		event.TxHash, event.GetBlockTime())

	logx.Infof("option purchased: buyer=%s, type=%s, strike=%s", payload.Buyer, payload.OptionType, payload.StrikePrice)
	return nil
}

func (c *OptionsEventConsumer) handleExercised(ctx context.Context, event *chains.ChainEvent) error {
	var payload map[string]interface{}
	json.Unmarshal(event.Payload, &payload)
	logx.Infof("option exercised: block=%d", event.BlockNumber)
	return nil
}

func (c *OptionsEventConsumer) handleExpired(ctx context.Context, event *chains.ChainEvent) error {
	logx.Infof("option expired: block=%d", event.BlockNumber)
	return nil
}

func (c *OptionsEventConsumer) handleSettlement(ctx context.Context, event *chains.ChainEvent) error {
	logx.Infof("option settlement executed: block=%d", event.BlockNumber)
	return nil
}
