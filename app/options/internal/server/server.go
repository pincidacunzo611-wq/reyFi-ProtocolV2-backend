package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reyfi/reyfi-backend/app/options/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

type OptionsServiceServer struct{ svcCtx *svc.ServiceContext }

func NewOptionsServiceServer(svcCtx *svc.ServiceContext) *OptionsServiceServer {
	return &OptionsServiceServer{svcCtx: svcCtx}
}
func RegisterOptionsServiceServer(s *grpc.Server, srv *OptionsServiceServer) {
	logx.Info("options service registered")
}

// ==================== GetChains ====================

func (s *OptionsServiceServer) GetChains(ctx context.Context) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	markets, _, err := s.svcCtx.MarketModel.ListMarkets(ctx, chainId, 1, 100)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]interface{}, 0, len(markets))
	for _, m := range markets {
		list = append(list, map[string]interface{}{
			"marketAddress":    m.MarketAddress,
			"underlyingSymbol": m.UnderlyingSymbol,
			"isActive":         m.IsActive,
		})
	}
	return map[string]interface{}{"list": list}, nil
}

// ==================== GetPositions ====================

func (s *OptionsServiceServer) GetPositions(ctx context.Context, userAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	positions, err := s.svcCtx.PositionModel.ListByUser(ctx, chainId, userAddress)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]interface{}, 0, len(positions))
	for _, p := range positions {
		pnl := "0"
		if p.Pnl.Valid {
			pnl = p.Pnl.String
		}
		list = append(list, map[string]interface{}{
			"optionId":    p.OptionId,
			"optionType":  p.OptionType,
			"strikePrice": p.StrikePrice,
			"premium":     p.Premium,
			"size":        p.Size,
			"expiry":      p.ExpiryTime.UTC().Format(time.RFC3339),
			"status":      p.Status,
			"pnl":         pnl,
		})
	}
	return map[string]interface{}{"list": list}, nil
}

// ==================== GetVolSurface ====================

func (s *OptionsServiceServer) GetVolSurface(ctx context.Context, underlying string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	surfaces, err := s.svcCtx.VolModel.GetLatestByUnderlying(ctx, chainId, underlying)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]interface{}, 0, len(surfaces))
	for _, v := range surfaces {
		iv := "0"
		if v.ImpliedVol.Valid {
			iv = v.ImpliedVol.String
		}
		list = append(list, map[string]interface{}{
			"strikePrice": v.StrikePrice,
			"expiry":      v.ExpiryTime.UTC().Format(time.RFC3339),
			"impliedVol":  iv,
		})
	}
	return map[string]interface{}{"volSurface": list}, nil
}

// ==================== Build* 交易构建 ====================

func (s *OptionsServiceServer) buildOptionsTx(method string, params map[string]interface{}) (interface{}, error) {
	optionsMarket := s.svcCtx.Config.Chain.Contracts["optionsMarket"]
	if optionsMarket == "" {
		optionsMarket = s.svcCtx.Config.Chain.Contracts["OptionsMarket"]
	}
	data, _ := json.Marshal(map[string]interface{}{"method": method, "params": params})
	return map[string]interface{}{
		"to": optionsMarket, "value": "0", "gasLimit": "400000",
		"data": string(data),
	}, nil
}

func (s *OptionsServiceServer) BuildWriteOption(ctx context.Context, userAddr, optionAddr, amount, collateralAmount string) (interface{}, error) {
	logx.Infof("build write option: user=%s, option=%s, amount=%s", userAddr, optionAddr, amount)
	return s.buildOptionsTx("writeOption", map[string]interface{}{
		"optionAddress": optionAddr, "amount": amount,
		"collateralAmount": collateralAmount, "user": userAddr,
	})
}

func (s *OptionsServiceServer) BuildBuyOption(ctx context.Context, userAddr, optionAddr, amount, maxPremium string) (interface{}, error) {
	logx.Infof("build buy option: user=%s, option=%s, amount=%s", userAddr, optionAddr, amount)

	// 校验期权未过期
	chainId := s.svcCtx.Config.Chain.ChainId
	var expiryTime time.Time
	err := s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT expiry_time FROM options_markets WHERE chain_id = ? AND market_address = ?`,
		chainId, optionAddr).Scan(&expiryTime)
	if err == nil && expiryTime.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("option has expired")
	}

	return s.buildOptionsTx("buyOption", map[string]interface{}{
		"optionAddress": optionAddr, "amount": amount,
		"maxPremium": maxPremium, "user": userAddr,
	})
}

func (s *OptionsServiceServer) BuildExercise(ctx context.Context, userAddr, positionId string) (interface{}, error) {
	logx.Infof("build exercise: user=%s, positionId=%s", userAddr, positionId)
	return s.buildOptionsTx("exercise", map[string]interface{}{
		"positionId": positionId, "user": userAddr,
	})
}
