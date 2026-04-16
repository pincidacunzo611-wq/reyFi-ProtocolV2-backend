package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/reyfi/reyfi-backend/app/lending/internal/model"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type LendingEventConsumer struct {
	db                *sql.DB
	chainId           int64
	redis             *redis.Redis
	userPositionModel *model.LendingUserPositionModel
	liquidationModel  *model.LendingLiquidationModel
	snapshotModel     *model.LendingMarketSnapshotModel
}

func NewLendingEventConsumer(db *sql.DB, chainId int64, rds *redis.Redis) *LendingEventConsumer {
	return &LendingEventConsumer{
		db:                db,
		chainId:           chainId,
		redis:             rds,
		userPositionModel: model.NewLendingUserPositionModel(db),
		liquidationModel:  model.NewLendingLiquidationModel(db),
		snapshotModel:     model.NewLendingMarketSnapshotModel(db),
	}
}

func (c *LendingEventConsumer) Start(ctx context.Context, brokers []string, groupId, topic string) error {
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

	logx.Infof("lending consumer started: topic=%s, group=%s", topic, groupId)

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
			logx.Errorf("handle lending message: %v", err)
		}
	}
}

func (c *LendingEventConsumer) handleMessage(ctx context.Context, value []byte) error {
	event, err := chains.UnmarshalChainEvent(value)
	if err != nil {
		return err
	}

	logx.Debugf("lending consuming event: %s from block %d", event.EventName, event.BlockNumber)

	switch event.EventName {
	case chains.EventDeposit:
		return c.handleDeposit(ctx, event)
	case chains.EventWithdraw:
		return c.handleWithdraw(ctx, event)
	case chains.EventBorrow:
		return c.handleBorrow(ctx, event)
	case chains.EventRepay:
		return c.handleRepay(ctx, event)
	case chains.EventLiquidate:
		return c.handleLiquidation(ctx, event)
	default:
		logx.Debugf("unhandled lending event: %s", event.EventName)
		return nil
	}
}

func (c *LendingEventConsumer) handleDeposit(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.DepositPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	// 更新用户存款头寸
	pos := &model.LendingUserPosition{
		ChainId:        event.ChainId,
		UserAddress:    payload.User,
		AssetAddress:   payload.Asset,
		SuppliedAmount: payload.Amount,
		HealthFactor:   "999999.99",
	}
	if err := c.userPositionModel.Upsert(ctx, pos); err != nil {
		return err
	}

	// 记录活动流
	c.recordActivity(ctx, payload.User, "deposit",
		"存入 "+payload.Amount+" 到借贷池", event.TxHash, event.GetBlockTime())

	logx.Infof("lending deposit: user=%s, asset=%s, amount=%s", payload.User, payload.Asset, payload.Amount)
	return nil
}

func (c *LendingEventConsumer) handleWithdraw(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.WithdrawPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	logx.Infof("lending withdraw: user=%s, asset=%s, amount=%s", payload.User, payload.Asset, payload.Amount)
	return nil
}

func (c *LendingEventConsumer) handleBorrow(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.BorrowPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	pos := &model.LendingUserPosition{
		ChainId:        event.ChainId,
		UserAddress:    payload.User,
		AssetAddress:   payload.Asset,
		BorrowedAmount: payload.Amount,
	}
	if err := c.userPositionModel.Upsert(ctx, pos); err != nil {
		return err
	}

	c.recordActivity(ctx, payload.User, "borrow",
		"借入 "+payload.Amount, event.TxHash, event.GetBlockTime())

	logx.Infof("lending borrow: user=%s, asset=%s, amount=%s", payload.User, payload.Asset, payload.Amount)
	return nil
}

func (c *LendingEventConsumer) handleRepay(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.RepayPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	c.recordActivity(ctx, payload.User, "repay",
		"偿还 "+payload.Amount, event.TxHash, event.GetBlockTime())

	logx.Infof("lending repay: user=%s, asset=%s, amount=%s", payload.User, payload.Asset, payload.Amount)
	return nil
}

func (c *LendingEventConsumer) handleLiquidation(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.LiquidatePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	liq := &model.LendingLiquidation{
		ChainId:            event.ChainId,
		BorrowerAddress:    payload.Borrower,
		LiquidatorAddress:  payload.Liquidator,
		CollateralAsset:    payload.CollateralAsset,
		DebtAsset:          payload.DebtAsset,
		CollateralAmount:   payload.CollateralAmount,
		DebtAmount:         payload.DebtAmount,
		PenaltyAmount:      "0",
		HealthFactorBefore: "0",
		BlockNumber:        event.BlockNumber,
		BlockTime:          event.GetBlockTime(),
		TxHash:             event.TxHash,
		LogIndex:           event.LogIndex,
	}

	if err := c.liquidationModel.Insert(ctx, liq); err != nil {
		return err
	}

	c.recordActivity(ctx, payload.Borrower, "liquidated",
		"被清算: 抵押 "+payload.CollateralAmount+" 偿还 "+payload.DebtAmount,
		event.TxHash, event.GetBlockTime())

	logx.Errorf("lending liquidation: borrower=%s, liquidator=%s", payload.Borrower, payload.Liquidator)
	return nil
}

func (c *LendingEventConsumer) recordActivity(ctx context.Context, userAddr, action, summary, txHash string, blockTime time.Time) {
	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'lending', ?, ?, ?, ?)`,
		userAddr, action, summary, txHash, blockTime)
}
