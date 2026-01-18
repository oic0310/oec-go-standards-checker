package checker

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-standards-checker/report"
	"github.com/go-standards-checker/rules"
)

// Checker 標準準拠チェッカー
type Checker struct {
	config  *rules.Config
	report  *report.Report
	fset    *token.FileSet
	fileMap map[string][]string // ファイル名→行内容のマップ
}

// NewChecker チェッカーを作成
func NewChecker(config *rules.Config) *Checker {
	return &Checker{
		config:  config,
		fset:    token.NewFileSet(),
		fileMap: make(map[string][]string),
	}
}

// Check ディレクトリをチェック
func (c *Checker) Check(targetDir string) (*report.Report, error) {
	c.report = report.NewReport(targetDir)

	// ディレクトリ構成チェック
	if c.config.Directory.Enabled {
		c.checkDirectory(targetDir)
	}

	// Goファイルを収集
	goFiles, err := c.collectGoFiles(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Go files: %w", err)
	}

	c.report.TotalFiles = len(goFiles)

	// 各ファイルをチェック
	for _, filePath := range goFiles {
		if err := c.checkFile(filePath); err != nil {
			fmt.Printf("Warning: failed to check %s: %v\n", filePath, err)
		}
	}

	// カスタムルールチェック
	for _, filePath := range goFiles {
		c.checkCustomRules(filePath)
	}

	c.report.Finalize()
	return c.report, nil
}

// collectGoFiles Goファイルを収集
func (c *Checker) collectGoFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// ディレクトリはスキップ判定のみ
		if info.IsDir() {
			// 除外パターンにマッチするディレクトリをスキップ
			for _, pattern := range c.config.Settings.ExcludePatterns {
				if matched, _ := filepath.Match(pattern, info.Name()); matched {
					return filepath.SkipDir
				}
				if matched, _ := filepath.Match(pattern, path); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// .goファイルのみ
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 除外パターンチェック
		relPath, _ := filepath.Rel(dir, path)
		for _, pattern := range c.config.Settings.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				return nil
			}
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// checkFile 単一ファイルをチェック
func (c *Checker) checkFile(filePath string) error {
	// ファイル内容を読み込み
	lines, err := c.readFileLines(filePath)
	if err != nil {
		return err
	}
	c.fileMap[filePath] = lines

	// AST解析
	file, err := parser.ParseFile(c.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// ファイル名チェック
	if c.config.Naming.Enabled && c.config.Naming.Rules.FileName.Enabled {
		c.checkFileName(filePath)
	}

	// パッケージ名チェック
	if c.config.Naming.Enabled && c.config.Naming.Rules.PackageName.Enabled {
		c.checkPackageName(file, filePath)
	}

	// 各種チェック
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			c.checkFunction(node, filePath)
		case *ast.GenDecl:
			c.checkGenDecl(node, filePath)
		case *ast.TypeSpec:
			c.checkTypeSpec(node, filePath)
		case *ast.AssignStmt:
			c.checkAssignment(node, filePath)
		case *ast.CallExpr:
			c.checkCallExpr(node, filePath)
		}
		return true
	})

	return nil
}

// readFileLines ファイルを行単位で読み込み
func (c *Checker) readFileLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// getCodeLine 指定行のコードを取得
func (c *Checker) getCodeLine(filePath string, line int) string {
	lines, ok := c.fileMap[filePath]
	if !ok || line < 1 || line > len(lines) {
		return ""
	}
	return lines[line-1]
}

// ========================================
// ファイル名チェック
// ========================================

func (c *Checker) checkFileName(filePath string) {
	fileName := filepath.Base(filePath)
	rule := c.config.Naming.Rules.FileName

	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return
	}

	if !pattern.MatchString(fileName) {
		c.report.AddViolation(report.Violation{
			File:       filePath,
			Line:       1,
			Rule:       "file_name",
			Category:   "naming",
			Severity:   rules.ParseSeverity(rule.Severity),
			Message:    rule.Message,
			Suggestion: fmt.Sprintf("Rename to: %s", toSnakeCase(strings.TrimSuffix(fileName, ".go"))+".go"),
		})
	}
}

// ========================================
// パッケージ名チェック
// ========================================

