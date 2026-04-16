package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/reyfi/reyfi-backend/app/futures/internal/model"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type FuturesEventConsumer struct {
	db            *sql.DB
	chainId       int64
	redis         *redis.Redis
	positionModel *model.FuturesPositionModel
	fundingModel  *model.FuturesFundingModel
	liqModel      *model.FuturesLiquidationModel
}

func NewFuturesEventConsumer(db *sql.DB, chainId int64, rds *redis.Redis) *FuturesEventConsumer {
	return &FuturesEventConsumer{
		db: db, chainId: chainId, redis: rds,
		positionModel: model.NewFuturesPositionModel(db),
		fundingModel:  model.NewFuturesFundingModel(db),
		liqModel:      model.NewFuturesLiquidationModel(db),
	}
}

func (c *FuturesEventConsumer) Start(ctx context.Context, brokers []string, groupId, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers, GroupID: groupId, Topic: topic,
		MinBytes: 1, MaxBytes: 10e6, CommitInterval: time.Second, StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	logx.Infof("futures consumer started: topic=%s", topic)

	for {
		select {
		case <-ctx.Done(): return nil
		default:
		}
		msg, err := reader.ReadMessage(ctx)
		if err != nil { logx.Errorf("read: %v", err); continue }
		if err := c.handle(ctx, msg.Value); err != nil { logx.Errorf("handle: %v", err) }
	}
}

func (c *FuturesEventConsumer) handle(ctx context.Context, value []byte) error {
	event, err := chains.UnmarshalChainEvent(value)
	if err != nil { return err }

	switch event.EventName {
	case chains.EventPositionOpened:
		return c.handlePositionOpened(ctx, event)
	case chains.EventPositionClosed:
		return c.handlePositionClosed(ctx, event)
	case chains.EventFundingSettled:
		return c.handleFundingSettled(ctx, event)
	case chains.EventLiquidated:
		return c.handleLiquidated(ctx, event)
	default:
		return nil
	}
}

func (c *FuturesEventConsumer) handlePositionOpened(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.PositionOpenedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil { return err }

	pos := &model.FuturesPosition{
		ChainId: event.ChainId, PositionId: payload.PositionId,
		UserAddress: payload.User, MarketAddress: payload.Market,
		Side: payload.Side, Size: payload.Size, EntryPrice: payload.EntryPrice,
		MarkPrice: payload.EntryPrice, Margin: payload.Margin, Leverage: payload.Leverage,
		UnrealizedPnl: "0", RealizedPnl: "0", LiquidationPrice: "0",
		Status: "open", OpenedAt: event.GetBlockTime(),
	}
	if err := c.positionModel.Upsert(ctx, pos); err != nil { return err }

	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'futures', 'open', ?, ?, ?)`,
		payload.User, "开仓 "+payload.Side+" "+payload.Size, event.TxHash, event.GetBlockTime())

	logx.Infof("futures position opened: user=%s, side=%s, size=%s", payload.User, payload.Side, payload.Size)
	return nil
}

func (c *FuturesEventConsumer) handlePositionClosed(ctx context.Context, event *chains.ChainEvent) error {
	// 通用解析
	var payload map[string]interface{}
	json.Unmarshal(event.Payload, &payload)
	logx.Infof("futures position closed: block=%d", event.BlockNumber)
	return nil
}

func (c *FuturesEventConsumer) handleFundingSettled(ctx context.Context, event *chains.ChainEvent) error {
	var payload map[string]interface{}
	json.Unmarshal(event.Payload, &payload)
	logx.Infof("futures funding settled: block=%d", event.BlockNumber)
	return nil
}

func (c *FuturesEventConsumer) handleLiquidated(ctx context.Context, event *chains.ChainEvent) error {
	var payload map[string]interface{}
	json.Unmarshal(event.Payload, &payload)
	logx.Errorf("futures liquidation: block=%d", event.BlockNumber)
	return nil
}
