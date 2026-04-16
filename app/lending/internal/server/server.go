package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/reyfi/reyfi-backend/app/lending/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

type LendingServiceServer struct {
	svcCtx *svc.ServiceContext
}

func NewLendingServiceServer(svcCtx *svc.ServiceContext) *LendingServiceServer {
	return &LendingServiceServer{svcCtx: svcCtx}
}

func RegisterLendingServiceServer(s *grpc.Server, srv *LendingServiceServer) {
	logx.Info("lending service registered")
}

// ==================== GetMarkets ====================

type MarketInfoResp struct {
	AssetAddress         string `json:"assetAddress"`
	AssetSymbol          string `json:"assetSymbol"`
	AssetDecimals        int    `json:"assetDecimals"`
	TotalSupply          string `json:"totalSupply"`
	TotalBorrow          string `json:"totalBorrow"`
	SupplyApr            string `json:"supplyApr"`
	BorrowApr            string `json:"borrowApr"`
	UtilizationRate      string `json:"utilizationRate"`
	TvlUsd               string `json:"tvlUsd"`
	CollateralFactor     string `json:"collateralFactor"`
	LiquidationThreshold string `json:"liquidationThreshold"`
	IsActive             bool   `json:"isActive"`
}

type GetMarketsResp struct {
	List  []*MarketInfoResp `json:"list"`
	Total int64             `json:"total"`
}

func (s *LendingServiceServer) GetMarkets(ctx context.Context, keyword string, page, pageSize int64) (*GetMarketsResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	cacheKey := fmt.Sprintf("lending:markets:page:%d:%s", page, keyword)
	if cached, err := s.svcCtx.Redis.Get(cacheKey); err == nil && cached != "" {
		var resp GetMarketsResp
		if json.Unmarshal([]byte(cached), &resp) == nil {
			return &resp, nil
		}
	}

	markets, total, err := s.svcCtx.MarketModel.ListMarkets(ctx, chainId, keyword, page, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]*MarketInfoResp, 0, len(markets))
	for _, m := range markets {
		info := &MarketInfoResp{
			AssetAddress:         m.AssetAddress,
			AssetSymbol:          m.AssetSymbol,
			AssetDecimals:        m.AssetDecimals,
			CollateralFactor:     m.CollateralFactor,
			LiquidationThreshold: m.LiquidationThreshold,
			IsActive:             m.IsActive,
		}
		snap, err := s.svcCtx.SnapshotModel.GetLatest(ctx, chainId, m.AssetAddress)
		if err == nil && snap != nil {
			info.TotalSupply = snap.TotalSupply
			info.TotalBorrow = snap.TotalBorrow
			info.SupplyApr = snap.SupplyApr
			info.BorrowApr = snap.BorrowApr
			info.UtilizationRate = snap.UtilizationRate
			info.TvlUsd = snap.TvlUsd
		}
		list = append(list, info)
	}

	resp := &GetMarketsResp{List: list, Total: total}
	if data, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(data), 15)
	}
	return resp, nil
}

// ==================== GetUserPosition ====================

type UserPositionResp struct {
	AssetAddress      string `json:"assetAddress"`
	AssetSymbol       string `json:"assetSymbol"`
	SuppliedAmount    string `json:"suppliedAmount"`
	SuppliedUsd       string `json:"suppliedUsd"`
	BorrowedAmount    string `json:"borrowedAmount"`
	BorrowedUsd       string `json:"borrowedUsd"`
	SupplyApr         string `json:"supplyApr"`
	BorrowApr         string `json:"borrowApr"`
	CollateralEnabled bool   `json:"collateralEnabled"`
}

type GetUserPositionResp struct {
	Positions        []*UserPositionResp `json:"positions"`
	TotalSuppliedUsd string              `json:"totalSuppliedUsd"`
	TotalBorrowedUsd string              `json:"totalBorrowedUsd"`
	NetApr           string              `json:"netApr"`
	HealthFactor     string              `json:"healthFactor"`
	BorrowLimit      string              `json:"borrowLimit"`
	BorrowLimitUsed  string              `json:"borrowLimitUsed"`
}

