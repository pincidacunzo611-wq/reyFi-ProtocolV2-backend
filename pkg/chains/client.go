// Package chains 封装链上交互工具。
// 提供 ETH RPC 客户端、合约调用、事件日志查询等能力。
package chains

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeromicro/go-zero/core/logx"
)

// ChainConfig 链配置
type ChainConfig struct {
	RpcUrl  string `json:"rpcUrl"`
	ChainId int64  `json:"chainId"`
	// 合约地址映射
	Contracts map[string]string `json:"contracts"`
}

// Client 链上 RPC 客户端封装
type Client struct {
	ethClient *ethclient.Client
	chainId   int64
	contracts map[string]common.Address
}

// NewClient 创建链上客户端
func NewClient(cfg ChainConfig) (*Client, error) {
	client, err := ethclient.Dial(cfg.RpcUrl)
	if err != nil {
		return nil, fmt.Errorf("connect to rpc node: %w", err)
	}

	contracts := make(map[string]common.Address)
	for name, addr := range cfg.Contracts {
		if !common.IsHexAddress(addr) {
			return nil, fmt.Errorf("invalid contract address for %s: %s", name, addr)
		}
		contracts[name] = common.HexToAddress(addr)
	}

	logx.Infof("chain client connected: chainId=%d, rpc=%s, contracts=%d",
		cfg.ChainId, cfg.RpcUrl, len(contracts))

	return &Client{
		ethClient: client,
		chainId:   cfg.ChainId,
		contracts: contracts,
	}, nil
}

// ChainId 返回链 ID
func (c *Client) ChainId() int64 {
	return c.chainId
}

// GetContract 获取合约地址
func (c *Client) GetContract(name string) (common.Address, bool) {
	addr, ok := c.contracts[name]
	return addr, ok
}

// LatestBlockNumber 获取链上最新区块号
func (c *Client) LatestBlockNumber(ctx context.Context) (int64, error) {
	num, err := c.ethClient.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("get latest block number: %w", err)
	}
	return int64(num), nil
}

// BlockByNumber 获取指定区块
func (c *Client) BlockByNumber(ctx context.Context, blockNum int64) (*types.Block, error) {
	block, err := c.ethClient.BlockByNumber(ctx, big.NewInt(blockNum))
	if err != nil {
		return nil, fmt.Errorf("get block %d: %w", blockNum, err)
	}
	return block, nil
}

// HeaderByNumber 获取区块头
func (c *Client) HeaderByNumber(ctx context.Context, blockNum int64) (*types.Header, error) {
	header, err := c.ethClient.HeaderByNumber(ctx, big.NewInt(blockNum))
	if err != nil {
		return nil, fmt.Errorf("get header %d: %w", blockNum, err)
	}
	return header, nil
}

// FilterLogs 按条件查询事件日志
func (c *Client) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	logs, err := c.ethClient.FilterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("filter logs: %w", err)
	}
	return logs, nil
}

// FilterLogsByBlockRange 按区块范围和合约地址查询事件日志
func (c *Client) FilterLogsByBlockRange(ctx context.Context, from, to int64, addresses []common.Address, topics [][]common.Hash) ([]types.Log, error) {
	fromBlock := big.NewInt(from)
	toBlock := big.NewInt(to)

	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: addresses,
		Topics:    topics,
	}

	return c.FilterLogs(ctx, query)
}

// GetAllContractAddresses 返回所有注册的合约地址列表
func (c *Client) GetAllContractAddresses() []common.Address {
	addresses := make([]common.Address, 0, len(c.contracts))
	for _, addr := range c.contracts {
		addresses = append(addresses, addr)
	}
	return addresses
}

// WaitForBlock 等待目标区块确认
func (c *Client) WaitForBlock(ctx context.Context, targetBlock int64, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timeout waiting for block %d", targetBlock)
		case <-ticker.C:
			latest, err := c.LatestBlockNumber(ctx)
			if err != nil {
				logx.Errorf("check block height: %v", err)
				continue
			}
			if latest >= targetBlock {
				return nil
			}
		}
	}
}

// Close 关闭客户端
func (c *Client) Close() {
	if c.ethClient != nil {
		c.ethClient.Close()
	}
}

// EthClient 返回底层 ethclient，供 ABI 绑定等高级使用
func (c *Client) EthClient() *ethclient.Client {
	return c.ethClient
}
