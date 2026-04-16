package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reyfi/reyfi-backend/app/vault/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

type VaultServiceServer struct{ svcCtx *svc.ServiceContext }

func NewVaultServiceServer(svcCtx *svc.ServiceContext) *VaultServiceServer {
	return &VaultServiceServer{svcCtx: svcCtx}
}
func RegisterVaultServiceServer(s *grpc.Server, srv *VaultServiceServer) {
	logx.Info("vault service registered")
}

// ==================== GetVaults ====================

func (s *VaultServiceServer) GetVaults(ctx context.Context, page, pageSize int64) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	cacheKey := fmt.Sprintf("vault:list:page:%d", page)
	if c, err := s.svcCtx.Redis.Get(cacheKey); err == nil && c != "" {
		var r interface{}
		json.Unmarshal([]byte(c), &r)
		return r, nil
	}
	vaults, total, err := s.svcCtx.VaultModel.ListVaults(ctx, chainId, page, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]interface{}, 0, len(vaults))
	for _, v := range vaults {
		info := map[string]interface{}{
			"vaultAddress": v.VaultAddress, "name": v.Name, "symbol": v.Symbol,
			"assetSymbol": v.AssetSymbol, "strategyType": v.StrategyType,
			"tvlUsd": "0", "nav": "0", "apr7d": "0", "apr30d": "0",
		}
		snap, err := s.svcCtx.SnapshotModel.GetLatest(ctx, chainId, v.VaultAddress)
		if err == nil && snap != nil {
			info["tvlUsd"] = snap.TvlUsd
			info["nav"] = snap.Nav
			if snap.Apr7d.Valid {
				info["apr7d"] = snap.Apr7d.String
			}
			if snap.Apr30d.Valid {
				info["apr30d"] = snap.Apr30d.String
			}
		}
		list = append(list, info)
	}
	resp := map[string]interface{}{"list": list, "total": total}
	if d, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(d), 15)
	}
	return resp, nil
}

// ==================== GetVaultDetail ====================

func (s *VaultServiceServer) GetVaultDetail(ctx context.Context, vaultAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	vault, err := s.svcCtx.VaultModel.FindByAddress(ctx, chainId, vaultAddress)
	if err != nil {
		return nil, fmt.Errorf("vault not found: %w", err)
	}

	info := map[string]interface{}{
		"vaultAddress": vault.VaultAddress, "name": vault.Name, "symbol": vault.Symbol,
		"assetSymbol": vault.AssetSymbol, "strategyType": vault.StrategyType,
		"tvlUsd": "0", "nav": "0", "isActive": vault.IsActive,
	}

	snap, err := s.svcCtx.SnapshotModel.GetLatest(ctx, chainId, vaultAddress)
	if err == nil && snap != nil {
		info["tvlUsd"] = snap.TvlUsd
		info["nav"] = snap.Nav
	}

	// 总存入量
	var totalDeposited string
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CAST(amount AS DECIMAL(65,18))), 0) FROM vault_deposits
		 WHERE chain_id = ? AND vault_address = ?`,
		chainId, vaultAddress).Scan(&totalDeposited)

	// APY
	apy7d := "0"
	apy30d := "0"
	if snap != nil {
		if snap.Apr7d.Valid {
			apy7d = snap.Apr7d.String
		}
		if snap.Apr30d.Valid {
			apy30d = snap.Apr30d.String
		}
	}

	return map[string]interface{}{
		"vault":          info,
		"totalDeposited": totalDeposited,
		"performanceFee": "0.10",
		"managementFee":  "0.02",
		"apy7d":          apy7d,
		"apy30d":         apy30d,
	}, nil
}

// ==================== GetPositions ====================

func (s *VaultServiceServer) GetPositions(ctx context.Context, userAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	positions, err := s.svcCtx.PositionModel.ListByUser(ctx, chainId, userAddress)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]interface{}, 0, len(positions))
	for _, p := range positions {
		vault, _ := s.svcCtx.VaultModel.FindByAddress(ctx, chainId, p.VaultAddress)
		name := p.VaultAddress
		if vault != nil {
			name = vault.Name
		}
		list = append(list, map[string]interface{}{
			"vaultAddress": p.VaultAddress, "vaultName": name,
			"shares": p.Shares, "costBasis": p.CostBasis,
			"depositedTotal": p.DepositedTotal, "withdrawnTotal": p.WithdrawnTotal,
		})
	}
	return map[string]interface{}{"list": list}, nil
}

// ==================== Build* 交易构建 ====================

func (s *VaultServiceServer) buildVaultTx(method string, params map[string]interface{}) (interface{}, error) {
	data, _ := json.Marshal(map[string]interface{}{"method": method, "params": params})
	return map[string]interface{}{
		"to": params["vaultAddress"], "value": "0", "gasLimit": "300000",
		"data": string(data),
	}, nil
}

func (s *VaultServiceServer) BuildDeposit(ctx context.Context, userAddr, vaultAddr, amount string) (interface{}, error) {
	logx.Infof("build vault deposit: user=%s, vault=%s, amount=%s", userAddr, vaultAddr, amount)
	return s.buildVaultTx("deposit", map[string]interface{}{
		"vaultAddress": vaultAddr, "amount": amount, "receiver": userAddr,
	})
}

func (s *VaultServiceServer) BuildWithdraw(ctx context.Context, userAddr, vaultAddr, shares string) (interface{}, error) {
	logx.Infof("build vault withdraw: user=%s, vault=%s, shares=%s", userAddr, vaultAddr, shares)
	return s.buildVaultTx("withdraw", map[string]interface{}{
		"vaultAddress": vaultAddr, "shares": shares, "receiver": userAddr, "owner": userAddr,
	})
}
