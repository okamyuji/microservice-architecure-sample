#!/usr/bin/env bash
# エッジケーステスト
# 特殊な状況でのテスト

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "========================================"
echo "エッジケーステスト開始"
echo "========================================"

# テスト1: 同一顧客からの連続注文
echo -e "\n[テスト1] 同一顧客からの連続注文"
CUSTOMER_ID="CUST-EDGE-SAME"
ORDER_IDS=()
for i in 1 2 3; do
    ORDER_RESPONSE=$(create_order "$CUSTOMER_ID" "PROD-001" 1 "$((i * 10)).00")
    ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
    ORDER_IDS+=("$ORDER_ID")
    echo "注文 $i: $ORDER_ID"
done

wait_for_saga 15

# 全注文の状態確認
for ORDER_ID in "${ORDER_IDS[@]}"; do
    STATUS=$(get_order_status "$ORDER_ID")
    echo "注文 $ORDER_ID: $STATUS"
done

# テスト2: 存在しない注文の取得
echo -e "\n[テスト2] 存在しない注文の取得"
RESPONSE=$(curl -s "http://localhost:8081/orders/00000000-0000-0000-0000-000000000000")
ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
if [ -n "$ERROR" ]; then
    echo "期待通り404エラー: $ERROR"
else
    echo "警告: 存在しない注文でエラーにならなかった"
fi

# テスト3: 空のリクエストボディ
echo -e "\n[テスト3] 空のリクエストボディ"
RESPONSE=$(curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d '{}')
ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
if [ -n "$ERROR" ]; then
    echo "期待通りエラー: $ERROR"
else
    echo "警告: 空リクエストでエラーにならなかった"
fi

# テスト4: 不正なJSON
echo -e "\n[テスト4] 不正なJSON"
RESPONSE=$(curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d 'invalid json')
ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
if [ -n "$ERROR" ]; then
    echo "期待通りエラー: $ERROR"
else
    echo "警告: 不正なJSONでエラーにならなかった"
fi

# テスト5: 在庫不足での注文
echo -e "\n[テスト5] 在庫不足での注文"
# 非常に大きな数量で注文
ORDER_RESPONSE=$(create_order "CUST-EDGE-STOCK" "PROD-001" 99999 1000000.00)
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
echo "注文ID: $ORDER_ID"

wait_for_saga 15

ORDER_STATUS=$(get_order_status "$ORDER_ID")
echo "注文状態: $ORDER_STATUS"
# 在庫不足でCANCELLEDになるはず
assert_equals "$ORDER_STATUS" "CANCELLED" "在庫不足でCANCELLEDになること"

# テスト6: 冪等性テスト（同じイベントの重複処理）
echo -e "\n[テスト6] 通常注文後の状態確認（冪等性確認用）"
ORDER_RESPONSE=$(create_order "CUST-EDGE-IDEMPOTENT" "PROD-002" 1 50.00)
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.order_id')
echo "注文ID: $ORDER_ID"

wait_for_saga

# 複数回状態を取得しても一貫していること
STATUS1=$(get_order_status "$ORDER_ID")
STATUS2=$(get_order_status "$ORDER_ID")
STATUS3=$(get_order_status "$ORDER_ID")

echo "状態確認1: $STATUS1"
echo "状態確認2: $STATUS2"
echo "状態確認3: $STATUS3"

if [ "$STATUS1" = "$STATUS2" ] && [ "$STATUS2" = "$STATUS3" ]; then
    echo "✓ 状態が一貫している"
else
    echo "✗ 状態が不一致"
fi

echo -e "\n${GREEN}エッジケーステスト完了${NC}"

