package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reyfi/reyfi-backend/app/bonds/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

type BondsServiceServer struct{ svcCtx *svc.ServiceContext }

func NewBondsServiceServer(svcCtx *svc.ServiceContext) *BondsServiceServer {
	return &BondsServiceServer{svcCtx: svcCtx}
}
func RegisterBondsServiceServer(s *grpc.Server, srv *BondsServiceServer) {
	logx.Info("bonds service registered")
}

// ==================== GetMarkets ====================

func (s *BondsServiceServer) GetMarkets(ctx context.Context, page, pageSize int64) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	cacheKey := fmt.Sprintf("bonds:markets:page:%d", page)
	if c, err := s.svcCtx.Redis.Get(cacheKey); err == nil && c != "" {
		var r interface{}
		json.Unmarshal([]byte(c), &r)
		return r, nil
	}
	markets, total, err := s.svcCtx.MarketModel.ListMarkets(ctx, chainId, page, pageSize)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]interface{}, 0, len(markets))
	for _, m := range markets {
		list = append(list, map[string]interface{}{
			"marketAddress": m.MarketAddress, "paymentToken": m.PaymentToken,
			"payoutToken": m.PayoutToken, "vestingTerm": m.VestingTerm,
			"discountRate": m.DiscountRate, "currentDebt": m.CurrentDebt, "isActive": m.IsActive,
		})
	}
	resp := map[string]interface{}{"list": list, "total": total}
	if d, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(d), 15)
	}
	return resp, nil
}

// ==================== GetMarketDetail ====================

func (s *BondsServiceServer) GetMarketDetail(ctx context.Context, marketAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	market, err := s.svcCtx.MarketModel.FindByAddress(ctx, chainId, marketAddress)
	if err != nil {
		return nil, fmt.Errorf("bond market not found: %w", err)
	}

	info := map[string]interface{}{
		"marketAddress": market.MarketAddress, "paymentToken": market.PaymentToken,
		"payoutToken": market.PayoutToken, "vestingTerm": market.VestingTerm,
		"discountRate": market.DiscountRate, "currentDebt": market.CurrentDebt,
		"isActive": market.IsActive,
	}

	// 总购买量
	var totalPurchased string
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CAST(payment_amount AS DECIMAL(65,18))), 0) FROM bond_positions
		 WHERE chain_id = ? AND market_address = ?`,
		chainId, marketAddress).Scan(&totalPurchased)

	return map[string]interface{}{
		"market":         info,
		"minPrice":       "0",
		"roi":            market.DiscountRate,
		"totalPurchased": totalPurchased,
	}, nil
}

// ==================== GetPositions ====================

func (s *BondsServiceServer) GetPositions(ctx context.Context, userAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	positions, err := s.svcCtx.PositionModel.ListByUser(ctx, chainId, userAddress)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]interface{}, 0, len(positions))
	for _, p := range positions {
		list = append(list, map[string]interface{}{
			"bondId": p.BondId, "marketAddress": p.MarketAddress,
			"paymentAmount": p.PaymentAmount, "payoutAmount": p.PayoutAmount,
			"vestedAmount": p.VestedAmount, "claimableAmount": p.ClaimableAmount,
			"vestingStart": p.VestingStart.UTC().Format(time.RFC3339),
			"vestingEnd":   p.VestingEnd.UTC().Format(time.RFC3339),
			"status":       p.Status,
		})
	}
	return map[string]interface{}{"list": list}, nil
}

// ==================== Build* 交易构建 ====================

func (s *BondsServiceServer) BuildPurchase(ctx context.Context, userAddr, marketAddr, amount string) (interface{}, error) {
	logx.Infof("build bond purchase: user=%s, market=%s, amount=%s", userAddr, marketAddr, amount)
	bondDepository := s.svcCtx.Config.Chain.Contracts["bondDepository"]
	if bondDepository == "" {
		bondDepository = s.svcCtx.Config.Chain.Contracts["BondDepository"]
	}
	data, _ := json.Marshal(map[string]interface{}{
		"method": "deposit",
		"params": map[string]interface{}{
			"market": marketAddr, "amount": amount,
			"maxPrice": "0", "user": userAddr,
		},
	})
	return map[string]interface{}{
		"to": bondDepository, "value": "0", "gasLimit": "350000",
		"data": string(data),
	}, nil
}

func (s *BondsServiceServer) BuildClaim(ctx context.Context, userAddr, bondId string) (interface{}, error) {
	logx.Infof("build bond claim: user=%s, bondId=%s", userAddr, bondId)
	bondDepository := s.svcCtx.Config.Chain.Contracts["bondDepository"]
	if bondDepository == "" {
		bondDepository = s.svcCtx.Config.Chain.Contracts["BondDepository"]
	}
	data, _ := json.Marshal(map[string]interface{}{
		"method": "redeem",
		"params": map[string]interface{}{"bondId": bondId, "user": userAddr},
	})
	return map[string]interface{}{
		"to": bondDepository, "value": "0", "gasLimit": "200000",
		"data": string(data),
	}, nil
}