func (s *LendingServiceServer) GetUserPosition(ctx context.Context, userAddress string) (*GetUserPositionResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	positions, err := s.svcCtx.UserPositionModel.ListByUser(ctx, chainId, userAddress)
	if err != nil {
		return nil, err
	}

	list := make([]*UserPositionResp, 0, len(positions))
	for _, p := range positions {
		market, _ := s.svcCtx.MarketModel.FindByAsset(ctx, chainId, p.AssetAddress)
		symbol := ""
		if market != nil {
			symbol = market.AssetSymbol
		}
		list = append(list, &UserPositionResp{
			AssetAddress:      p.AssetAddress,
			AssetSymbol:       symbol,
			SuppliedAmount:    p.SuppliedAmount,
			SuppliedUsd:       "0",
			BorrowedAmount:    p.BorrowedAmount,
			BorrowedUsd:       "0",
			CollateralEnabled: p.CollateralEnabled,
		})
	}

	// 计算汇总健康因子
	var minHealth sql.NullString
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT MIN(health_factor) FROM lending_user_positions WHERE chain_id = ? AND user_address = ?`,
		chainId, userAddress).Scan(&minHealth)

	healthFactor := "999999.99"
	if minHealth.Valid {
		healthFactor = minHealth.String
	}

	return &GetUserPositionResp{
		Positions:        list,
		TotalSuppliedUsd: "0",
		TotalBorrowedUsd: "0",
		NetApr:           "0",
		HealthFactor:     healthFactor,
		BorrowLimit:      "0",
		BorrowLimitUsed:  "0",
	}, nil
}

// ==================== GetMarketDetail ====================

func (s *LendingServiceServer) GetMarketDetail(ctx context.Context, assetAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	market, err := s.svcCtx.MarketModel.FindByAsset(ctx, chainId, assetAddress)
	if err != nil {
		return nil, fmt.Errorf("market not found: %w", err)
	}

	info := &MarketInfoResp{
		AssetAddress:         market.AssetAddress,
		AssetSymbol:          market.AssetSymbol,
		AssetDecimals:        market.AssetDecimals,
		CollateralFactor:     market.CollateralFactor,
		LiquidationThreshold: market.LiquidationThreshold,
		IsActive:             market.IsActive,
	}

	snap, err := s.svcCtx.SnapshotModel.GetLatest(ctx, chainId, assetAddress)
	if err == nil && snap != nil {
		info.TotalSupply = snap.TotalSupply
		info.TotalBorrow = snap.TotalBorrow
		info.SupplyApr = snap.SupplyApr
		info.BorrowApr = snap.BorrowApr
		info.UtilizationRate = snap.UtilizationRate
		info.TvlUsd = snap.TvlUsd
	}

	// 可用流动性 = TotalSupply - TotalBorrow
	availableLiquidity := "0"
	if info.TotalSupply != "" && info.TotalBorrow != "" {
		supply := new(big.Float)
		borrow := new(big.Float)
		if _, ok := supply.SetString(info.TotalSupply); ok {
			if _, ok := borrow.SetString(info.TotalBorrow); ok {
				avail := new(big.Float).Sub(supply, borrow)
				availableLiquidity = avail.Text('f', 0)
			}
		}
	}

	// 利率历史（最近 30 条快照）
	rateHistory := make([]map[string]interface{}, 0)
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT snapshot_time, supply_apr, borrow_apr, utilization_rate
		 FROM lending_market_snapshots 
		 WHERE chain_id = ? AND asset_address = ?
		 ORDER BY snapshot_time DESC LIMIT 30`,
		chainId, assetAddress)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var snapTime time.Time
			var supplyApr, borrowApr, utilization string
			rows.Scan(&snapTime, &supplyApr, &borrowApr, &utilization)
			rateHistory = append(rateHistory, map[string]interface{}{
				"timestamp":   snapTime.Unix(),
				"supplyApr":   supplyApr,
				"borrowApr":   borrowApr,
				"utilization": utilization,
			})
		}
	}

	return map[string]interface{}{
		"market":             info,
		"availableLiquidity": availableLiquidity,
		"reserveFactor":      "0.10",
		"liquidationPenalty": "0.05",
		"rateHistory":        rateHistory,
	}, nil
}

// ==================== GetLiquidations ====================

func (s *LendingServiceServer) GetLiquidations(ctx context.Context, assetAddress, userAddress string, page, pageSize int64) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	liqs, total, err := s.svcCtx.LiquidationModel.ListByBorrower(ctx, chainId, userAddress, page, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]interface{}, 0, len(liqs))
	for _, l := range liqs {
		list = append(list, map[string]interface{}{
			"borrower":         l.BorrowerAddress,
			"liquidator":       l.LiquidatorAddress,
			"collateralAsset":  l.CollateralAsset,
			"debtAsset":        l.DebtAsset,
			"collateralAmount": l.CollateralAmount,
			"debtAmount":       l.DebtAmount,
			"penaltyAmount":    l.PenaltyAmount,
			"txHash":           l.TxHash,
			"blockTime":        l.BlockTime.UTC().Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"list":  list,
		"total": total,
	}, nil
}

// ==================== Build* 交易构建方法 ====================

// buildLendingTx 通用借贷交易构建辅助函数
func (s *LendingServiceServer) buildLendingTx(method, userAddress, assetAddress, amount string) (map[string]interface{}, error) {
	lendingPool := s.svcCtx.Config.Chain.Contracts["lendingPool"]
	if lendingPool == "" {
		lendingPool = s.svcCtx.Config.Chain.Contracts["LendingPool"]
	}

	return map[string]interface{}{
		"to":       lendingPool,
		"value":    "0",
		"gasLimit": "300000",
		"data": fmt.Sprintf(`{"method":"%s","params":{"asset":"%s","amount":"%s","onBehalfOf":"%s"}}`,
			method, assetAddress, amount, userAddress),
		"estimatedGasUsd": "0",
	}, nil
}

func (s *LendingServiceServer) BuildSupply(ctx context.Context, userAddress, assetAddress, amount string) (interface{}, error) {
	logx.Infof("build supply: user=%s, asset=%s, amount=%s", userAddress, assetAddress, amount)
	return s.buildLendingTx("supply", userAddress, assetAddress, amount)
}

func (s *LendingServiceServer) BuildBorrow(ctx context.Context, userAddress, assetAddress, amount string) (interface{}, error) {
	logx.Infof("build borrow: user=%s, asset=%s, amount=%s", userAddress, assetAddress, amount)
	return s.buildLendingTx("borrow", userAddress, assetAddress, amount)
}

func (s *LendingServiceServer) BuildRepay(ctx context.Context, userAddress, assetAddress, amount string) (interface{}, error) {
	logx.Infof("build repay: user=%s, asset=%s, amount=%s", userAddress, assetAddress, amount)
	return s.buildLendingTx("repay", userAddress, assetAddress, amount)
}

func (s *LendingServiceServer) BuildWithdraw(ctx context.Context, userAddress, assetAddress, amount string) (interface{}, error) {
	logx.Infof("build withdraw: user=%s, asset=%s, amount=%s", userAddress, assetAddress, amount)
	return s.buildLendingTx("withdraw", userAddress, assetAddress, amount)
}
