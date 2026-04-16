// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type VaultGetPositionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVaultGetPositionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VaultGetPositionsLogic {
	return &VaultGetPositionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VaultGetPositionsLogic) VaultGetPositions() error {
	// todo: add your logic here and delete this line

	return nil
}
