package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-standards-checker/checker"
	"github.com/go-standards-checker/rules"
)

const version = "1.0.0"

func main() {
	// コマンドライン引数
	var (
		configPath  string
		targetDir   string
		outputJSON  bool
		minSeverity string
		showVersion bool
		initConfig  bool
	)

	flag.StringVar(&configPath, "config", "", "設定ファイルのパス (デフォルト: ./go-standards.yaml)")
	flag.StringVar(&configPath, "c", "", "設定ファイルのパス (短縮形)")
	flag.StringVar(&targetDir, "target", ".", "チェック対象ディレクトリ")
	flag.StringVar(&targetDir, "t", ".", "チェック対象ディレクトリ (短縮形)")
	flag.BoolVar(&outputJSON, "json", false, "JSON形式で出力")
	flag.StringVar(&minSeverity, "severity", "info", "最小重要度フィルター (error, warning, info)")
	flag.StringVar(&minSeverity, "s", "info", "最小重要度フィルター (短縮形)")
	flag.BoolVar(&showVersion, "version", false, "バージョン表示")
	flag.BoolVar(&showVersion, "v", false, "バージョン表示 (短縮形)")
	flag.BoolVar(&initConfig, "init", false, "設定ファイルのテンプレートを生成")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Go Standards Checker v%s
Go言語API開発標準ドキュメントへの準拠をチェックするツール

Usage:
  go-standards-checker [options] [target-directory]

Options:
`, version)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # カレントディレクトリをチェック
  go-standards-checker

  # 特定ディレクトリをチェック
  go-standards-checker -t ./myproject

  # カスタム設定ファイルを使用
  go-standards-checker -c ./my-rules.yaml

  # エラーのみ表示
  go-standards-checker -s error

  # JSON形式で出力
  go-standards-checker -json

  # 設定ファイルのテンプレートを生成
  go-standards-checker -init

Categories:
  - naming:         命名規則
  - structure:      コード構造（行数、ネスト等）
  - error_handling: エラーハンドリング
  - logging:        ログ出力
  - directory:      ディレクトリ構成
  - struct_tags:    構造体タグ
  - architecture:   レイヤーアーキテクチャ
  - custom:         カスタムルール

Severity Levels:
  - error:   修正必須
  - warning: 修正推奨
  - info:    情報
`)
	}

	flag.Parse()

	// バージョン表示
	if showVersion {
		fmt.Printf("go-standards-checker v%s\n", version)
		os.Exit(0)
	}

	// 設定ファイルテンプレート生成
	if initConfig {
		generateConfigTemplate()
		os.Exit(0)
	}

	// 位置引数があればターゲットディレクトリとして使用
	if flag.NArg() > 0 {
		targetDir = flag.Arg(0)
	}

	// 設定読み込み
	var cfg *rules.Config
	var err error

	if configPath != "" {
		cfg, err = rules.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: 設定ファイルの読み込みに失敗しました: %v\n", err)
			os.Exit(1)
		}
	} else {
		// デフォルト設定ファイルを探す
		defaultPaths := []string{
			"go-standards.yaml",
			"go-standards.yml",
			".go-standards.yaml",
			".go-standards.yml",
		}

		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				cfg, err = rules.LoadConfig(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %s の読み込みに失敗しました: %v\n", path, err)
				} else {
					fmt.Printf("📋 Using config: %s\n", path)
					break
				}
			}
		}

		// 設定ファイルが見つからない場合はデフォルト設定
		if cfg == nil {
			cfg = rules.DefaultConfig()
			fmt.Println("📋 Using default configuration")
		}
	}

	// 重要度フィルターをコマンドラインから上書き
	if minSeverity != "" {
		cfg.Settings.MinSeverity = minSeverity
	}

	// JSON出力設定
	if outputJSON {
		cfg.Settings.ReportFormat = "json"
	}

	// ターゲットディレクトリを絶対パスに
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: ターゲットディレクトリの解決に失敗しました: %v\n", err)
		os.Exit(1)
	}

	// ディレクトリ存在確認
	if info, err := os.Stat(absTargetDir); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: ディレクトリが見つかりません: %s\n", absTargetDir)
		os.Exit(1)
	}

	// チェック実行
	fmt.Printf("🔍 Checking: %s\n\n", absTargetDir)

	c := checker.NewChecker(cfg)
	report, err := c.Check(absTargetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: チェックに失敗しました: %v\n", err)
		os.Exit(1)
	}

	// 重要度フィルタリング
	filteredReport := report.Filter(rules.ParseSeverity(cfg.Settings.MinSeverity))

	// レポート出力
	if cfg.Settings.ReportFormat == "json" {
		output, err := filteredReport.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: JSON出力に失敗しました: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(output)
	} else {
		fmt.Print(filteredReport.ToText())
	}

	// 終了コード
	os.Exit(filteredReport.ExitCode())
}

