package consumer

import (
	"context"; "database/sql"; "encoding/json"; "time"
	"github.com/segmentio/kafka-go"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type VaultEventConsumer struct {
	db *sql.DB; chainId int64; redis *redis.Redis
}

func NewVaultEventConsumer(db *sql.DB, chainId int64, rds *redis.Redis) *VaultEventConsumer {
	return &VaultEventConsumer{db: db, chainId: chainId, redis: rds}
}

func (c *VaultEventConsumer) Start(ctx context.Context, brokers []string, groupId, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers, GroupID: groupId, Topic: topic,
		MinBytes: 1, MaxBytes: 10e6, CommitInterval: time.Second, StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	logx.Infof("vault consumer started: topic=%s", topic)

	for {
		select { case <-ctx.Done(): return nil; default: }
		msg, err := reader.ReadMessage(ctx)
		if err != nil { logx.Errorf("read: %v", err); continue }
		if err := c.handle(ctx, msg.Value); err != nil { logx.Errorf("handle: %v", err) }
	}
}

func (c *VaultEventConsumer) handle(ctx context.Context, value []byte) error {
	event, err := chains.UnmarshalChainEvent(value)
	if err != nil { return err }

	switch event.EventName {
	case chains.EventDeposit:
		return c.handleDeposit(ctx, event)
	case chains.EventWithdraw:
		return c.handleWithdraw(ctx, event)
	case chains.EventHarvest:
		return c.handleHarvest(ctx, event)
	case chains.EventVaultCreated:
		return c.handleVaultCreated(ctx, event)
	default:
		return nil
	}
}

func (c *VaultEventConsumer) handleDeposit(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.DepositPayload
	json.Unmarshal(event.Payload, &payload)
	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'vault', 'deposit', ?, ?, ?)`,
		payload.User, "申购金库 "+payload.Amount, event.TxHash, event.GetBlockTime())
	logx.Infof("vault deposit: user=%s, amount=%s", payload.User, payload.Amount)
	return nil
}

func (c *VaultEventConsumer) handleWithdraw(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.WithdrawPayload
	json.Unmarshal(event.Payload, &payload)
	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'vault', 'withdraw', ?, ?, ?)`,
		payload.User, "赎回金库 "+payload.Amount, event.TxHash, event.GetBlockTime())
	logx.Infof("vault withdraw: user=%s, amount=%s", payload.User, payload.Amount)
	return nil
}

func (c *VaultEventConsumer) handleHarvest(ctx context.Context, event *chains.ChainEvent) error {
	logx.Infof("vault harvest: block=%d, contract=%s", event.BlockNumber, event.Contract)
	return nil
}

func (c *VaultEventConsumer) handleVaultCreated(ctx context.Context, event *chains.ChainEvent) error {
	logx.Infof("vault created: block=%d, contract=%s", event.BlockNumber, event.Contract)
	return nil
}
