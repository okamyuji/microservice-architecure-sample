.PHONY: all build up down logs test clean lint check setup-hooks e2e-test

# デフォルトターゲット
all: build up

# 全サービスをビルド
build:
	docker compose build

# 全サービスを起動
up:
	docker compose up -d

# 全サービスを停止
down:
	docker compose down

# ログ表示
logs:
	docker compose logs -f

# 各サービスのログ表示
logs-order:
	docker compose logs -f order-service

logs-inventory:
	docker compose logs -f inventory-service

logs-payment:
	docker compose logs -f payment-service

# ユニットテスト実行
test:
	go test -shuffle=on -count=1 -race -v ./...

# テストとカバレッジ出力
test-coverage:
	mkdir -p coverage
	go test -shuffle=on -count=1 -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
	go tool cover -func=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "カバレッジレポート: coverage/coverage.html"

# リンター実行
lint:
	golangci-lint run ./...

# 静的解析（staticcheck）
staticcheck:
	staticcheck ./...

# go vet 実行
vet:
	go vet ./...

# コードフォーマット
fmt:
	go fmt ./...

# 品質チェック（全て実行）
check:
	./scripts/check.sh

# Git フックセットアップ
setup-hooks:
	./scripts/setup-hooks.sh

# 依存関係ダウンロード
deps:
	go mod download
	go mod tidy

# クリーンアップ
clean:
	docker compose down -v
	rm -rf vendor/ coverage/

# ヘルスチェック
health:
	@echo "Order Service:"
	@curl -s http://localhost:8081/health | jq
	@echo "\nInventory Service:"
	@curl -s http://localhost:8082/health | jq
	@echo "\nPayment Service:"
	@curl -s http://localhost:8083/health | jq

# E2Eテスト（結合テスト）実行
e2e-test:
	./tests/e2e/run_all.sh

# 正常系テスト
e2e-success:
	./tests/e2e/test_success.sh

# 異常系テスト
e2e-failure:
	./tests/e2e/test_failure.sh

# 境界値テスト
e2e-boundary:
	./tests/e2e/test_boundary.sh

# エッジケーステスト
e2e-edge:
	./tests/e2e/test_edge.sh

# 在庫確認
check-inventory:
	curl -s http://localhost:8082/inventory/PROD-001 | jq

# 注文確認
check-order:
	@read -p "Order ID: " order_id; \
	curl -s http://localhost:8081/orders/$$order_id | jq

# 決済確認
check-payment:
	@read -p "Order ID: " order_id; \
	curl -s http://localhost:8083/payments/order/$$order_id | jq