func (c *Checker) checkPackageName(file *ast.File, filePath string) {
	rule := c.config.Naming.Rules.PackageName
	pkgName := file.Name.Name

	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return
	}

	if !pattern.MatchString(pkgName) {
		pos := c.fset.Position(file.Name.Pos())
		c.report.AddViolation(report.Violation{
			File:       filePath,
			Line:       pos.Line,
			Column:     pos.Column,
			Rule:       "package_name",
			Category:   "naming",
			Severity:   rules.ParseSeverity(rule.Severity),
			Message:    fmt.Sprintf("%s: '%s'", rule.Message, pkgName),
			Code:       c.getCodeLine(filePath, pos.Line),
		})
	}
}

// ========================================
// 関数チェック
// ========================================

func (c *Checker) checkFunction(fn *ast.FuncDecl, filePath string) {
	pos := c.fset.Position(fn.Pos())
	endPos := c.fset.Position(fn.End())
	funcName := fn.Name.Name

	// 関数名チェック（公開/非公開）
	if c.config.Naming.Enabled && c.config.Naming.Rules.ExportedNames.Enabled {
		if ast.IsExported(funcName) {
			// PascalCaseチェック
			if !isPascalCase(funcName) {
				c.report.AddViolation(report.Violation{
					File:     filePath,
					Line:     pos.Line,
					Column:   pos.Column,
					Rule:     "exported_name",
					Category: "naming",
					Severity: rules.ParseSeverity(c.config.Naming.Rules.ExportedNames.Severity),
					Message:  fmt.Sprintf("公開関数 '%s' はPascalCaseで命名してください", funcName),
					Code:     c.getCodeLine(filePath, pos.Line),
				})
			}
		}
	}

	// 関数行数チェック
	if c.config.Structure.Enabled && c.config.Structure.Rules.MaxFunctionLines.Enabled {
		lineCount := endPos.Line - pos.Line
		limit := c.config.Structure.Rules.MaxFunctionLines.Limit

		if lineCount > limit {
			c.report.AddViolation(report.Violation{
				File:       filePath,
				Line:       pos.Line,
				Rule:       "max_function_lines",
				Category:   "structure",
				Severity:   rules.ParseSeverity(c.config.Structure.Rules.MaxFunctionLines.Severity),
				Message:    fmt.Sprintf("関数 '%s' は%d行あります（上限: %d行）", funcName, lineCount, limit),
				Code:       c.getCodeLine(filePath, pos.Line),
				Suggestion: "関数を分割してください",
			})
		}
	}

	// パラメータ数チェック
	if c.config.Structure.Enabled && c.config.Structure.Rules.MaxParameters.Enabled {
		if fn.Type.Params != nil {
			paramCount := len(fn.Type.Params.List)
			limit := c.config.Structure.Rules.MaxParameters.Limit

			if paramCount > limit {
				c.report.AddViolation(report.Violation{
					File:       filePath,
					Line:       pos.Line,
					Rule:       "max_parameters",
					Category:   "structure",
					Severity:   rules.ParseSeverity(c.config.Structure.Rules.MaxParameters.Severity),
					Message:    fmt.Sprintf("関数 '%s' のパラメータ数は%d個です（上限: %d個）", funcName, paramCount, limit),
					Code:       c.getCodeLine(filePath, pos.Line),
					Suggestion: "パラメータを構造体にまとめることを検討してください",
				})
			}
		}
	}

	// 戻り値数チェック
	if c.config.Structure.Enabled && c.config.Structure.Rules.MaxReturnValues.Enabled {
		if fn.Type.Results != nil {
			resultCount := len(fn.Type.Results.List)
			limit := c.config.Structure.Rules.MaxReturnValues.Limit

			if resultCount > limit {
				c.report.AddViolation(report.Violation{
					File:       filePath,
					Line:       pos.Line,
					Rule:       "max_return_values",
					Category:   "structure",
					Severity:   rules.ParseSeverity(c.config.Structure.Rules.MaxReturnValues.Severity),
					Message:    fmt.Sprintf("関数 '%s' の戻り値数は%d個です（上限: %d個）", funcName, resultCount, limit),
					Code:       c.getCodeLine(filePath, pos.Line),
					Suggestion: "戻り値を構造体にまとめることを検討してください",
				})
			}
		}
	}

	// ネストレベルチェック
	if c.config.Structure.Enabled && c.config.Structure.Rules.MaxNestingLevel.Enabled {
		maxNest := c.checkNestingLevel(fn.Body, 0)
		limit := c.config.Structure.Rules.MaxNestingLevel.Limit

		if maxNest > limit {
			c.report.AddViolation(report.Violation{
				File:       filePath,
				Line:       pos.Line,
				Rule:       "max_nesting_level",
				Category:   "structure",
				Severity:   rules.ParseSeverity(c.config.Structure.Rules.MaxNestingLevel.Severity),
				Message:    fmt.Sprintf("関数 '%s' のネストレベルは%dです（上限: %d）", funcName, maxNest, limit),
				Code:       c.getCodeLine(filePath, pos.Line),
				Suggestion: "早期リターンを使用してネストを浅くしてください",
			})
		}
	}
}

