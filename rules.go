package rules

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config 全体設定
type Config struct {
	Settings      Settings           `yaml:"settings"`
	Naming        NamingConfig       `yaml:"naming"`
	Structure     StructureConfig    `yaml:"structure"`
	ErrorHandling ErrorHandlingConfig `yaml:"error_handling"`
	Logging       LoggingConfig      `yaml:"logging"`
	Architecture  ArchitectureConfig `yaml:"architecture"`
	Directory     DirectoryConfig    `yaml:"directory"`
	StructTags    StructTagsConfig   `yaml:"struct_tags"`
	AWSLambda     AWSLambdaConfig    `yaml:"aws_lambda"`
	CustomRules   []CustomRule       `yaml:"custom_rules"`
	ProjectRules  []ProjectRule      `yaml:"project_rules"`
}

// Settings 基本設定
type Settings struct {
	TargetDir       string   `yaml:"target_dir"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
	ReportFormat    string   `yaml:"report_format"`
	MinSeverity     string   `yaml:"min_severity"`
}

// Severity 重要度
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ParseSeverity 文字列からSeverityを解析
func ParseSeverity(s string) Severity {
	switch s {
	case "error":
		return SeverityError
	case "warning":
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

// SeverityLevel 重要度の数値レベル
func (s Severity) Level() int {
	switch s {
	case SeverityError:
		return 3
	case SeverityWarning:
		return 2
	default:
		return 1
	}
}

// ========================================
// 命名規則設定
// ========================================

type NamingConfig struct {
	Enabled bool             `yaml:"enabled"`
	Rules   NamingRulesConfig `yaml:"rules"`
}

type NamingRulesConfig struct {
	PackageName   PatternRule   `yaml:"package_name"`
	ExportedNames BaseRule      `yaml:"exported_names"`
	Acronyms      AcronymsRule  `yaml:"acronyms"`
	FileName      PatternRule   `yaml:"file_name"`
	InterfaceName SuffixRule    `yaml:"interface_name"`
	ErrorVar      PatternRule   `yaml:"error_var"`
}

type BaseRule struct {
	Enabled  bool   `yaml:"enabled"`
	Severity string `yaml:"severity"`
	Message  string `yaml:"message"`
}

type PatternRule struct {
	BaseRule `yaml:",inline"`
	Pattern  string `yaml:"pattern"`
}

type AcronymsRule struct {
	BaseRule `yaml:",inline"`
	Words    []string `yaml:"words"`
}

type SuffixRule struct {
	BaseRule `yaml:",inline"`
	Suffixes []string `yaml:"suffixes"`
}

// ========================================
// コード構造設定
// ========================================

type StructureConfig struct {
	Enabled bool               `yaml:"enabled"`
	Rules   StructureRulesConfig `yaml:"rules"`
}

type StructureRulesConfig struct {
	MaxFunctionLines LimitRule `yaml:"max_function_lines"`
	MaxNestingLevel  LimitRule `yaml:"max_nesting_level"`
	MaxParameters    LimitRule `yaml:"max_parameters"`
	MaxReturnValues  LimitRule `yaml:"max_return_values"`
}

type LimitRule struct {
	BaseRule `yaml:",inline"`
	Limit    int `yaml:"limit"`
}

// ========================================
// エラーハンドリング設定
// ========================================

type ErrorHandlingConfig struct {
	Enabled bool                    `yaml:"enabled"`
	Rules   ErrorHandlingRulesConfig `yaml:"rules"`
}

type ErrorHandlingRulesConfig struct {
	NoIgnoredErrors IgnoredErrorsRule `yaml:"no_ignored_errors"`
	ErrorWrapping   BaseRule          `yaml:"error_wrapping"`
	NoPanic         AllowedInRule     `yaml:"no_panic"`
}

type IgnoredErrorsRule struct {
	BaseRule        `yaml:",inline"`
	AllowedPatterns []string `yaml:"allowed_patterns"`
}

type AllowedInRule struct {
	BaseRule  `yaml:",inline"`
	AllowedIn []string `yaml:"allowed_in"`
}

// ========================================
// ログ設定
// ========================================

type LoggingConfig struct {
	Enabled bool              `yaml:"enabled"`
	Rules   LoggingRulesConfig `yaml:"rules"`
}

type LoggingRulesConfig struct {
	NoStdLog     BaseRule `yaml:"no_std_log"`
	NoFmtPrintln BaseRule `yaml:"no_fmt_println"`
}

// ========================================
// アーキテクチャ設定
// ========================================

type ArchitectureConfig struct {
	Enabled bool                    `yaml:"enabled"`
	Rules   ArchitectureRulesConfig `yaml:"rules"`
}

type ArchitectureRulesConfig struct {
	LayerDependencies LayerDependenciesRule `yaml:"layer_dependencies"`
}

type LayerDependenciesRule struct {
	BaseRule `yaml:",inline"`
	Layers   []LayerRule `yaml:"layers"`
}

type LayerRule struct {
	Name         string   `yaml:"name"`
	CanImport    []string `yaml:"can_import"`
	CannotImport []string `yaml:"cannot_import"`
}

// ========================================
// ディレクトリ設定
// ========================================

type DirectoryConfig struct {
	Enabled bool                 `yaml:"enabled"`
	Rules   DirectoryRulesConfig `yaml:"rules"`
}

type DirectoryRulesConfig struct {
	RequiredDirs    DirsRule `yaml:"required_dirs"`
	RecommendedDirs DirsRule `yaml:"recommended_dirs"`
}

type DirsRule struct {
	BaseRule `yaml:",inline"`
	Dirs     []string `yaml:"dirs"`
}

// ========================================
// 構造体タグ設定
// ========================================

type StructTagsConfig struct {
	Enabled bool                  `yaml:"enabled"`
	Rules   StructTagsRulesConfig `yaml:"rules"`
}

type StructTagsRulesConfig struct {
	JSONTag       JSONTagRule       `yaml:"json_tag"`
	ValidationTag ValidationTagRule `yaml:"validation_tag"`
}

type JSONTagRule struct {
	BaseRule `yaml:",inline"`
	Style    string `yaml:"style"`
}

type ValidationTagRule struct {
	BaseRule    `yaml:",inline"`
	RequiredFor []string `yaml:"required_for"`
}

// ========================================
// AWS Lambda設定
// ========================================

type AWSLambdaConfig struct {
	Enabled bool                 `yaml:"enabled"`
	Rules   AWSLambdaRulesConfig `yaml:"rules"`
}

type AWSLambdaRulesConfig struct {
	InitAWSClients     BaseRule `yaml:"init_aws_clients"`
	ContextPropagation BaseRule `yaml:"context_propagation"`
	SQSBatchFailures   BaseRule `yaml:"sqs_batch_failures"`
}

// ========================================
// カスタムルール
// ========================================

type CustomRule struct {
	Name         string   `yaml:"name"`
	Enabled      bool     `yaml:"enabled"`
	Severity     string   `yaml:"severity"`
	Pattern      string   `yaml:"pattern"`
	Message      string   `yaml:"message"`
	ExcludeFiles []string `yaml:"exclude_files"`
}

// Compile パターンをコンパイル
func (r *CustomRule) Compile() (*regexp.Regexp, error) {
	return regexp.Compile(r.Pattern)
}

// ProjectRule プロジェクト固有ルール
type ProjectRule struct {
	Name     string   `yaml:"name"`
	Enabled  bool     `yaml:"enabled"`
	Severity string   `yaml:"severity"`
	Type     string   `yaml:"type"`
	Packages []string `yaml:"packages"`
	Message  string   `yaml:"message"`
}

// ========================================
// 設定読み込み
// ========================================

// LoadConfig 設定ファイルを読み込む
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// DefaultConfig デフォルト設定を返す
func DefaultConfig() *Config {
	return &Config{
		Settings: Settings{
			ReportFormat: "text",
			MinSeverity:  "info",
			ExcludePatterns: []string{
				"*_test.go",
				"vendor/*",
				".git/*",
			},
		},
		Naming: NamingConfig{
			Enabled: true,
			Rules: NamingRulesConfig{
				PackageName: PatternRule{
					BaseRule: BaseRule{Enabled: true, Severity: "error", Message: "パッケージ名は小文字のみ"},
					Pattern:  "^[a-z][a-z0-9]*$",
				},
				FileName: PatternRule{
					BaseRule: BaseRule{Enabled: true, Severity: "warning", Message: "ファイル名はスネークケース"},
					Pattern:  "^[a-z][a-z0-9_]*\\.go$",
				},
			},
		},
		Structure: StructureConfig{
			Enabled: true,
			Rules: StructureRulesConfig{
				MaxFunctionLines: LimitRule{
					BaseRule: BaseRule{Enabled: true, Severity: "warning", Message: "関数は50行以内"},
					Limit:    50,
				},
				MaxNestingLevel: LimitRule{
					BaseRule: BaseRule{Enabled: true, Severity: "warning", Message: "ネストは3レベル以内"},
					Limit:    3,
				},
			},
		},
		ErrorHandling: ErrorHandlingConfig{
			Enabled: true,
			Rules: ErrorHandlingRulesConfig{
				NoIgnoredErrors: IgnoredErrorsRule{
					BaseRule: BaseRule{Enabled: true, Severity: "error", Message: "エラーを無視しないでください"},
				},
			},
		},
	}
}
