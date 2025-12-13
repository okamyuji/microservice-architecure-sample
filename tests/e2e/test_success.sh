#!/usr/bin/env bash
# 正常系テスト
# 注文作成 → 在庫予約 → 決済成功 → 注文完了 のフロー確認

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "========================================"
echo "正常系テスト開始"
echo "========================================"

# テスト1: 通常の注文（成功するはず）
echo -e "\n[テスト1] 通常の注文作成"
ORDER_RESPONSE=$(create_order "CUST-SUCCESS-001" "PROD-001" 1 50.00)
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
echo "注文ID: $ORDER_ID"
assert_not_empty "$ORDER_ID" "注文IDが取得できること"

# Saga処理待機
wait_for_saga

# 注文状態確認
echo -e "\n[検証] 注文状態確認"
ORDER_STATUS=$(get_order_status "$ORDER_ID")
echo "注文状態: $ORDER_STATUS"
assert_equals "$ORDER_STATUS" "COMPLETED" "注文がCOMPLETEDになること"

# 決済状態確認
echo -e "\n[検証] 決済状態確認"
PAYMENT_STATUS=$(get_payment_status "$ORDER_ID")
echo "決済状態: $PAYMENT_STATUS"
assert_equals "$PAYMENT_STATUS" "COMPLETED" "決済がCOMPLETEDになること"

# テスト2: 複数注文（連続）
echo -e "\n[テスト2] 連続注文テスト"
for i in 1 2 3; do
    ORDER_RESPONSE=$(create_order "CUST-MULTI-$i" "PROD-002" 1 100.00)
    ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
    echo "注文 $i: $ORDER_ID"
done

wait_for_saga 15

echo -e "\n${GREEN}正常系テスト完了${NC}"

