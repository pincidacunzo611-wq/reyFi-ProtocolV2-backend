// Package jobs 实现 Bot 服务的定时任务调度
package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"github.com/reyfi/reyfi-backend/app/bot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// Scheduler 任务调度器
type Scheduler struct {
	svcCtx *svc.ServiceContext
}

func NewScheduler(svcCtx *svc.ServiceContext) *Scheduler {
	return &Scheduler{svcCtx: svcCtx}
}

// Start 启动所有定时任务
func (s *Scheduler) Start() {
	cfg := s.svcCtx.Config.Jobs

	// 1. 清算监控
	go s.runJob("liquidation_monitor", time.Duration(cfg.LiquidationMonitorInterval)*time.Second, s.monitorLiquidations)

	// 2. 价格更新
	go s.runJob("price_updater", time.Duration(cfg.PriceUpdateInterval)*time.Second, s.updatePrices)

	// 3. 资金费率结算
	go s.runJob("funding_settlement", time.Duration(cfg.FundingSettlementInterval)*time.Second, s.settleFunding)

	// 4. 期权到期检查
	go s.runJob("options_expiry", time.Duration(cfg.OptionsExpiryCheckInterval)*time.Second, s.checkOptionsExpiry)

	// 5. 日统计聚合（每天凌晨 1 点）
	go s.runDailyAggregation()

	logx.Info("all bot jobs started")
}

// runJob 通用定时任务运行器
func (s *Scheduler) runJob(name string, interval time.Duration, fn func(context.Context) error) {
	logx.Infof("job [%s] started with interval %v", name, interval)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), interval)
		startTime := time.Now()

		err := fn(ctx)
		duration := time.Since(startTime)

		status := "success"
		var errMsg string
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			logx.Errorf("job [%s] failed (took %v): %v", name, duration, err)
		} else {
			logx.Debugf("job [%s] completed (took %v)", name, duration)
		}

		// 记录运行历史
		s.recordJobRun(name, status, errMsg, duration)

		cancel()
		time.Sleep(interval)
	}
}

