#!/usr/bin/env bash
# E2E テスト共通関数

# カラー定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# APIエンドポイント
ORDER_API="http://localhost:8081"
INVENTORY_API="http://localhost:8082"
PAYMENT_API="http://localhost:8083"

# 注文作成
# 引数: customer_id, product_id, quantity, total_amount
create_order() {
    local customer_id=$1
    local product_id=$2
    local quantity=$3
    local total_amount=$4
    
    curl -s -X POST "$ORDER_API/orders" \
        -H "Content-Type: application/json" \
        -d "{\"customer_id\":\"$customer_id\",\"product_id\":\"$product_id\",\"quantity\":$quantity,\"total_amount\":$total_amount}"
}

# 注文状態取得
get_order_status() {
    local order_id=$1
    curl -s "$ORDER_API/orders/$order_id" | jq -r '.status // "NOT_FOUND"'
}

# 決済状態取得
get_payment_status() {
    local order_id=$1
    curl -s "$PAYMENT_API/payments/order/$order_id" | jq -r '.status // "NOT_FOUND"'
}

# 在庫取得
get_inventory() {
    local product_id=$1
    curl -s "$INVENTORY_API/inventory/$product_id"
}

# Saga処理完了待機
# 引数: 待機秒数（デフォルト10秒）
wait_for_saga() {
    local wait_time=${1:-10}
    echo "Saga処理待機中（${wait_time}秒）..."
    sleep "$wait_time"
}

# アサーション: 値が等しいことを確認
assert_equals() {
    local actual=$1
    local expected=$2
    local message=$3
    
    if [ "$actual" = "$expected" ]; then
        echo -e "${GREEN}✓ $message${NC}"
        return 0
    else
        echo -e "${RED}✗ $message${NC}"
        echo -e "${RED}  期待値: $expected${NC}"
        echo -e "${RED}  実際値: $actual${NC}"
        return 1
    fi
}

# アサーション: 値が空でないことを確認
assert_not_empty() {
    local value=$1
    local message=$2
    
    if [ -n "$value" ] && [ "$value" != "null" ]; then
        echo -e "${GREEN}✓ $message${NC}"
        return 0
    else
        echo -e "${RED}✗ $message${NC}"
        echo -e "${RED}  値が空または null です${NC}"
        return 1
    fi
}

# アサーション: 値が特定の値でないことを確認
assert_not_equals() {
    local actual=$1
    local not_expected=$2
    local message=$3
    
    if [ "$actual" != "$not_expected" ]; then
        echo -e "${GREEN}✓ $message${NC}"
        return 0
    else
        echo -e "${RED}✗ $message${NC}"
        echo -e "${RED}  値が $not_expected であってはいけません${NC}"
        return 1
    fi
}

