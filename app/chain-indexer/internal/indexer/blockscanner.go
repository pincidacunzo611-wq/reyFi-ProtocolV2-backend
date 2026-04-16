// Package indexer 实现区块扫描器核心逻辑。
// 负责追踪最新区块、获取事件日志、分发到 Kafka。
package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/svc"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
)

// BlockScanner 区块扫描器
type BlockScanner struct {
	svcCtx       *svc.ServiceContext
	parser       *EventParser
	reorgDetector *ReorgDetector
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewBlockScanner 创建区块扫描器
func NewBlockScanner(svcCtx *svc.ServiceContext) *BlockScanner {
	return &BlockScanner{
		svcCtx:        svcCtx,
		parser:        NewEventParser(svcCtx),
		reorgDetector: NewReorgDetector(svcCtx),
		stopCh:        make(chan struct{}),
	}
}

// Start 启动区块扫描
func (s *BlockScanner) Start() error {
	s.wg.Add(1)
	defer s.wg.Done()

	cfg := s.svcCtx.Config.Scanner
	pollInterval := time.Duration(cfg.PollInterval) * time.Millisecond

	// 获取上次同步位置
	lastBlock, err := s.getLastSyncedBlock()
	if err != nil {
		return fmt.Errorf("get last synced block: %w", err)
	}

	// 如果配置了起始区块且更大，使用配置值
	if cfg.StartBlock > lastBlock {
		lastBlock = cfg.StartBlock
	}

	logx.Infof("starting block scanner from block %d, batch=%d, poll=%dms, confirms=%d",
		lastBlock, cfg.BatchSize, cfg.PollInterval, cfg.ConfirmBlocks)

	for {
		select {
		case <-s.stopCh:
			logx.Info("block scanner received stop signal")
			return nil
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// 获取链上最新区块
		latestBlock, err := s.svcCtx.Chain.LatestBlockNumber(ctx)
		if err != nil {
			logx.Errorf("get latest block: %v", err)
			cancel()
			time.Sleep(pollInterval)
			continue
		}

		// 检查是否有新区块
		if lastBlock >= latestBlock {
			cancel()
			time.Sleep(pollInterval)
			continue
		}

		// 计算本批次扫描范围
		toBlock := lastBlock + cfg.BatchSize
		if toBlock > latestBlock {
			toBlock = latestBlock
		}

		logx.Infof("scanning blocks %d -> %d (chain height: %d, lag: %d)",
			lastBlock+1, toBlock, latestBlock, latestBlock-toBlock)

		// 检测链重组
		if lastBlock > 0 {
			reorged, forkBlock, err := s.reorgDetector.Detect(ctx, lastBlock)
			if err != nil {
				logx.Errorf("reorg detection error: %v", err)
			} else if reorged {
				logx.Errorf("chain reorg detected at block %d, rolling back", forkBlock)
				if err := s.reorgDetector.Rollback(ctx, forkBlock); err != nil {
					logx.Errorf("rollback error: %v", err)
				}
				lastBlock = forkBlock - 1
				cancel()
				continue
			}
		}

		// 扫描区块并获取事件
		if err := s.scanBlockRange(ctx, lastBlock+1, toBlock); err != nil {
			logx.Errorf("scan blocks %d-%d error: %v", lastBlock+1, toBlock, err)
			cancel()
			s.retryWithBackoff(cfg.MaxRetries, cfg.RetryInterval)
			continue
		}

		// 确认旧事件
		confirmBlock := toBlock - cfg.ConfirmBlocks
		if confirmBlock > 0 {
			if err := s.confirmEvents(ctx, confirmBlock); err != nil {
				logx.Errorf("confirm events error: %v", err)
			}
		}

		// 更新同步游标
		if err := s.updateSyncCursor(ctx, toBlock); err != nil {
			logx.Errorf("update sync cursor: %v", err)
		}

		lastBlock = toBlock
		cancel()
	}
}

// Stop 停止扫描器
func (s *BlockScanner) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// scanBlockRange 扫描指定区块范围内的事件
func (s *BlockScanner) scanBlockRange(ctx context.Context, from, to int64) error {
	// 获取所有需要监听的合约地址
	addresses := s.svcCtx.Chain.GetAllContractAddresses()
	if len(addresses) == 0 {
		return nil
	}

	// 查询事件日志
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(from),
		ToBlock:   big.NewInt(to),
		Addresses: addresses,
	}

	logs, err := s.svcCtx.Chain.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("filter logs [%d, %d]: %w", from, to, err)
	}

	logx.Infof("found %d logs in blocks %d-%d", len(logs), from, to)

	// 记录区块信息
	if err := s.saveBlockHeaders(ctx, from, to); err != nil {
		return fmt.Errorf("save block headers: %w", err)
	}

	// 解析并发布事件
	for i := range logs {
		event, err := s.parser.ParseLog(&logs[i])
		if err != nil {
			logx.Errorf("parse log (tx=%s, idx=%d): %v", logs[i].TxHash.Hex(), logs[i].Index, err)
			continue
		}
		if event == nil {
			continue // 未识别的事件，跳过
		}

		// 保存原始事件到数据库
		if err := s.saveRawEvent(ctx, event); err != nil {
			logx.Errorf("save raw event: %v", err)
			continue
		}

		// 发布到 Kafka
		topic, ok := chains.ModuleTopicMap[event.Module]
		if !ok {
			logx.Errorf("unknown module: %s", event.Module)
			continue
		}

		data, err := event.Marshal()
		if err != nil {
			logx.Errorf("marshal event: %v", err)
			continue
		}

		if err := s.svcCtx.Publisher.Publish(ctx, topic, event.UniqueKey(), data); err != nil {
			logx.Errorf("publish event to %s: %v", topic, err)
			continue
		}

		logx.Debugf("published event: module=%s, event=%s, block=%d, tx=%s",
			event.Module, event.EventName, event.BlockNumber, event.TxHash)
	}

	return nil
}