// recordJobRun 记录任务运行记录到 job_runs 表
func (s *Scheduler) recordJobRun(jobName, status, errMsg string, duration time.Duration) {
	s.svcCtx.DB.Exec(
		`INSERT INTO job_runs (job_name, status, error_message, duration_ms, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		jobName, status, errMsg, duration.Milliseconds(),
		time.Now().Add(-duration), time.Now())
}

// ==================== 具体任务实现 ====================

// monitorLiquidations 监控借贷仓位健康因子，识别濒临清算的仓位
func (s *Scheduler) monitorLiquidations(ctx context.Context) error {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 查询健康因子低于阈值的仓位
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT user_address, asset_address, health_factor, supplied_amount, borrowed_amount
		 FROM lending_user_positions
		 WHERE chain_id = ? AND CAST(health_factor AS DECIMAL(20,4)) < 1.2 AND CAST(health_factor AS DECIMAL(20,4)) > 0
		 ORDER BY CAST(health_factor AS DECIMAL(20,4)) ASC LIMIT 100`,
		chainId)
	if err != nil {
		return fmt.Errorf("query risky positions: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var userAddr, assetAddr, healthFactor, supplied, borrowed string
		rows.Scan(&userAddr, &assetAddr, &healthFactor, &supplied, &borrowed)

		// 记录到清算候选表
		s.svcCtx.DB.ExecContext(ctx,
			`INSERT INTO liquidation_candidates (chain_id, module, user_address, position_key,
			 health_factor, collateral_value, debt_value, status)
			 VALUES (?, 'lending', ?, ?, ?, ?, ?, 'pending')
			 ON DUPLICATE KEY UPDATE health_factor=VALUES(health_factor),
			 collateral_value=VALUES(collateral_value), debt_value=VALUES(debt_value), updated_at=NOW()`,
			chainId, userAddr, assetAddr, healthFactor, supplied, borrowed)

		count++
	}

	if count > 0 {
		logx.Errorf("found %d risky lending positions", count)

		// 记录告警
		s.svcCtx.DB.ExecContext(ctx,
			`INSERT INTO alert_records (module, alert_type, severity, message, data)
			 VALUES ('lending', 'liquidation_risk', 'high', ?, ?)`,
			fmt.Sprintf("发现 %d 个濒临清算的借贷仓位", count),
			fmt.Sprintf(`{"count": %d, "chainId": %d}`, count, chainId))
	}

	// 同样检查永续合约仓位
	futuresRows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT user_address, position_id, margin_ratio, margin, size
		 FROM futures_positions
		 WHERE chain_id = ? AND status = 'open'
		 AND CAST(margin AS DECIMAL(65,18)) > 0
		 ORDER BY CAST(margin AS DECIMAL(65,18)) ASC LIMIT 100`,
		chainId)
	if err == nil {
		defer futuresRows.Close()
		futuresCount := 0
		for futuresRows.Next() {
			var userAddr, posId string
			var marginRatio, margin, size sql.NullString
			futuresRows.Scan(&userAddr, &posId, &marginRatio, &margin, &size)

			s.svcCtx.DB.ExecContext(ctx,
				`INSERT INTO liquidation_candidates (chain_id, module, user_address, position_key,
				 health_factor, collateral_value, debt_value, status)
				 VALUES (?, 'futures', ?, ?, '0', ?, ?, 'pending')
				 ON DUPLICATE KEY UPDATE updated_at=NOW()`,
				chainId, userAddr, posId, margin, size)
			futuresCount++
		}
		if futuresCount > 0 {
			logx.Errorf("found %d risky futures positions", futuresCount)
		}
	}

	return nil
}

// updatePrices 更新资产价格（从 DEX 储备量或外部预言机）
func (s *Scheduler) updatePrices(ctx context.Context) error {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 查询所有活跃交易对的最新储备量
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT s.pair_address, s.reserve0, s.reserve1, p.token0_symbol, p.token1_symbol,
		 p.token0_decimals, p.token1_decimals
		 FROM dex_pair_snapshots s
		 INNER JOIN dex_pairs p ON p.chain_id = s.chain_id AND p.pair_address = s.pair_address
		 WHERE s.chain_id = ? AND p.is_active = 1
		 AND s.id = (SELECT MAX(id) FROM dex_pair_snapshots WHERE chain_id = s.chain_id AND pair_address = s.pair_address)`,
		chainId)
	if err != nil {
		return fmt.Errorf("query pair snapshots: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var pairAddr, reserve0, reserve1, sym0, sym1 string
		var dec0, dec1 int
		rows.Scan(&pairAddr, &reserve0, &reserve1, &sym0, &sym1, &dec0, &dec1)

		// 从储备量计算价格: price = reserve1 / reserve0
		r0, ok0 := new(big.Float).SetString(reserve0)
		r1, ok1 := new(big.Float).SetString(reserve1)
		if ok0 && ok1 && r0.Sign() > 0 {
			price := new(big.Float).Quo(r1, r0)
			priceStr := price.Text('f', 18)

			// 写入 Redis 缓存
			cacheKey := fmt.Sprintf("price:%s:%s", sym0, sym1)
			s.svcCtx.Redis.Setex(cacheKey, priceStr, 60)

			// 反向价格
			if r1.Sign() > 0 {
				reversePrice := new(big.Float).Quo(r0, r1)
				reverseCacheKey := fmt.Sprintf("price:%s:%s", sym1, sym0)
				s.svcCtx.Redis.Setex(reverseCacheKey, reversePrice.Text('f', 18), 60)
			}
		}
		count++
	}

	logx.Debugf("updated prices for %d pairs", count)
	return nil
}

// settleFunding 结算永续合约资金费率
func (s *Scheduler) settleFunding(ctx context.Context) error {
	chainId := s.svcCtx.Config.Chain.ChainId
	logx.Debugf("checking funding settlement for chain %d...", chainId)

	// 查询所有活跃市场
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT market_address, market_name FROM futures_markets WHERE chain_id = ? AND is_active = 1`,
		chainId)
	if err != nil {
		return fmt.Errorf("query futures markets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var marketAddr, marketName string
		rows.Scan(&marketAddr, &marketName)

		// 计算资金费率: 基于多空持仓比例
		var longOI, shortOI sql.NullString
		s.svcCtx.DB.QueryRowContext(ctx,
			`SELECT 
				COALESCE(SUM(CASE WHEN side='long' THEN CAST(size AS DECIMAL(65,18)) ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN side='short' THEN CAST(size AS DECIMAL(65,18)) ELSE 0 END), 0)
			FROM futures_positions WHERE chain_id = ? AND market_address = ? AND status = 'open'`,
			chainId, marketAddr).Scan(&longOI, &shortOI)

		// 简化资金费率计算: rate = (longOI - shortOI) / (longOI + shortOI) * 基础费率
		fundingRate := "0.0001" // 默认基础费率 0.01%
		if longOI.Valid && shortOI.Valid {
			lOI, _ := new(big.Float).SetString(longOI.String)
			sOI, _ := new(big.Float).SetString(shortOI.String)
			if lOI != nil && sOI != nil {
				totalOI := new(big.Float).Add(lOI, sOI)
				if totalOI.Sign() > 0 {
					diff := new(big.Float).Sub(lOI, sOI)
					rate := new(big.Float).Quo(diff, totalOI)
					rate.Mul(rate, big.NewFloat(0.0001))
					fundingRate = rate.Text('f', 8)
				}
			}
		}

		// 记录资金费率
		s.svcCtx.DB.ExecContext(ctx,
			`INSERT INTO futures_funding_history (chain_id, market_address, funding_rate, settlement_time)
			 VALUES (?, ?, ?, NOW())`,
			chainId, marketAddr, fundingRate)

		logx.Debugf("funding settled: market=%s, rate=%s", marketName, fundingRate)
	}

	return nil
}

// checkOptionsExpiry 检查期权到期
func (s *Scheduler) checkOptionsExpiry(ctx context.Context) error {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 找到已到期但未结算的期权
	result, err := s.svcCtx.DB.ExecContext(ctx,
		`UPDATE options_positions SET status = 'expired', updated_at = NOW()
		 WHERE chain_id = ? AND status = 'open' AND expiry_time < NOW()`,
		chainId)
	if err != nil {
		return fmt.Errorf("expire options: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		logx.Infof("expired %d options positions", affected)
	}
	return nil
}

// runDailyAggregation 每日数据聚合
func (s *Scheduler) runDailyAggregation() {
	for {
		now := time.Now().UTC()
		// 计算下一个凌晨 1 点
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 1, 0, 0, 0, time.UTC)
		duration := next.Sub(now)
		logx.Infof("daily aggregation scheduled in %v", duration)

		time.Sleep(duration)

		ctx := context.Background()
		s.aggregateDailyStats(ctx)
	}
}

// aggregateDailyStats 聚合每日统计数据
func (s *Scheduler) aggregateDailyStats(ctx context.Context) {
	chainId := s.svcCtx.Config.Chain.ChainId
	yesterday := time.Now().UTC().Add(-24 * time.Hour).Truncate(24 * time.Hour)
	today := yesterday.Add(24 * time.Hour)

	logx.Infof("aggregating daily stats for %s", yesterday.Format("2006-01-02"))

	// DEX 日统计
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT pair_address,
		 COUNT(*) as trades,
		 COALESCE(SUM(CAST(amount_usd AS DECIMAL(65,18))), 0) as volume
		 FROM dex_trades
		 WHERE chain_id = ? AND block_time >= ? AND block_time < ?
		 GROUP BY pair_address`,
		chainId, yesterday, today)
	if err != nil {
		logx.Errorf("aggregate dex daily: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var pairAddr string
		var trades int64
		var volume string
		rows.Scan(&pairAddr, &trades, &volume)

		s.svcCtx.DB.ExecContext(ctx,
			`INSERT INTO dex_pair_stats_daily (chain_id, pair_address, date, volume_usd, trade_count, unique_traders, tvl_usd, fee_usd)
			 VALUES (?, ?, ?, ?, ?, 0, 0, 0)
			 ON DUPLICATE KEY UPDATE volume_usd=VALUES(volume_usd), trade_count=VALUES(trade_count)`,
			chainId, pairAddr, yesterday.Format("2006-01-02"), volume, trades)
		count++
	}

	// 用户资产快照
	s.svcCtx.DB.ExecContext(ctx,
		`INSERT INTO user_asset_snapshots (chain_id, wallet_address, snapshot_date, total_asset_usd, total_debt_usd, net_value_usd)
		 SELECT ?, wallet_address, ?, '0', '0', '0'
		 FROM users WHERE status = 'active'
		 ON DUPLICATE KEY UPDATE id=id`,
		chainId, yesterday.Format("2006-01-02"))

	s.recordJobRun("daily_aggregation", "success", "", time.Second)
	logx.Infof("daily aggregation completed: %d pairs stats aggregated", count)
}
