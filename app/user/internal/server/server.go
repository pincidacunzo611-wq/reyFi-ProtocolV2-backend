// Package server User gRPC 服务实现
package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/reyfi/reyfi-backend/app/user/internal/svc"
	"github.com/reyfi/reyfi-backend/pkg/errorx"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

type UserServiceServer struct {
	svcCtx *svc.ServiceContext
}

func NewUserServiceServer(svcCtx *svc.ServiceContext) *UserServiceServer {
	return &UserServiceServer{svcCtx: svcCtx}
}

func RegisterUserServiceServer(s *grpc.Server, srv *UserServiceServer) {
	logx.Info("user service registered")
}

// ==================== Auth: GetNonce ====================

type GetNonceReq struct {
	Address string `json:"address"`
}

type GetNonceResp struct {
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
	ExpiresIn int64  `json:"expiresIn"`
}

func (s *UserServiceServer) GetNonce(ctx context.Context, req *GetNonceReq) (*GetNonceResp, error) {
	addr := strings.ToLower(req.Address)
	if !common.IsHexAddress(addr) {
		return nil, errorx.ErrAddressInvalid
	}

	// 生成随机 nonce
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, errorx.ErrSystemBusy
	}
	nonceStr := hex.EncodeToString(nonceBytes)

	// 构造签名消息
	message := fmt.Sprintf("Sign this message to login to ReyFi:\n\nNonce: %s\nTimestamp: %s",
		nonceStr, time.Now().UTC().Format(time.RFC3339))

	expiresAt := time.Now().UTC().Add(5 * time.Minute)

	// 保存 nonce 到数据库
	_, err := s.svcCtx.DB.ExecContext(ctx,
		`INSERT INTO user_nonces (wallet_address, nonce, expires_at) VALUES (?, ?, ?)`,
		addr, nonceStr, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("save nonce: %w", err)
	}

	return &GetNonceResp{
		Address:   addr,
		Nonce:     message,
		ExpiresIn: 300,
	}, nil
}

// ==================== Auth: Login ====================

type LoginReq struct {
	Address   string `json:"address"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

type LoginResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

func (s *UserServiceServer) Login(ctx context.Context, req *LoginReq) (*LoginResp, error) {
	addr := strings.ToLower(req.Address)

	// 1. 验证签名
	recoveredAddr, err := recoverAddress(req.Message, req.Signature)
	if err != nil {
		return nil, errorx.ErrSignatureInvalid
	}

	if strings.ToLower(recoveredAddr) != addr {
		return nil, errorx.ErrSignatureInvalid
	}

	// 2. 检查 nonce 是否有效
	var nonceId int64
	err = s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT id FROM user_nonces 
		 WHERE wallet_address = ? AND is_used = 0 AND expires_at > NOW()
		 ORDER BY id DESC LIMIT 1`,
		addr).Scan(&nonceId)
	if err == sql.ErrNoRows {
		return nil, errorx.New(errorx.CodeSignatureInvalid, "Nonce 已过期或不存在")
	}
	if err != nil {
		return nil, err
	}

	// 3. 标记 nonce 已使用
	s.svcCtx.DB.ExecContext(ctx,
		`UPDATE user_nonces SET is_used = 1 WHERE id = ?`, nonceId)

	// 4. 创建或获取用户
	userId, err := s.getOrCreateUser(ctx, addr)
	if err != nil {
		return nil, err
	}

	// 5. 生成 JWT
	jwtCfg := middleware.JwtAuthConfig{
		AccessSecret:  s.svcCtx.Config.JwtAuth.AccessSecret,
		AccessExpire:  s.svcCtx.Config.JwtAuth.AccessExpire,
		RefreshExpire: s.svcCtx.Config.JwtAuth.RefreshExpire,
	}
	accessToken, refreshToken, err := middleware.GenerateToken(jwtCfg, userId, addr)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 6. 保存会话
	expiresAt := time.Now().UTC().Add(time.Duration(jwtCfg.RefreshExpire) * time.Second)
	s.svcCtx.DB.ExecContext(ctx,
		`INSERT INTO user_sessions (user_id, wallet_address, refresh_token, expires_at) 
		 VALUES (?, ?, ?, ?)`,
		userId, addr, refreshToken, expiresAt)

	// 7. 更新最后登录时间
	s.svcCtx.DB.ExecContext(ctx,
		`UPDATE users SET last_login_at = NOW() WHERE id = ?`, userId)

	logx.Infof("user logged in: %s (id=%d)", addr, userId)

	return &LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    jwtCfg.AccessExpire,
	}, nil
}

// ==================== Portfolio ====================

type ModuleAllocation struct {
	Module   string `json:"module"`
	Label    string `json:"label"`
	ValueUsd string `json:"valueUsd"`
	Percent  string `json:"percent"`
}

