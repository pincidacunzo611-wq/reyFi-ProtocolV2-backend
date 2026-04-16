package indexer

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// ReorgDetector 链重组检测器
// 通过对比本地存储的区块哈希与链上的父哈希来检测分叉
type ReorgDetector struct {
	svcCtx *svc.ServiceContext
}

// NewReorgDetector 创建重组检测器
func NewReorgDetector(svcCtx *svc.ServiceContext) *ReorgDetector {
	return &ReorgDetector{svcCtx: svcCtx}
}

// Detect 检测是否发生了链重组
// 返回: (是否重组, 分叉块号, 错误)
func (d *ReorgDetector) Detect(ctx context.Context, blockNum int64) (bool, int64, error) {
	chainId := d.svcCtx.Config.Chain.ChainId

	// 获取本地存储的当前区块的哈希
	var localHash string
	err := d.svcCtx.DB.QueryRowContext(ctx,
		`SELECT block_hash FROM chain_blocks WHERE chain_id = ? AND block_number = ?`,
		chainId, blockNum,
	).Scan(&localHash)

	if err == sql.ErrNoRows {
		return false, 0, nil // 本地没有这个区块，不算重组
	}
	if err != nil {
		return false, 0, fmt.Errorf("query local block %d: %w", blockNum, err)
	}

	// 获取链上对应区块的哈希
	header, err := d.svcCtx.Chain.HeaderByNumber(ctx, blockNum)
	if err != nil {
		return false, 0, fmt.Errorf("query chain block %d: %w", blockNum, err)
	}

	chainHash := header.Hash().Hex()

	// 如果哈希一致，没有重组
	if localHash == chainHash {
		return false, 0, nil
	}

	logx.Errorf("reorg detected at block %d: local=%s, chain=%s", blockNum, localHash, chainHash)

	// 找到分叉点：向前回溯直到找到一致的区块
	forkBlock := blockNum
	for forkBlock > 0 {
		forkBlock--

		var localParentHash string
		err := d.svcCtx.DB.QueryRowContext(ctx,
			`SELECT block_hash FROM chain_blocks WHERE chain_id = ? AND block_number = ?`,
			chainId, forkBlock,
		).Scan(&localParentHash)

		if err == sql.ErrNoRows {
			break // 到达本地存储的最早区块
		}
		if err != nil {
			return true, forkBlock, fmt.Errorf("query local block %d: %w", forkBlock, err)
		}

		parentHeader, err := d.svcCtx.Chain.HeaderByNumber(ctx, forkBlock)
		if err != nil {
			return true, forkBlock, fmt.Errorf("query chain block %d: %w", forkBlock, err)
		}

		if localParentHash == parentHeader.Hash().Hex() {
			// 找到分叉点：这个区块一致，下一个区块开始分叉
			forkBlock++
			break
		}
	}

	logx.Errorf("fork point found at block %d", forkBlock)
	return true, forkBlock, nil
}

// Rollback 回滚指定区块之后的所有数据
func (d *ReorgDetector) Rollback(ctx context.Context, fromBlock int64) error {
	chainId := d.svcCtx.Config.Chain.ChainId

	logx.Errorf("rolling back data from block %d on chain %d", fromBlock, chainId)

	// 1. 将原始事件标记为 reverted
	result, err := d.svcCtx.DB.ExecContext(ctx,
		`UPDATE chain_raw_events 
		 SET status = 'reverted' 
		 WHERE chain_id = ? AND block_number >= ? AND status != 'reverted'`,
		chainId, fromBlock,
	)
	if err != nil {
		return fmt.Errorf("revert raw events: %w", err)
	}
	affected, _ := result.RowsAffected()
	logx.Infof("reverted %d raw events", affected)

	// 2. 删除分叉后的区块记录
	_, err = d.svcCtx.DB.ExecContext(ctx,
		`DELETE FROM chain_blocks WHERE chain_id = ? AND block_number >= ?`,
		chainId, fromBlock,
	)
	if err != nil {
		return fmt.Errorf("delete blocks: %w", err)
	}

	// 3. 更新同步游标
	_, err = d.svcCtx.DB.ExecContext(ctx,
		`UPDATE chain_sync_cursors 
		 SET last_scanned_block = ?, error_message = 'reorg rollback'
		 WHERE module = 'indexer' AND chain_id = ?`,
		fromBlock-1, chainId,
	)
	if err != nil {
		return fmt.Errorf("update sync cursor: %w", err)
	}

	logx.Infof("rollback completed, cursor reset to block %d", fromBlock-1)
	return nil
}
