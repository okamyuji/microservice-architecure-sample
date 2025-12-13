#!/usr/bin/env bash
# 境界値テスト
# 数量・金額の境界値でのテスト

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "========================================"
echo "境界値テスト開始"
echo "========================================"

# テスト1: 最小数量（1個）
echo -e "\n[テスト1] 最小数量（1個）での注文"
ORDER_RESPONSE=$(create_order "CUST-BOUNDARY-001" "PROD-001" 1 10.00)
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
echo "注文ID: $ORDER_ID"
assert_not_empty "$ORDER_ID" "最小数量で注文が作成されること"

wait_for_saga

ORDER_STATUS=$(get_order_status "$ORDER_ID")
echo "注文状態: $ORDER_STATUS"

# テスト2: 大量注文（在庫上限に近い）
echo -e "\n[テスト2] 大量注文テスト"
# 現在の在庫確認
INVENTORY=$(curl -s "http://localhost:8082/inventory/PROD-003")
AVAILABLE=$(echo "$INVENTORY" | jq -r '.available_quantity')
echo "PROD-003 の利用可能在庫: $AVAILABLE"

if [ "$AVAILABLE" -gt 10 ]; then
    ORDER_RESPONSE=$(create_order "CUST-BOUNDARY-002" "PROD-003" 10 1000.00)
    ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
    echo "注文ID: $ORDER_ID"
    assert_not_empty "$ORDER_ID" "大量注文が作成されること"
fi

# テスト3: 最小金額（0.01）
echo -e "\n[テスト3] 最小金額での注文"
ORDER_RESPONSE=$(create_order "CUST-BOUNDARY-003" "PROD-001" 1 0.01)
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
echo "注文ID: $ORDER_ID"
assert_not_empty "$ORDER_ID" "最小金額で注文が作成されること"

# テスト4: ゼロ数量（エラーになるはず）
echo -e "\n[テスト4] ゼロ数量での注文（エラー期待）"
RESPONSE=$(curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d '{"customer_id":"CUST-BOUNDARY-004","product_id":"PROD-001","quantity":0,"total_amount":100.00}')
ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
if [ -n "$ERROR" ]; then
    echo "期待通りエラー: $ERROR"
else
    echo "警告: ゼロ数量でエラーにならなかった"
fi

# テスト5: 負の金額（エラーになるはず）
echo -e "\n[テスト5] 負の金額での注文（エラー期待）"
RESPONSE=$(curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d '{"customer_id":"CUST-BOUNDARY-005","product_id":"PROD-001","quantity":1,"total_amount":-100.00}')
ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
if [ -n "$ERROR" ]; then
    echo "期待通りエラー: $ERROR"
else
    echo "警告: 負の金額でエラーにならなかった"
fi

echo -e "\n${GREEN}境界値テスト完了${NC}"