// checkNestingLevel ネストレベルを計算
func (c *Checker) checkNestingLevel(block *ast.BlockStmt, currentLevel int) int {
	if block == nil {
		return currentLevel
	}

	maxLevel := currentLevel
	for _, stmt := range block.List {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			level := c.checkNestingLevel(s.Body, currentLevel+1)
			if level > maxLevel {
				maxLevel = level
			}
			if s.Else != nil {
				if elseBlock, ok := s.Else.(*ast.BlockStmt); ok {
					level = c.checkNestingLevel(elseBlock, currentLevel+1)
					if level > maxLevel {
						maxLevel = level
					}
				}
			}
		case *ast.ForStmt:
			level := c.checkNestingLevel(s.Body, currentLevel+1)
			if level > maxLevel {
				maxLevel = level
			}
		case *ast.RangeStmt:
			level := c.checkNestingLevel(s.Body, currentLevel+1)
			if level > maxLevel {
				maxLevel = level
			}
		case *ast.SwitchStmt:
			level := c.checkNestingLevel(s.Body, currentLevel+1)
			if level > maxLevel {
				maxLevel = level
			}
		case *ast.SelectStmt:
			level := c.checkNestingLevel(s.Body, currentLevel+1)
			if level > maxLevel {
				maxLevel = level
			}
		}
	}
	return maxLevel
}

// ========================================
// 型定義チェック
// ========================================

func (c *Checker) checkTypeSpec(ts *ast.TypeSpec, filePath string) {
	pos := c.fset.Position(ts.Pos())
	typeName := ts.Name.Name

	// インタフェース名チェック
	if _, ok := ts.Type.(*ast.InterfaceType); ok {
		if c.config.Naming.Enabled && c.config.Naming.Rules.InterfaceName.Enabled {
			rule := c.config.Naming.Rules.InterfaceName
			validSuffix := false
			for _, suffix := range rule.Suffixes {
				if strings.HasSuffix(typeName, suffix) {
					validSuffix = true
					break
				}
			}

			if !validSuffix && ast.IsExported(typeName) {
				c.report.AddViolation(report.Violation{
					File:       filePath,
					Line:       pos.Line,
					Column:     pos.Column,
					Rule:       "interface_name",
					Category:   "naming",
					Severity:   rules.ParseSeverity(rule.Severity),
					Message:    fmt.Sprintf("インタフェース '%s' は標準的なサフィックス(%v)を使用してください", typeName, rule.Suffixes),
					Code:       c.getCodeLine(filePath, pos.Line),
				})
			}
		}
	}

	// 構造体タグチェック
	if st, ok := ts.Type.(*ast.StructType); ok {
		if c.config.StructTags.Enabled {
			c.checkStructTags(st, typeName, filePath)
		}
	}
}

// ========================================
// 構造体タグチェック
// ========================================

func (c *Checker) checkStructTags(st *ast.StructType, structName string, filePath string) {
	if st.Fields == nil {
		return
	}

	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		pos := c.fset.Position(field.Pos())
		tagValue := field.Tag.Value

		// JSONタグチェック
		if c.config.StructTags.Rules.JSONTag.Enabled {
			c.checkJSONTag(tagValue, structName, filePath, pos)
		}

		// バリデーションタグチェック
		if c.config.StructTags.Rules.ValidationTag.Enabled {
			c.checkValidationTag(tagValue, structName, filePath, pos)
		}
	}
}

func (c *Checker) checkJSONTag(tagValue, structName, filePath string, pos token.Position) {
	rule := c.config.StructTags.Rules.JSONTag

	// json:"xxx" を抽出
	jsonTagRe := regexp.MustCompile(`json:"([^"]+)"`)
	matches := jsonTagRe.FindStringSubmatch(tagValue)
	if len(matches) < 2 {
		return
	}

	jsonName := strings.Split(matches[1], ",")[0]
	if jsonName == "-" || jsonName == "" {
		return
	}

	var isValid bool
	switch rule.Style {
	case "snake_case":
		isValid = isSnakeCase(jsonName)
	case "camelCase":
		isValid = isCamelCase(jsonName)
	default:
		isValid = true
	}

	if !isValid {
		c.report.AddViolation(report.Violation{
			File:       filePath,
			Line:       pos.Line,
			Column:     pos.Column,
			Rule:       "json_tag",
			Category:   "struct_tags",
			Severity:   rules.ParseSeverity(rule.Severity),
			Message:    fmt.Sprintf("JSONタグ '%s' は%sで命名してください", jsonName, rule.Style),
			Code:       c.getCodeLine(filePath, pos.Line),
			Suggestion: fmt.Sprintf("json:\"%s\"", toSnakeCase(jsonName)),
		})
	}
}

