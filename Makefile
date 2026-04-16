# Makefile — ReyFi Backend

# ==================== 开发环境 ====================
.PHONY: dev-up dev-down

dev-up:                              ## 启动开发依赖 (MySQL + Redis + Kafka)
	docker compose -f deploy/docker-compose.dev.yml up -d

dev-down:                            ## 停止开发依赖
	docker compose -f deploy/docker-compose.dev.yml down

# ==================== 代码生成 ====================
.PHONY: gen-api gen-rpc-dex gen-rpc-user gen-model-dex

gen-api:                             ## 生成 gateway API 代码
	cd app/gateway && goctl api go -api api/gateway.api -dir .

gen-rpc-dex:                         ## 生成 DEX RPC 代码
	cd app/dex && goctl rpc protoc rpc/dex.proto \
		--go_out=. --go-grpc_out=. --zrpc_out=.

gen-rpc-user:                        ## 生成 User RPC 代码
	cd app/user && goctl rpc protoc rpc/user.proto \
		--go_out=. --go-grpc_out=. --zrpc_out=.

gen-model-dex:                       ## 生成 DEX model 代码
	goctl model mysql datasource \
		-url="root:reyfi_dev_123@tcp(127.0.0.1:3306)/reyfi" \
		-table="dex_pairs,dex_trades,dex_pair_snapshots,dex_liquidity_events,dex_liquidity_positions,dex_pair_stats_daily" \
		-dir="app/dex/internal/model" -cache

# ==================== 运行服务 ====================
.PHONY: run-gateway run-dex run-user run-indexer

run-gateway:                         ## 启动 Gateway
	cd app/gateway && go run gateway.go -f etc/gateway-dev.yaml

run-dex:                             ## 启动 DEX RPC
	cd app/dex && go run dex.go -f etc/dex-dev.yaml

run-user:                            ## 启动 User RPC
	cd app/user && go run user.go -f etc/user-dev.yaml

run-indexer:                         ## 启动 Chain Indexer
	cd app/chain-indexer && go run indexer.go -f etc/indexer-dev.yaml

# ==================== 数据库 ====================
.PHONY: migrate migrate-down migrate-status

migrate:                             ## 执行数据库迁移
	migrate -path scripts/migration -database "mysql://root:reyfi_dev_123@tcp(127.0.0.1:3306)/reyfi" up

migrate-down:                        ## 回滚最近一次迁移
	migrate -path scripts/migration -database "mysql://root:reyfi_dev_123@tcp(127.0.0.1:3306)/reyfi" down 1

migrate-status:                      ## 查看迁移状态
	migrate -path scripts/migration -database "mysql://root:reyfi_dev_123@tcp(127.0.0.1:3306)/reyfi" version

# ==================== 测试 ====================
.PHONY: test test-cover lint build

test:                                ## 运行单元测试
	go test ./... -v -count=1

test-cover:                          ## 运行测试并生成覆盖率报告
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:                                ## 代码检查
	golangci-lint run ./...

build:                               ## 编译所有服务
	go build -o bin/gateway    ./app/gateway/gateway.go
	go build -o bin/dex        ./app/dex/dex.go
	go build -o bin/user       ./app/user/user.go
	go build -o bin/indexer    ./app/chain-indexer/indexer.go

# ==================== 帮助 ====================
.PHONY: help
help:                                ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
