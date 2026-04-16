package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reyfi/reyfi-backend/app/futures/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

type FuturesServiceServer struct{ svcCtx *svc.ServiceContext }

func NewFuturesServiceServer(svcCtx *svc.ServiceContext) *FuturesServiceServer {
	return &FuturesServiceServer{svcCtx: svcCtx}
}
func RegisterFuturesServiceServer(s *grpc.Server, srv *FuturesServiceServer) {
	logx.Info("futures service registered")
}

// ==================== GetMarkets ====================

type FuturesMarketResp struct {
	MarketAddress string `json:"marketAddress"`
	MarketName    string `json:"marketName"`
	MarkPrice     string `json:"markPrice"`
	IndexPrice    string `json:"indexPrice"`
	FundingRate   string `json:"fundingRate"`
	Volume24h     string `json:"volume24h"`
	OpenInterest  string `json:"openInterest"`
	MaxLeverage   string `json:"maxLeverage"`
}

func (s *FuturesServiceServer) GetMarkets(ctx context.Context, page, pageSize int64) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	cacheKey := fmt.Sprintf("futures:markets:page:%d", page)
	if c, err := s.svcCtx.Redis.Get(cacheKey); err == nil && c != "" {
		var resp interface{}
		json.Unmarshal([]byte(c), &resp)
		return resp, nil
	}

	markets, total, err := s.svcCtx.MarketModel.ListMarkets(ctx, chainId, page, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]*FuturesMarketResp, 0, len(markets))
	for _, m := range markets {
		info := &FuturesMarketResp{
			MarketAddress: m.MarketAddress,
			MarketName:    m.MarketName,
			MaxLeverage:   m.MaxLeverage,
			MarkPrice:     m.MarkPrice,
			IndexPrice:    m.IndexPrice,
			FundingRate:   "0",
			Volume24h:     "0",
			OpenInterest:  "0",
		}

		// 查最近一次资金费率
		var latestRate string
		err := s.svcCtx.DB.QueryRowContext(ctx,
			`SELECT funding_rate FROM futures_funding_history 
			 WHERE chain_id = ? AND market_address = ? ORDER BY settlement_time DESC LIMIT 1`,
			chainId, m.MarketAddress).Scan(&latestRate)
		if err == nil {
			info.FundingRate = latestRate
		}

		// 24h 交易量（从仓位变更聚合）
		var vol24h string
		s.svcCtx.DB.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(CAST(size AS DECIMAL(65,18))), 0) FROM futures_positions
			 WHERE chain_id = ? AND market_address = ? AND updated_at >= ?`,
			chainId, m.MarketAddress, time.Now().UTC().Add(-24*time.Hour)).Scan(&vol24h)
		info.Volume24h = vol24h

		// 未平仓量
		var oi string
		s.svcCtx.DB.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(CAST(size AS DECIMAL(65,18))), 0) FROM futures_positions
			 WHERE chain_id = ? AND market_address = ? AND status = 'open'`,
			chainId, m.MarketAddress).Scan(&oi)
		info.OpenInterest = oi

		list = append(list, info)
	}
	resp := map[string]interface{}{"list": list, "total": total}
	if data, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(data), 5)
	}
	return resp, nil
}

// ==================== GetMarketDetail ====================