func (c *Checker) checkValidationTag(tagValue, structName, filePath string, pos token.Position) {
	rule := c.config.StructTags.Rules.ValidationTag

	// 対象構造体かチェック
	isTarget := false
	for _, pattern := range rule.RequiredFor {
		if matched, _ := filepath.Match(pattern, structName); matched {
			isTarget = true
			break
		}
	}

	if !isTarget {
		return
	}

	// validateタグがあるかチェック
	if !strings.Contains(tagValue, `validate:"`) {
		c.report.AddViolation(report.Violation{
			File:       filePath,
			Line:       pos.Line,
			Column:     pos.Column,
			Rule:       "validation_tag",
			Category:   "struct_tags",
			Severity:   rules.ParseSeverity(rule.Severity),
			Message:    rule.Message,
			Code:       c.getCodeLine(filePath, pos.Line),
		})
	}
}

// ========================================
// 変数宣言チェック
// ========================================

func (c *Checker) checkGenDecl(gd *ast.GenDecl, filePath string) {
	if gd.Tok != token.VAR {
		return
	}

	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, name := range vs.Names {
			// センチネルエラーチェック
			if c.config.Naming.Enabled && c.config.Naming.Rules.ErrorVar.Enabled {
				// エラー型の変数かチェック
				if vs.Type != nil {
					if ident, ok := vs.Type.(*ast.Ident); ok && ident.Name == "error" {
						c.checkErrorVarName(name, filePath)
					}
				}
			}
		}
	}
}

func (c *Checker) checkErrorVarName(name *ast.Ident, filePath string) {
	rule := c.config.Naming.Rules.ErrorVar
	pos := c.fset.Position(name.Pos())

	if !ast.IsExported(name.Name) {
		return // 非公開エラーは対象外
	}

	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return
	}

	if !pattern.MatchString(name.Name) {
		c.report.AddViolation(report.Violation{
			File:       filePath,
			Line:       pos.Line,
			Column:     pos.Column,
			Rule:       "error_var",
			Category:   "naming",
			Severity:   rules.ParseSeverity(rule.Severity),
			Message:    fmt.Sprintf("エラー変数 '%s' はErrプレフィックスで命名してください", name.Name),
			Code:       c.getCodeLine(filePath, pos.Line),
			Suggestion: fmt.Sprintf("Err%s", strings.TrimPrefix(name.Name, "err")),
		})
	}
}

// ========================================
// 代入文チェック（エラー無視検出）
// ========================================

func (c *Checker) checkAssignment(as *ast.AssignStmt, filePath string) {
	if !c.config.ErrorHandling.Enabled || !c.config.ErrorHandling.Rules.NoIgnoredErrors.Enabled {
		return
	}

	// _ への代入をチェック
	for i, lhs := range as.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name != "_" {
			continue
		}

		// 右辺がエラーを返す可能性のある関数呼び出しかチェック
		if i < len(as.Rhs) {
			if call, ok := as.Rhs[i].(*ast.CallExpr); ok {
				// 許可パターンをチェック
				callStr := c.getCallExprString(call)
				rule := c.config.ErrorHandling.Rules.NoIgnoredErrors
				allowed := false
				for _, pattern := range rule.AllowedPatterns {
					if matched, _ := regexp.MatchString(pattern, callStr); matched {
						allowed = true
						break
					}
				}

				if !allowed {
					pos := c.fset.Position(as.Pos())
					c.report.AddViolation(report.Violation{
						File:       filePath,
						Line:       pos.Line,
						Column:     pos.Column,
						Rule:       "no_ignored_errors",
						Category:   "error_handling",
						Severity:   rules.ParseSeverity(rule.Severity),
						Message:    rule.Message,
						Code:       c.getCodeLine(filePath, pos.Line),
						Suggestion: "エラーを適切にハンドリングしてください",
					})
				}
			}
		}
	}
}