// saveBlockHeaders 保存区块头到数据库
func (s *BlockScanner) saveBlockHeaders(ctx context.Context, from, to int64) error {
	chainId := s.svcCtx.Config.Chain.ChainId

	for blockNum := from; blockNum <= to; blockNum++ {
		header, err := s.svcCtx.Chain.HeaderByNumber(ctx, blockNum)
		if err != nil {
			return fmt.Errorf("get header %d: %w", blockNum, err)
		}

		blockTime := time.Unix(int64(header.Time), 0).UTC()

		_, err = s.svcCtx.DB.ExecContext(ctx,
			`INSERT INTO chain_blocks (chain_id, block_number, block_hash, parent_hash, block_time, tx_count)
			 VALUES (?, ?, ?, ?, ?, 0)
			 ON DUPLICATE KEY UPDATE block_hash=VALUES(block_hash), parent_hash=VALUES(parent_hash)`,
			chainId, blockNum, header.Hash().Hex(), header.ParentHash.Hex(), blockTime,
		)
		if err != nil {
			return fmt.Errorf("insert block %d: %w", blockNum, err)
		}
	}
	return nil
}

// saveRawEvent 保存原始事件到 chain_raw_events 表
func (s *BlockScanner) saveRawEvent(ctx context.Context, event *chains.ChainEvent) error {
	_, err := s.svcCtx.DB.ExecContext(ctx,
		`INSERT INTO chain_raw_events 
		 (chain_id, module, contract_address, event_name, block_number, block_time, 
		  tx_hash, tx_index, log_index, topic0, payload_json, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id=id`,
		event.ChainId, event.Module, event.Contract, event.EventName,
		event.BlockNumber, event.GetBlockTime(),
		event.TxHash, event.TxIndex, event.LogIndex,
		event.Topic0, string(event.Payload), event.Status,
	)
	return err
}

// confirmEvents 确认已达到确认块数的事件
func (s *BlockScanner) confirmEvents(ctx context.Context, confirmBlock int64) error {
	chainId := s.svcCtx.Config.Chain.ChainId
	_, err := s.svcCtx.DB.ExecContext(ctx,
		`UPDATE chain_raw_events 
		 SET status = 'confirmed'
		 WHERE chain_id = ? AND block_number <= ? AND status = 'pending'`,
		chainId, confirmBlock,
	)
	return err
}

// getLastSyncedBlock 获取上次同步到的区块号
func (s *BlockScanner) getLastSyncedBlock() (int64, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	var lastBlock sql.NullInt64

	err := s.svcCtx.DB.QueryRow(
		`SELECT last_scanned_block FROM chain_sync_cursors 
		 WHERE module = 'indexer' AND chain_id = ? LIMIT 1`,
		chainId,
	).Scan(&lastBlock)

	if err == sql.ErrNoRows {
		// 首次运行，初始化游标
		_, err := s.svcCtx.DB.Exec(
			`INSERT INTO chain_sync_cursors (module, chain_id, last_scanned_block, status)
			 VALUES ('indexer', ?, 0, 'running')`,
			chainId,
		)
		if err != nil {
			return 0, err
		}
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return lastBlock.Int64, nil
}

// updateSyncCursor 更新同步游标
func (s *BlockScanner) updateSyncCursor(ctx context.Context, blockNum int64) error {
	chainId := s.svcCtx.Config.Chain.ChainId
	_, err := s.svcCtx.DB.ExecContext(ctx,
		`UPDATE chain_sync_cursors 
		 SET last_scanned_block = ?, status = 'running', error_message = NULL
		 WHERE module = 'indexer' AND chain_id = ?`,
		blockNum, chainId,
	)
	return err
}

// retryWithBackoff 退避重试
func (s *BlockScanner) retryWithBackoff(maxRetries int, intervalMs int64) {
	interval := time.Duration(intervalMs) * time.Millisecond
	for i := 0; i < maxRetries; i++ {
		select {
		case <-s.stopCh:
			return
		case <-time.After(interval):
			return
		}
	}
}

// BackfillBlocks 补块：手动指定区块范围重扫
func (s *BlockScanner) BackfillBlocks(ctx context.Context, from, to int64) error {
	logx.Infof("backfilling blocks %d -> %d", from, to)

	batchSize := s.svcCtx.Config.Scanner.BatchSize
	for blockNum := from; blockNum <= to; blockNum += batchSize {
		end := blockNum + batchSize - 1
		if end > to {
			end = to
		}
		if err := s.scanBlockRange(ctx, blockNum, end); err != nil {
			return fmt.Errorf("backfill block range [%d, %d]: %w", blockNum, end, err)
		}
	}

	logx.Infof("backfill completed: %d -> %d", from, to)
	return nil
}

// 确保接口类型合规
var _ = (*types.Log)(nil)
var _ = common.Address{}
