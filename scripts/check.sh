#!/usr/bin/env bash
# コミット前品質チェックスクリプト
# go fmt, go vet, staticcheck, golangci-lint, go test を実行

set -e

# カラー定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# プロジェクトルートへ移動
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}コミット前品質チェック開始${NC}"
echo -e "${YELLOW}========================================${NC}"

# go fmt
echo -e "\n${YELLOW}[1/5] go fmt 実行中...${NC}"
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}以下のファイルがフォーマットされていません:${NC}"
    echo "$UNFORMATTED"
    echo -e "${YELLOW}go fmt ./... を実行してフォーマットしてください${NC}"
    exit 1
fi
echo -e "${GREEN}✓ go fmt 完了${NC}"

# go vet
echo -e "\n${YELLOW}[2/5] go vet 実行中...${NC}"
if ! go vet ./...; then
    echo -e "${RED}✗ go vet でエラーが検出されました${NC}"
    exit 1
fi
echo -e "${GREEN}✓ go vet 完了${NC}"

# staticcheck
echo -e "\n${YELLOW}[3/5] staticcheck 実行中...${NC}"
if command -v staticcheck &> /dev/null; then
    if ! staticcheck ./...; then
        echo -e "${RED}✗ staticcheck でエラーが検出されました${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ staticcheck 完了${NC}"
else
    echo -e "${YELLOW}⚠ staticcheck がインストールされていません（スキップ）${NC}"
fi

# golangci-lint
echo -e "\n${YELLOW}[4/5] golangci-lint 実行中...${NC}"
if command -v golangci-lint &> /dev/null; then
    if ! golangci-lint run ./...; then
        echo -e "${RED}✗ golangci-lint でエラーが検出されました${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ golangci-lint 完了${NC}"
else
    echo -e "${YELLOW}⚠ golangci-lint がインストールされていません（スキップ）${NC}"
fi

# go test with coverage
echo -e "\n${YELLOW}[5/5] go test 実行中（カバレッジ出力）...${NC}"
COVERAGE_DIR="$PROJECT_ROOT/coverage"
mkdir -p "$COVERAGE_DIR"

if ! go test -shuffle=on -count=1 -race -coverprofile="$COVERAGE_DIR/coverage.out" -covermode=atomic ./...; then
    echo -e "${RED}✗ テストが失敗しました${NC}"
    exit 1
fi

# カバレッジレポート生成
go tool cover -func="$COVERAGE_DIR/coverage.out" | tail -1
go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"

echo -e "${GREEN}✓ テスト完了${NC}"
echo -e "${GREEN}カバレッジレポート: $COVERAGE_DIR/coverage.html${NC}"

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}すべてのチェックが成功しました${NC}"
echo -e "${GREEN}========================================${NC}"