type RiskSummary struct {
	LendingHealthFactor string `json:"lendingHealthFactor"`
	FuturesMarginRatio  string `json:"futuresMarginRatio"`
	RiskLevel           string `json:"riskLevel"`
}

type GetPortfolioReq struct {
	WalletAddress string `json:"walletAddress"`
}

type GetPortfolioResp struct {
	WalletAddress  string              `json:"walletAddress"`
	TotalAssetUsd  string              `json:"totalAssetUsd"`
	TotalDebtUsd   string              `json:"totalDebtUsd"`
	NetValueUsd    string              `json:"netValueUsd"`
	Pnl24h         string              `json:"pnl24h"`
	Pnl24hPercent  string              `json:"pnl24hPercent"`
	Allocation     []*ModuleAllocation `json:"allocation"`
	RiskSummary    *RiskSummary        `json:"riskSummary"`
}

func (s *UserServiceServer) GetPortfolio(ctx context.Context, req *GetPortfolioReq) (*GetPortfolioResp, error) {
	addr := strings.ToLower(req.WalletAddress)

	// 先查缓存
	cacheKey := "user:" + addr + ":portfolio"
	if cached, err := s.svcCtx.Redis.Get(cacheKey); err == nil && cached != "" {
		// 反序列化并返回
		logx.Debugf("portfolio cache hit: %s", addr)
	}

	// 并行查询各模块资产（Phase 1 只查 DEX）
	// TODO: 接入 Lending, Futures, Options, Vault, Bonds, Governance RPC
	resp := &GetPortfolioResp{
		WalletAddress: addr,
		TotalAssetUsd: "0",
		TotalDebtUsd:  "0",
		NetValueUsd:   "0",
		Pnl24h:        "0",
		Pnl24hPercent: "0",
		Allocation: []*ModuleAllocation{
			{Module: "dex", Label: "流动性池", ValueUsd: "0", Percent: "0"},
			{Module: "lending", Label: "借贷", ValueUsd: "0", Percent: "0"},
			{Module: "futures", Label: "永续", ValueUsd: "0", Percent: "0"},
			{Module: "options", Label: "期权", ValueUsd: "0", Percent: "0"},
			{Module: "vault", Label: "金库", ValueUsd: "0", Percent: "0"},
			{Module: "bonds", Label: "债券", ValueUsd: "0", Percent: "0"},
			{Module: "governance", Label: "治理", ValueUsd: "0", Percent: "0"},
		},
		RiskSummary: &RiskSummary{
			LendingHealthFactor: "999999.99",
			FuturesMarginRatio:  "1.00",
			RiskLevel:           "low",
		},
	}

	// 缓存 15 秒（序列化实际数据）
	if data, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(data), 15)
	}

	return resp, nil
}

// ==================== Activity ====================

type GetActivityReq struct {
	WalletAddress string `json:"walletAddress"`
	Module        string `json:"module"`
	Page          int64  `json:"page"`
	PageSize      int64  `json:"pageSize"`
}

type ActivityItem struct {
	Id        int64  `json:"id"`
	Module    string `json:"module"`
	Action    string `json:"action"`
	Summary   string `json:"summary"`
	AmountUsd string `json:"amountUsd"`
	TxHash    string `json:"txHash"`
	BlockTime string `json:"blockTime"`
}

type GetActivityResp struct {
	List  []*ActivityItem `json:"list"`
	Total int64           `json:"total"`
}

