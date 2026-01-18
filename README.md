# Go Standards Checker

Go言語 API開発標準ドキュメントへの準拠をチェックする静的解析ツール

## 特徴

- 📝 **命名規則チェック**: パッケージ名、ファイル名、関数名、変数名
- 📏 **コード構造チェック**: 関数行数、ネストレベル、パラメータ数
- ⚠️ **エラーハンドリングチェック**: エラー無視、panic使用
- 🏗️ **ディレクトリ構成チェック**: 標準構成との比較
- 🏷️ **構造体タグチェック**: JSONタグ、バリデーションタグ
- 🔧 **カスタムルール**: YAMLで独自ルールを追加可能

## インストール

```bash
go install github.com/go-standards-checker@latest
```

または、ソースからビルド:

```bash
git clone https://github.com/your-org/go-standards-checker
cd go-standards-checker
go build -o go-standards-checker .
```

## 使い方

### 基本的な使い方

```bash
# カレントディレクトリをチェック
go-standards-checker

# 特定のディレクトリをチェック
go-standards-checker ./myproject

# または -t オプションで指定
go-standards-checker -t ./myproject
```

### 設定ファイルを使用

```bash
# 設定ファイルのテンプレートを生成
go-standards-checker -init

# カスタム設定ファイルを使用
go-standards-checker -c ./my-rules.yaml
```

### フィルタリング

```bash
# エラーのみ表示
go-standards-checker -s error

# 警告以上を表示
go-standards-checker -s warning

# すべて表示（デフォルト）
go-standards-checker -s info
```

### 出力形式

```bash
# テキスト形式（デフォルト）
go-standards-checker

# JSON形式
go-standards-checker -json
```

## 設定ファイル

プロジェクトルートに `go-standards.yaml` を配置すると自動で読み込みます。

```yaml
# go-standards.yaml

settings:
  exclude_patterns:
    - "*_test.go"
    - "vendor/*"
  min_severity: "info"

naming:
  enabled: true
  rules:
    package_name:
      enabled: true
      pattern: "^[a-z][a-z0-9]*$"
      severity: "error"
      message: "パッケージ名は小文字のみ"

structure:
  enabled: true
  rules:
    max_function_lines:
      enabled: true
      limit: 50
      severity: "warning"

error_handling:
  enabled: true
  rules:
    no_ignored_errors:
      enabled: true
      severity: "error"
      allowed_patterns:
        - "defer.*Close"

# カスタムルール
custom_rules:
  - name: "no_hardcoded_secrets"
    enabled: true
    severity: "error"
    pattern: '(?i)(password|secret)\s*=\s*"[^"]+"'
    message: "認証情報をハードコードしないでください"
```

## チェックカテゴリ

### 命名規則 (naming)

| ルール | 説明 | デフォルト重要度 |
|--------|------|-----------------|
| `package_name` | パッケージ名は小文字のみ | error |
| `file_name` | ファイル名はスネークケース | warning |
| `exported_names` | 公開シンボルはPascalCase | warning |
| `interface_name` | インタフェース名のサフィックス | info |
| `error_var` | センチネルエラーはErrプレフィックス | warning |

### コード構造 (structure)

| ルール | 説明 | デフォルト |
|--------|------|-----------|
| `max_function_lines` | 関数の最大行数 | 50行 |
| `max_nesting_level` | 最大ネストレベル | 3 |
| `max_parameters` | パラメータの最大数 | 5 |
| `max_return_values` | 戻り値の最大数 | 3 |

### エラーハンドリング (error_handling)

| ルール | 説明 | デフォルト重要度 |
|--------|------|-----------------|
| `no_ignored_errors` | エラー無視の禁止 | error |
| `no_panic` | panicの使用制限 | warning |

### ディレクトリ構成 (directory)

| ルール | 説明 |
|--------|------|
| `required_dirs` | 必須ディレクトリ（cmd, internal等） |
| `recommended_dirs` | 推奨ディレクトリ（handler, service等） |

### 構造体タグ (struct_tags)

| ルール | 説明 |
|--------|------|
| `json_tag` | JSONタグの命名規則（snake_case推奨） |
| `validation_tag` | Requestで終わる構造体にvalidateタグを要求 |

## カスタムルールの追加

正規表現ベースのカスタムルールを追加できます：

```yaml
custom_rules:
  # ハードコードされたポート番号の検出
  - name: "no_hardcoded_ports"
    enabled: true
    severity: "warning"
    pattern: ':\d{4,5}["\']'
    message: "ポート番号は環境変数から取得してください"
    exclude_files:
      - "*_test.go"
      - "config.go"

  # time.Sleepの使用警告
  - name: "no_time_sleep"
    enabled: true
    severity: "warning"
    pattern: 'time\.Sleep\('
    message: "time.Sleepの使用は避けてください"
    exclude_files:
      - "*_test.go"

  # 直接的なos.Exit
  - name: "no_os_exit"
    enabled: true
    severity: "warning"
    pattern: 'os\.Exit\('
    message: "os.Exitは避け、エラーを返却してください"
    exclude_files:
      - "main.go"
```

## CI/CDへの統合

### GitHub Actions

```yaml
# .github/workflows/lint.yml
name: Lint

on: [push, pull_request]

jobs:
  go-standards:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Install go-standards-checker
        run: go install github.com/go-standards-checker@latest
      
      - name: Run standards check
        run: go-standards-checker -s warning
```

### Makefile統合

```makefile
.PHONY: lint standards

# 標準チェック
standards:
	@echo "🔍 Running Go Standards Checker..."
	go-standards-checker -s warning

# 全リントツール実行
lint: standards
	golangci-lint run
```

## 終了コード

| コード | 意味 |
|--------|------|
| 0 | チェック成功（エラーなし） |
| 1 | チェック失敗（エラーあり） |

## 出力例

### テキスト形式

```
╔══════════════════════════════════════════════════════════════════════╗
║          Go Standards Checker - Compliance Report                    ║
╚══════════════════════════════════════════════════════════════════════╝

📁 Project: /path/to/project
📄 Files Checked: 15

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                              SUMMARY                                   
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔴 Errors:   2
🟡 Warnings: 5
🔵 Info:     3
📊 Total:    10 violations

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                             VIOLATIONS                                 
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📄 internal/service/user_service.go
────────────────────────────────────────────────────────────────────────
🔴 [no_ignored_errors] Line 45: エラーは必ず明示的にハンドリングしてください
   │ result, _ := someFunction()
   💡 Suggestion: エラーを適切にハンドリングしてください

🟡 [max_function_lines] Line 60: 関数 'ProcessData' は75行あります（上限: 50行）
   │ func ProcessData(ctx context.Context, data []byte) error {
   💡 Suggestion: 関数を分割してください
```

### JSON形式

```json
{
  "project_path": "/path/to/project",
  "total_files": 15,
  "violations": [
    {
      "file": "internal/service/user_service.go",
      "line": 45,
      "rule": "no_ignored_errors",
      "category": "error_handling",
      "severity": "error",
      "message": "エラーは必ず明示的にハンドリングしてください"
    }
  ],
  "summary": {
    "total_violations": 10,
    "by_severity": {"error": 2, "warning": 5, "info": 3}
  }
}
```

## ライセンス

MIT License
