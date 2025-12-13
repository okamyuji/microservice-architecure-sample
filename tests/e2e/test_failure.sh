#!/usr/bin/env bash
# 異常系テスト
# 決済失敗 → 補償トランザクション のフロー確認

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "========================================"
echo "異常系テスト開始"
echo "========================================"

# テスト1: 決済失敗（金額が.99で終わる場合）
echo -e "\n[テスト1] 決済失敗シナリオ（金額 99.99）"
ORDER_RESPONSE=$(create_order "CUST-FAIL-001" "PROD-001" 2 99.99)
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
echo "注文ID: $ORDER_ID"

# 初期状態確認
INITIAL_STATUS=$(echo "$ORDER_RESPONSE" | jq -r '.status')
assert_equals "$INITIAL_STATUS" "PENDING" "初期状態がPENDINGであること"

# Saga処理（補償含む）待機
wait_for_saga 15

# 注文状態確認（CANCELLEDになるはず）
echo -e "\n[検証] 注文状態確認"
ORDER_STATUS=$(get_order_status "$ORDER_ID")
echo "注文状態: $ORDER_STATUS"
assert_equals "$ORDER_STATUS" "CANCELLED" "注文がCANCELLEDになること"

# 決済状態確認（FAILEDになるはず）
echo -e "\n[検証] 決済状態確認"
PAYMENT=$(curl -s "http://localhost:8083/payments/order/$ORDER_ID")
PAYMENT_STATUS=$(echo "$PAYMENT" | jq -r '.status')
FAILURE_REASON=$(echo "$PAYMENT" | jq -r '.failure_reason')
echo "決済状態: $PAYMENT_STATUS"
echo "失敗理由: $FAILURE_REASON"
assert_equals "$PAYMENT_STATUS" "FAILED" "決済がFAILEDになること"
assert_not_empty "$FAILURE_REASON" "失敗理由が記録されること"

# テスト2: 存在しない商品での注文
echo -e "\n[テスト2] 存在しない商品での注文"
ORDER_RESPONSE=$(create_order "CUST-FAIL-002" "PROD-NOTEXIST" 1 100.00)
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
echo "注文ID: $ORDER_ID"

wait_for_saga 15

ORDER_STATUS=$(get_order_status "$ORDER_ID")
echo "注文状態: $ORDER_STATUS"
assert_equals "$ORDER_STATUS" "CANCELLED" "存在しない商品の注文がCANCELLEDになること"

echo -e "\n${GREEN}異常系テスト完了${NC}"