func (s *UserServiceServer) GetActivity(ctx context.Context, req *GetActivityReq) (*GetActivityResp, error) {
	addr := strings.ToLower(req.WalletAddress)

	where := "wallet_address = ?"
	args := []interface{}{addr}

	if req.Module != "" {
		where += " AND module = ?"
		args = append(args, req.Module)
	}

	var total int64
	s.svcCtx.DB.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM user_activity_stream WHERE %s", where),
		args...).Scan(&total)

	offset := (req.Page - 1) * req.PageSize
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, module, action, summary, COALESCE(amount_usd, 0), 
		 COALESCE(tx_hash, ''), block_time 
		 FROM user_activity_stream WHERE %s ORDER BY block_time DESC LIMIT ? OFFSET ?`, where),
		append(args, req.PageSize, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ActivityItem
	for rows.Next() {
		item := &ActivityItem{}
		var blockTime time.Time
		if err := rows.Scan(&item.Id, &item.Module, &item.Action, &item.Summary,
			&item.AmountUsd, &item.TxHash, &blockTime); err != nil {
			return nil, err
		}
		item.BlockTime = blockTime.UTC().Format(time.RFC3339)
		list = append(list, item)
	}

	return &GetActivityResp{List: list, Total: total}, nil
}

// ==================== RefreshToken ====================

type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshTokenResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

func (s *UserServiceServer) RefreshToken(ctx context.Context, req *RefreshTokenReq) (*RefreshTokenResp, error) {
	// 1. 验证 refresh token 是否存在且有效
	var userId int64
	var walletAddress string
	err := s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT user_id, wallet_address FROM user_sessions 
		 WHERE refresh_token = ? AND expires_at > NOW() AND is_revoked = 0
		 ORDER BY id DESC LIMIT 1`,
		req.RefreshToken).Scan(&userId, &walletAddress)
	if err != nil {
		return nil, errorx.New(errorx.CodeTokenExpired, "refresh token 已过期或无效")
	}

	// 2. 撤销旧 token
	s.svcCtx.DB.ExecContext(ctx,
		`UPDATE user_sessions SET is_revoked = 1 WHERE refresh_token = ?`, req.RefreshToken)

	// 3. 生成新 token 对
	jwtCfg := middleware.JwtAuthConfig{
		AccessSecret:  s.svcCtx.Config.JwtAuth.AccessSecret,
		AccessExpire:  s.svcCtx.Config.JwtAuth.AccessExpire,
		RefreshExpire: s.svcCtx.Config.JwtAuth.RefreshExpire,
	}
	newAccess, newRefresh, err := middleware.GenerateToken(jwtCfg, userId, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 4. 保存新会话
	expiresAt := time.Now().UTC().Add(time.Duration(jwtCfg.RefreshExpire) * time.Second)
	s.svcCtx.DB.ExecContext(ctx,
		`INSERT INTO user_sessions (user_id, wallet_address, refresh_token, expires_at)
		 VALUES (?, ?, ?, ?)`,
		userId, walletAddress, newRefresh, expiresAt)

	logx.Infof("token refreshed for user %s (id=%d)", walletAddress, userId)

	return &RefreshTokenResp{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
		ExpiresIn:    jwtCfg.AccessExpire,
	}, nil
}

// ==================== UpdateSettings ====================

type UpdateSettingsReq struct {
	WalletAddress   string `json:"walletAddress"`
	Nickname        string `json:"nickname"`
	Language        string `json:"language"`
	Currency        string `json:"currency"`
	SlippageDefault int    `json:"slippageDefault"`
}

type UpdateSettingsResp struct {
	Success bool `json:"success"`
}

func (s *UserServiceServer) UpdateSettings(ctx context.Context, req *UpdateSettingsReq) (*UpdateSettingsResp, error) {
	addr := strings.ToLower(req.WalletAddress)

	// 更新用户设置（使用 UPSERT 模式）
	_, err := s.svcCtx.DB.ExecContext(ctx,
		`UPDATE users SET 
			nickname = COALESCE(NULLIF(?, ''), nickname),
			language = COALESCE(NULLIF(?, ''), language),
			currency = COALESCE(NULLIF(?, ''), currency),
			slippage_default = CASE WHEN ? > 0 THEN ? ELSE slippage_default END,
			updated_at = NOW()
		 WHERE wallet_address = ?`,
		req.Nickname, req.Language, req.Currency,
		req.SlippageDefault, req.SlippageDefault, addr)
	if err != nil {
		return nil, fmt.Errorf("update settings: %w", err)
	}

	// 清除用户缓存
	s.svcCtx.Redis.Del("user:" + addr + ":settings")

	logx.Infof("settings updated for %s", addr)
	return &UpdateSettingsResp{Success: true}, nil
}

// ==================== 辅助函数 ====================

// getOrCreateUser 获取或创建用户
func (s *UserServiceServer) getOrCreateUser(ctx context.Context, addr string) (int64, error) {
	var userId int64
	err := s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT id FROM users WHERE wallet_address = ?`, addr).Scan(&userId)

	if err == sql.ErrNoRows {
		result, err := s.svcCtx.DB.ExecContext(ctx,
			`INSERT INTO users (wallet_address, status) VALUES (?, 'active')`, addr)
		if err != nil {
			return 0, err
		}
		userId, _ = result.LastInsertId()
		logx.Infof("new user created: %s (id=%d)", addr, userId)
		return userId, nil
	}
	if err != nil {
		return 0, err
	}
	return userId, nil
}

// recoverAddress 从消息和签名中恢复以太坊地址
func recoverAddress(message, sigHex string) (string, error) {
	// 以太坊签名消息前缀
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	prefixedMsg := []byte(prefix + message)
	msgHash := crypto.Keccak256Hash(prefixedMsg)

	// 解码签名
	sig, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("decode signature: %w", err)
	}

	if len(sig) != 65 {
		return "", fmt.Errorf("invalid signature length: %d", len(sig))
	}

	// EIP-155: v 值修正
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	pubKey, err := crypto.SigToPub(msgHash.Bytes(), sig)
	if err != nil {
		return "", fmt.Errorf("recover public key: %w", err)
	}

	addr := crypto.PubkeyToAddress(*pubKey)
	return addr.Hex(), nil
}