// generateConfigTemplate 設定ファイルテンプレートを生成
func generateConfigTemplate() {
	template := `# Go Standards Checker 設定ファイル
# このファイルをプロジェクトルートに配置してください

# ========================================
# 基本設定
# ========================================
settings:
  # 除外パターン
  exclude_patterns:
    - "*_test.go"      # テストファイル
    - "vendor/*"       # vendorディレクトリ
    - ".git/*"         # gitディレクトリ
    - "*.pb.go"        # Protocol Buffers生成ファイル
  # レポート形式: text, json
  report_format: "text"
  # 最小重要度: error, warning, info
  min_severity: "info"

# ========================================
# 命名規則チェック
# ========================================
naming:
  enabled: true
  rules:
    package_name:
      enabled: true
      pattern: "^[a-z][a-z0-9]*$"
      severity: "error"
      message: "パッケージ名は小文字のみで構成してください"
    
    file_name:
      enabled: true
      pattern: "^[a-z][a-z0-9_]*\\.go$"
      severity: "warning"
      message: "ファイル名はスネークケース小文字で命名してください"
    
    exported_names:
      enabled: true
      severity: "warning"
      message: "公開シンボルはPascalCaseで命名してください"
    
    interface_name:
      enabled: true
      suffixes: ["er", "or", "Repository", "Service", "Client", "Handler"]
      severity: "info"
      message: "インタフェース名は標準的なサフィックスを使用してください"
    
    error_var:
      enabled: true
      pattern: "^Err[A-Z]"
      severity: "warning"
      message: "センチネルエラーはErrプレフィックスで定義してください"

# ========================================
# コード構造チェック
# ========================================
structure:
  enabled: true
  rules:
    max_function_lines:
      enabled: true
      limit: 50
      severity: "warning"
      message: "関数は50行以内を目安にしてください"
    
    max_nesting_level:
      enabled: true
      limit: 3
      severity: "warning"
      message: "ネストは3レベル以内を目安にしてください"
    
    max_parameters:
      enabled: true
      limit: 5
      severity: "info"
      message: "関数のパラメータは5個以内を目安にしてください"
    
    max_return_values:
      enabled: true
      limit: 3
      severity: "info"
      message: "関数の戻り値は3個以内を目安にしてください"

# ========================================
# エラーハンドリングチェック
# ========================================
error_handling:
  enabled: true
  rules:
    no_ignored_errors:
      enabled: true
      severity: "error"
      message: "エラーは必ず明示的にハンドリングしてください"
      allowed_patterns:
        - "defer.*Close"
        - "fmt\\.Print"
    
    no_panic:
      enabled: true
      severity: "warning"
      message: "panicの使用は避け、エラーを返却してください"
      allowed_in:
        - "main.go"
        - "*_test.go"

# ========================================
# ログ出力チェック
# ========================================
logging:
  enabled: true
  rules:
    no_fmt_println:
      enabled: true
      severity: "warning"
      message: "本番コードでfmt.Printlnは使用せず、適切なログライブラリを使用してください"

# ========================================
# ディレクトリ構成チェック
# ========================================
directory:
  enabled: true
  rules:
    required_dirs:
      enabled: true
      severity: "info"
      dirs:
        - "cmd"
        - "internal"
      message: "標準ディレクトリ構成を使用してください"
    
    recommended_dirs:
      enabled: false
      severity: "info"
      dirs:
        - "internal/handler"
        - "internal/service"
        - "internal/repository"
      message: "レイヤードアーキテクチャに基づくディレクトリ構成を推奨します"

# ========================================
# 構造体タグチェック
# ========================================
struct_tags:
  enabled: true
  rules:
    json_tag:
      enabled: true
      style: "snake_case"
      severity: "warning"
      message: "JSONタグはスネークケースで記述してください"
    
    validation_tag:
      enabled: true
      severity: "info"
      required_for:
        - "*Request"
        - "*Input"
      message: "リクエスト構造体にはvalidateタグを付与してください"

# ========================================
# カスタムルール（正規表現ベース）
# ========================================
custom_rules:
  # ハードコードされた認証情報の検出
  - name: "no_hardcoded_secrets"
    enabled: true
    severity: "error"
    pattern: '(?i)(password|secret|api_key)\s*[:=]\s*["\'][^"\']{8,}["\']'
    message: "認証情報をハードコードしないでください"
    exclude_files:
      - "*_test.go"
  
  # TODO/FIXMEの形式チェック
  - name: "todo_format"
    enabled: true
    severity: "info"
    pattern: '(TODO|FIXME)(?!\([a-zA-Z]+\))'
    message: "TODO/FIXMEには担当者を記載してください"
    exclude_files: []

# ========================================
# プロジェクト固有ルール
# ========================================
# ここに独自ルールを追加してください
project_rules: []
`

	filename := "go-standards.yaml"
	if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: 設定ファイルの生成に失敗しました: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 設定ファイルを生成しました: %s\n", filename)
	fmt.Println("\n次のステップ:")
	fmt.Println("  1. go-standards.yaml をプロジェクトに合わせてカスタマイズ")
	fmt.Println("  2. go-standards-checker を実行してチェック")
}