func (s *FuturesServiceServer) GetMarketDetail(ctx context.Context, marketAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	market, err := s.svcCtx.MarketModel.FindByAddress(ctx, chainId, marketAddress)
	if err != nil {
		return nil, fmt.Errorf("market not found: %w", err)
	}

	info := &FuturesMarketResp{
		MarketAddress: market.MarketAddress,
		MarketName:    market.MarketName,
		MaxLeverage:   market.MaxLeverage,
		MarkPrice:     market.MarkPrice,
		IndexPrice:    market.IndexPrice,
	}

	// 最近资金费率历史
	fundingHistory := make([]map[string]interface{}, 0)
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT funding_rate, settlement_time FROM futures_funding_history
		 WHERE chain_id = ? AND market_address = ?
		 ORDER BY settlement_time DESC LIMIT 24`,
		chainId, marketAddress)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rate string
			var t time.Time
			rows.Scan(&rate, &t)
			fundingHistory = append(fundingHistory, map[string]interface{}{
				"rate": rate, "time": t.Unix(),
			})
		}
	}

	return map[string]interface{}{
		"market":                  info,
		"takerFee":                "0.0006",
		"makerFee":                "0.0001",
		"maintenanceMarginRate":   "0.005",
		"fundingHistory":          fundingHistory,
	}, nil
}

// ==================== GetPositions ====================

type FuturesPositionResp struct {
	PositionId       string `json:"positionId"`
	MarketName       string `json:"marketName"`
	Side             string `json:"side"`
	Size             string `json:"size"`
	EntryPrice       string `json:"entryPrice"`
	MarkPrice        string `json:"markPrice"`
	Margin           string `json:"margin"`
	Leverage         string `json:"leverage"`
	UnrealizedPnl    string `json:"unrealizedPnl"`
	LiquidationPrice string `json:"liquidationPrice"`
}

func (s *FuturesServiceServer) GetPositions(ctx context.Context, userAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	positions, err := s.svcCtx.PositionModel.ListByUser(ctx, chainId, userAddress)
	if err != nil {
		return nil, err
	}

	list := make([]*FuturesPositionResp, 0, len(positions))
	totalPnl := "0"
	totalMargin := "0"
	for _, p := range positions {
		market, _ := s.svcCtx.MarketModel.FindByAddress(ctx, chainId, p.MarketAddress)
		name := p.MarketAddress
		if market != nil {
			name = market.MarketName
		}
		list = append(list, &FuturesPositionResp{
			PositionId: p.PositionId, MarketName: name, Side: p.Side,
			Size: p.Size, EntryPrice: p.EntryPrice, MarkPrice: p.MarkPrice,
			Margin: p.Margin, Leverage: p.Leverage,
			UnrealizedPnl: p.UnrealizedPnl, LiquidationPrice: p.LiquidationPrice,
		})
	}
	return map[string]interface{}{"list": list, "totalUnrealizedPnl": totalPnl, "totalMargin": totalMargin}, nil
}

// ==================== GetFundingHistory ====================

func (s *FuturesServiceServer) GetFundingHistory(ctx context.Context, marketAddr string, page, pageSize int64) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	records, total, err := s.svcCtx.FundingModel.ListByMarket(ctx, chainId, marketAddr, page, pageSize)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		list = append(list, map[string]interface{}{
			"rate": r.FundingRate, "time": r.SettlementTime.UTC().Format(time.RFC3339),
		})
	}
	return map[string]interface{}{"list": list, "total": total}, nil
}

// ==================== Build* 交易构建 ====================

func (s *FuturesServiceServer) buildFuturesTx(method string, params map[string]interface{}) (interface{}, error) {
	perpExchange := s.svcCtx.Config.Chain.Contracts["perpExchange"]
	if perpExchange == "" {
		perpExchange = s.svcCtx.Config.Chain.Contracts["PerpExchange"]
	}
	data, _ := json.Marshal(map[string]interface{}{"method": method, "params": params})
	return map[string]interface{}{
		"to": perpExchange, "value": "0", "gasLimit": "500000",
		"data": string(data),
	}, nil
}

func (s *FuturesServiceServer) BuildOpenPosition(ctx context.Context, userAddr, marketAddr, side, margin, leverage, limitPrice string) (interface{}, error) {
	logx.Infof("build open position: user=%s, market=%s, side=%s", userAddr, marketAddr, side)
	return s.buildFuturesTx("openPosition", map[string]interface{}{
		"market": marketAddr, "side": side, "margin": margin,
		"leverage": leverage, "limitPrice": limitPrice, "user": userAddr,
	})
}

func (s *FuturesServiceServer) BuildClosePosition(ctx context.Context, userAddr, positionId, amount string) (interface{}, error) {
	logx.Infof("build close position: user=%s, positionId=%s", userAddr, positionId)
	return s.buildFuturesTx("closePosition", map[string]interface{}{
		"positionId": positionId, "amount": amount, "user": userAddr,
	})
}

func (s *FuturesServiceServer) BuildAdjustMargin(ctx context.Context, userAddr, positionId, amount string, isAdd bool) (interface{}, error) {
	method := "addMargin"
	if !isAdd {
		method = "removeMargin"
	}
	logx.Infof("build %s: user=%s, positionId=%s, amount=%s", method, userAddr, positionId, amount)
	return s.buildFuturesTx(method, map[string]interface{}{
		"positionId": positionId, "amount": amount, "user": userAddr,
	})
}
