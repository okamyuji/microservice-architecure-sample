#!/usr/bin/env bash
# Git フックセットアップスクリプト
# pre-commit フックをローカルリポジトリに設定

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Git フックをセットアップ中..."

# .githooks ディレクトリをフックパスとして設定
git config core.hooksPath "$PROJECT_ROOT/.githooks"

# 実行権限を付与
chmod +x "$PROJECT_ROOT/.githooks/pre-commit"
chmod +x "$PROJECT_ROOT/scripts/check.sh"

echo "✓ Git フックのセットアップが完了しました"
echo "  pre-commit フックが有効になりました"

