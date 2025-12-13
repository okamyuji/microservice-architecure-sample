#!/usr/bin/env bash
# E2E 結合テスト実行スクリプト
# 全てのテストシナリオを実行

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# カラー定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}E2E 結合テスト開始${NC}"
echo -e "${YELLOW}========================================${NC}"

# サービス起動確認
echo -e "\n${YELLOW}サービス起動確認中...${NC}"
for port in 8081 8082 8083; do
    if ! curl -s "http://localhost:$port/health" > /dev/null 2>&1; then
        echo -e "${RED}ポート $port のサービスが起動していません${NC}"
        echo -e "${YELLOW}make up を実行してサービスを起動してください${NC}"
        exit 1
    fi
done
echo -e "${GREEN}✓ 全サービス起動確認完了${NC}"

# 各テストスクリプト実行
TESTS=("test_success.sh" "test_failure.sh" "test_boundary.sh" "test_edge.sh")
PASSED=0
FAILED=0

for test in "${TESTS[@]}"; do
    echo -e "\n${YELLOW}----------------------------------------${NC}"
    echo -e "${YELLOW}実行中: $test${NC}"
    echo -e "${YELLOW}----------------------------------------${NC}"
    
    if "$SCRIPT_DIR/$test"; then
        echo -e "${GREEN}✓ $test 成功${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ $test 失敗${NC}"
        ((FAILED++))
    fi
done

# 結果サマリー
echo -e "\n${YELLOW}========================================${NC}"
echo -e "${YELLOW}テスト結果サマリー${NC}"
echo -e "${YELLOW}========================================${NC}"
echo -e "成功: ${GREEN}$PASSED${NC}"
echo -e "失敗: ${RED}$FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}全てのテストが成功しました${NC}"
    exit 0
else
    echo -e "\n${RED}一部のテストが失敗しました${NC}"
    exit 1
fi

