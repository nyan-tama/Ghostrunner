#!/bin/bash
set -euo pipefail

# テンプレートファイル静的検証スクリプト
# 計画書のテストケース 1-9 に対応

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PASS=0
FAIL=0

check() {
  local desc="$1"
  local result="$2"
  if [ "$result" = "0" ]; then
    echo "  OK: $desc"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $desc"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== テンプレートファイル静的検証 ==="
echo ""

# --- ケース1: base テンプレートファイル存在確認 (21ファイル) ---
echo "[1] base テンプレートファイル存在確認"
BASE_FILES=(
  "backend/cmd/server/main.go"
  "backend/internal/handler/health.go"
  "backend/internal/handler/hello.go"
  "backend/go.mod"
  "backend/Dockerfile"
  "backend/.env.example"
  "frontend/src/app/page.tsx"
  "frontend/src/app/layout.tsx"
  "frontend/src/app/globals.css"
  "frontend/package.json"
  "frontend/tsconfig.json"
  "frontend/next.config.ts"
  "frontend/postcss.config.mjs"
  "frontend/eslint.config.mjs"
  "frontend/vitest.config.ts"
  "frontend/vitest.setup.ts"
  "frontend/Dockerfile"
  "Makefile"
  "docker-compose.yml"
  "cloudbuild.yaml"
  ".gitignore"
)

base_missing=0
for f in "${BASE_FILES[@]}"; do
  if [ ! -f "$SCRIPT_DIR/base/$f" ]; then
    echo "    MISSING: base/$f"
    base_missing=1
  fi
done
check "base テンプレート 21ファイル全て存在" "$base_missing"

# --- ケース2: with-db テンプレートファイル存在確認 (7ファイル) ---
echo "[2] with-db テンプレートファイル存在確認"
WITHDB_FILES=(
  "backend/cmd/server/main.go"
  "backend/internal/handler/sample.go"
  "backend/internal/domain/model/sample.go"
  "backend/internal/infrastructure/database.go"
  "backend/go.mod"
  "docker-compose.yml"
  "db/init.sql"
)

withdb_missing=0
for f in "${WITHDB_FILES[@]}"; do
  if [ ! -f "$SCRIPT_DIR/with-db/$f" ]; then
    echo "    MISSING: with-db/$f"
    withdb_missing=1
  fi
done
check "with-db テンプレート 7ファイル全て存在" "$withdb_missing"

# --- ケース3: go.mod (base) 構文 ---
echo "[3] go.mod (base) 構文確認"
base_gomod="$SCRIPT_DIR/base/backend/go.mod"
r=0
grep -q 'module {{PROJECT_NAME}}/backend' "$base_gomod" || r=1
grep -q 'go 1.24' "$base_gomod" || r=1
check "base go.mod: module名とGoバージョン" "$r"

# --- ケース4: go.mod (with-db) 構文 ---
echo "[4] go.mod (with-db) 構文確認"
withdb_gomod="$SCRIPT_DIR/with-db/backend/go.mod"
r=0
grep -q 'module {{PROJECT_NAME}}/backend' "$withdb_gomod" || r=1
grep -q 'gorm.io/gorm' "$withdb_gomod" || r=1
grep -q 'gorm.io/driver/postgres' "$withdb_gomod" || r=1
check "with-db go.mod: GORM依存あり" "$r"

# --- ケース5: package.json JSON構文 ---
echo "[5] package.json JSON構文確認"
pkg="$SCRIPT_DIR/base/frontend/package.json"
r=0
python3 -m json.tool "$pkg" > /dev/null 2>&1 || r=1
check "package.json JSONパース成功" "$r"

# --- ケース6: package.json プレースホルダー ---
echo "[6] package.json プレースホルダー確認"
r=0
grep -q '"name": "{{PROJECT_NAME}}"' "$pkg" || r=1
check "package.json に {{PROJECT_NAME}} あり" "$r"

# --- ケース7: プレースホルダー網羅チェック ---
echo "[7] プレースホルダー網羅チェック"
PLACEHOLDER_FILES=(
  "base/backend/go.mod"
  "base/frontend/package.json"
  "base/frontend/src/app/layout.tsx"
  "base/cloudbuild.yaml"
  "base/docker-compose.yml"
  "base/Makefile"
)

placeholder_fail=0
for f in "${PLACEHOLDER_FILES[@]}"; do
  if ! grep -q '{{PROJECT_NAME}}' "$SCRIPT_DIR/$f"; then
    echo "    MISSING placeholder: $f"
    placeholder_fail=1
  fi
done
check "計画書記載の6ファイル全てに {{PROJECT_NAME}} あり" "$placeholder_fail"

# --- ケース8: with-db 上書き対象の整合性 ---
echo "[8] with-db 上書き対象の整合性確認"
OVERWRITE_FILES=(
  "backend/cmd/server/main.go"
  "backend/go.mod"
  "docker-compose.yml"
)

overwrite_fail=0
for f in "${OVERWRITE_FILES[@]}"; do
  if [ ! -f "$SCRIPT_DIR/base/$f" ]; then
    echo "    MISSING base counterpart: base/$f"
    overwrite_fail=1
  fi
  if [ ! -f "$SCRIPT_DIR/with-db/$f" ]; then
    echo "    MISSING with-db: with-db/$f"
    overwrite_fail=1
  fi
done
check "with-db 置換ファイルのbase側対応あり" "$overwrite_fail"

# --- ケース9: init.md の存在確認 ---
echo "[9] init.md の存在確認"
GHOSTRUNNER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
r=0
[ -f "$GHOSTRUNNER_ROOT/.claude/commands/init.md" ] || r=1
check "init.md 存在" "$r"

# --- 結果集計 ---
echo ""
echo "=== 結果 ==="
echo "  PASS: $PASS"
echo "  FAIL: $FAIL"
TOTAL=$((PASS + FAIL))
echo "  TOTAL: $TOTAL"

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "検証失敗があります。上記のFAILを確認してください。"
  exit 1
else
  echo ""
  echo "全検証項目がパスしました。"
  exit 0
fi
