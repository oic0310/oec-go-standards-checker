package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-standards-checker/rules"
)

// Violation 違反情報
type Violation struct {
	File       string         `json:"file"`
	Line       int            `json:"line"`
	Column     int            `json:"column"`
	Rule       string         `json:"rule"`
	Category   string         `json:"category"`
	Severity   rules.Severity `json:"severity"`
	Message    string         `json:"message"`
	Suggestion string         `json:"suggestion,omitempty"`
	Code       string         `json:"code,omitempty"` // 該当コード行
}

// Report チェックレポート
type Report struct {
	ProjectPath string      `json:"project_path"`
	TotalFiles  int         `json:"total_files"`
	Violations  []Violation `json:"violations"`
	Summary     Summary     `json:"summary"`
}

// Summary サマリー情報
type Summary struct {
	TotalViolations int            `json:"total_violations"`
	ByCategory      map[string]int `json:"by_category"`
	BySeverity      map[string]int `json:"by_severity"`
	PassedRules     int            `json:"passed_rules"`
	FailedRules     int            `json:"failed_rules"`
}

// NewReport 新しいレポートを作成
func NewReport(projectPath string) *Report {
	return &Report{
		ProjectPath: projectPath,
		Violations:  make([]Violation, 0),
		Summary: Summary{
			ByCategory: make(map[string]int),
			BySeverity: make(map[string]int),
		},
	}
}

// AddViolation 違反を追加
func (r *Report) AddViolation(v Violation) {
	r.Violations = append(r.Violations, v)
}

// Finalize レポートを完成させる
func (r *Report) Finalize() {
	r.Summary.TotalViolations = len(r.Violations)

	// カテゴリ別カウント
	for _, v := range r.Violations {
		r.Summary.ByCategory[v.Category]++
		r.Summary.BySeverity[string(v.Severity)]++
	}

	// 違反を重要度・ファイル順にソート
	sort.Slice(r.Violations, func(i, j int) bool {
		if r.Violations[i].Severity.Level() != r.Violations[j].Severity.Level() {
			return r.Violations[i].Severity.Level() > r.Violations[j].Severity.Level()
		}
		if r.Violations[i].File != r.Violations[j].File {
			return r.Violations[i].File < r.Violations[j].File
		}
		return r.Violations[i].Line < r.Violations[j].Line
	})
}

// Filter 重要度でフィルタリング
func (r *Report) Filter(minSeverity rules.Severity) *Report {
	filtered := NewReport(r.ProjectPath)
	filtered.TotalFiles = r.TotalFiles

	for _, v := range r.Violations {
		if v.Severity.Level() >= minSeverity.Level() {
			filtered.AddViolation(v)
		}
	}

	filtered.Finalize()
	return filtered
}

// ToJSON JSON形式で出力
func (r *Report) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToText テキスト形式で出力
func (r *Report) ToText() string {
	var sb strings.Builder

	// ヘッダー
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║          Go Standards Checker - Compliance Report                    ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════╝\n\n")

	sb.WriteString(fmt.Sprintf("📁 Project: %s\n", r.ProjectPath))
	sb.WriteString(fmt.Sprintf("📄 Files Checked: %d\n\n", r.TotalFiles))

	// サマリー
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("                              SUMMARY                                   \n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	errorCount := r.Summary.BySeverity["error"]
	warningCount := r.Summary.BySeverity["warning"]
	infoCount := r.Summary.BySeverity["info"]

	sb.WriteString(fmt.Sprintf("🔴 Errors:   %d\n", errorCount))
	sb.WriteString(fmt.Sprintf("🟡 Warnings: %d\n", warningCount))
	sb.WriteString(fmt.Sprintf("🔵 Info:     %d\n", infoCount))
	sb.WriteString(fmt.Sprintf("📊 Total:    %d violations\n\n", r.Summary.TotalViolations))

	// カテゴリ別
	if len(r.Summary.ByCategory) > 0 {
		sb.WriteString("By Category:\n")
		for category, count := range r.Summary.ByCategory {
			sb.WriteString(fmt.Sprintf("  • %s: %d\n", category, count))
		}
		sb.WriteString("\n")
	}

	// 違反がない場合
	if len(r.Violations) == 0 {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("✅ Congratulations! No violations found.\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		return sb.String()
	}

	// 違反詳細
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("                             VIOLATIONS                                 \n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	currentFile := ""
	for i, v := range r.Violations {
		// ファイルが変わったらヘッダー出力
		if v.File != currentFile {
			currentFile = v.File
			sb.WriteString(fmt.Sprintf("📄 %s\n", currentFile))
			sb.WriteString("────────────────────────────────────────────────────────────────────────\n")
		}

		// 重要度アイコン
		icon := "🔵"
		switch v.Severity {
		case rules.SeverityError:
			icon = "🔴"
		case rules.SeverityWarning:
			icon = "🟡"
		}

		// 違反情報
		sb.WriteString(fmt.Sprintf("%s [%s] Line %d: %s\n", icon, v.Rule, v.Line, v.Message))
		
		// コードがあれば表示
		if v.Code != "" {
			sb.WriteString(fmt.Sprintf("   │ %s\n", strings.TrimSpace(v.Code)))
		}
		
		// 提案があれば表示
		if v.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("   💡 Suggestion: %s\n", v.Suggestion))
		}

		// 最後の違反以外は空行
		if i < len(r.Violations)-1 && r.Violations[i+1].File == currentFile {
			sb.WriteString("\n")
		} else if i < len(r.Violations)-1 {
			sb.WriteString("\n")
		}
	}

	// フッター
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	if errorCount > 0 {
		sb.WriteString("❌ Check FAILED - Please fix errors before committing.\n")
	} else if warningCount > 0 {
		sb.WriteString("⚠️  Check PASSED with warnings - Consider reviewing.\n")
	} else {
		sb.WriteString("✅ Check PASSED - Good job!\n")
	}
	
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return sb.String()
}

// HasErrors エラーがあるか
func (r *Report) HasErrors() bool {
	return r.Summary.BySeverity["error"] > 0
}

// HasWarnings 警告があるか
func (r *Report) HasWarnings() bool {
	return r.Summary.BySeverity["warning"] > 0
}

// ExitCode 終了コードを返す
func (r *Report) ExitCode() int {
	if r.HasErrors() {
		return 1
	}
	return 0
}