// ========================================
// 関数呼び出しチェック
// ========================================

func (c *Checker) checkCallExpr(call *ast.CallExpr, filePath string) {
	callStr := c.getCallExprString(call)
	pos := c.fset.Position(call.Pos())

	// panic チェック
	if c.config.ErrorHandling.Enabled && c.config.ErrorHandling.Rules.NoPanic.Enabled {
		if callStr == "panic" {
			rule := c.config.ErrorHandling.Rules.NoPanic
			// 許可されたファイルかチェック
			allowed := false
			for _, pattern := range rule.AllowedIn {
				if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
					allowed = true
					break
				}
			}

			if !allowed {
				c.report.AddViolation(report.Violation{
					File:       filePath,
					Line:       pos.Line,
					Rule:       "no_panic",
					Category:   "error_handling",
					Severity:   rules.ParseSeverity(rule.Severity),
					Message:    rule.Message,
					Code:       c.getCodeLine(filePath, pos.Line),
					Suggestion: "エラーを返却してください",
				})
			}
		}
	}

	// fmt.Println チェック
	if c.config.Logging.Enabled && c.config.Logging.Rules.NoFmtPrintln.Enabled {
		if strings.HasPrefix(callStr, "fmt.Print") {
			rule := c.config.Logging.Rules.NoFmtPrintln
			c.report.AddViolation(report.Violation{
				File:       filePath,
				Line:       pos.Line,
				Rule:       "no_fmt_println",
				Category:   "logging",
				Severity:   rules.ParseSeverity(rule.Severity),
				Message:    rule.Message,
				Code:       c.getCodeLine(filePath, pos.Line),
				Suggestion: "構造化ログライブラリ（zerolog等）を使用してください",
			})
		}
	}
}

func (c *Checker) getCallExprString(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if x, ok := fn.X.(*ast.Ident); ok {
			return x.Name + "." + fn.Sel.Name
		}
	}
	return ""
}

// ========================================
// ディレクトリ構成チェック
// ========================================

func (c *Checker) checkDirectory(targetDir string) {
	// 必須ディレクトリ
	if c.config.Directory.Rules.RequiredDirs.Enabled {
		rule := c.config.Directory.Rules.RequiredDirs
		for _, dir := range rule.Dirs {
			path := filepath.Join(targetDir, dir)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				c.report.AddViolation(report.Violation{
					File:       targetDir,
					Line:       1,
					Rule:       "required_dirs",
					Category:   "directory",
					Severity:   rules.ParseSeverity(rule.Severity),
					Message:    fmt.Sprintf("必須ディレクトリ '%s' が見つかりません", dir),
					Suggestion: fmt.Sprintf("mkdir -p %s", path),
				})
			}
		}
	}

	// 推奨ディレクトリ
	if c.config.Directory.Rules.RecommendedDirs.Enabled {
		rule := c.config.Directory.Rules.RecommendedDirs
		for _, dir := range rule.Dirs {
			path := filepath.Join(targetDir, dir)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				c.report.AddViolation(report.Violation{
					File:       targetDir,
					Line:       1,
					Rule:       "recommended_dirs",
					Category:   "directory",
					Severity:   rules.ParseSeverity(rule.Severity),
					Message:    fmt.Sprintf("推奨ディレクトリ '%s' が見つかりません", dir),
				})
			}
		}
	}
}

// ========================================
// カスタムルールチェック
// ========================================

func (c *Checker) checkCustomRules(filePath string) {
	for _, rule := range c.config.CustomRules {
		if !rule.Enabled {
			continue
		}

		// 除外ファイルチェック
		excluded := false
		for _, pattern := range rule.ExcludeFiles {
			if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// パターンマッチ
		pattern, err := rule.Compile()
		if err != nil {
			continue
		}

		lines := c.fileMap[filePath]
		for i, line := range lines {
			if pattern.MatchString(line) {
				c.report.AddViolation(report.Violation{
					File:     filePath,
					Line:     i + 1,
					Rule:     rule.Name,
					Category: "custom",
					Severity: rules.ParseSeverity(rule.Severity),
					Message:  rule.Message,
					Code:     strings.TrimSpace(line),
				})
			}
		}
	}
}

// ========================================
// ヘルパー関数
// ========================================

func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	return true
}

func isCamelCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] < 'a' || s[0] > 'z' {
		return false
	}
	return true
}

func isSnakeCase(s string) bool {
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9_]*$`, s)
	return matched
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
