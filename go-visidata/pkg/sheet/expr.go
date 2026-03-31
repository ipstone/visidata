package sheet

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
)

var exprIdentifierRE = regexp.MustCompile(`[^[:alnum:]_]+`)

func (s *Sheet) AddExprColumn(input string) (string, error) {
	name, expression, err := parseExprColumnSpec(input)
	if err != nil {
		return "", err
	}

	program, err := expr.Compile(expression,
		expr.Env(map[string]any{}),
		expr.AllowUndefinedVariables(),
		expr.AsAny(),
		expr.Function("str", exprString),
	)
	if err != nil {
		return "", err
	}

	values := make([]string, len(s.Rows))
	for rowIndex := range s.Rows {
		result, err := expr.Run(program, s.exprEnvForRow(rowIndex))
		if err != nil {
			return "", fmt.Errorf("row %d: %w", rowIndex+1, err)
		}
		values[rowIndex] = formatExprResult(result)
	}

	if name == "" {
		name = s.nextExprColumnName()
	}
	name = uniqueColumnName(s, name)

	s.pushUndo("add expression column")
	s.Columns = append(s.Columns, Column{Name: name, Kind: KindString})
	for rowIndex := range s.Rows {
		s.Rows[rowIndex] = append(s.Rows[rowIndex], values[rowIndex])
	}
	lastCol := len(s.Columns) - 1
	s.Columns[lastCol].Kind = inferKindForColumn(s.Rows, lastCol)
	s.CursorCol = lastCol
	s.clampCursor()
	s.refreshSearch()

	return name, nil
}

func parseExprColumnSpec(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("expression cannot be empty")
	}
	if strings.Contains(input, ":=") {
		parts := strings.SplitN(input, ":=", 2)
		name := strings.TrimSpace(parts[0])
		expression := strings.TrimSpace(parts[1])
		if name == "" {
			return "", "", fmt.Errorf("expression column name cannot be empty")
		}
		if expression == "" {
			return "", "", fmt.Errorf("expression cannot be empty")
		}
		return name, expression, nil
	}
	return "", input, nil
}

func (s *Sheet) exprEnvForRow(rowIndex int) map[string]any {
	env := make(map[string]any, len(s.Columns)*3)
	for colIndex, col := range s.Columns {
		value := typedValue(col.EffectiveKind(), s.Cell(rowIndex, colIndex))
		alias := fmt.Sprintf("col%d", colIndex+1)
		env[alias] = value
		if isExprIdentifier(col.Name) {
			setExprEnvValue(env, col.Name, value)
		}
		setExprEnvValue(env, safeExprIdentifier(col.Name), value)
	}
	return env
}

func setExprEnvValue(env map[string]any, name string, value any) {
	if name == "" {
		return
	}
	if _, exists := env[name]; exists {
		return
	}
	env[name] = value
}

func isExprIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && r != '_' {
				return false
			}
			continue
		}
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func safeExprIdentifier(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = exprIdentifierRE.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return ""
	}
	first := rune(name[0])
	if (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') && first != '_' {
		name = "_" + name
	}
	return name
}

func formatExprResult(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int8:
		return strconv.FormatInt(int64(typed), 10)
	case int16:
		return strconv.FormatInt(int64(typed), 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(value)
	}
}

func exprString(params ...any) (any, error) {
	if len(params) != 1 {
		return nil, fmt.Errorf("str expects exactly one argument")
	}
	return formatExprResult(params[0]), nil
}

func (s *Sheet) nextExprColumnName() string {
	base := "expr"
	for index := 1; ; index++ {
		name := fmt.Sprintf("%s_%d", base, index)
		if !sheetHasColumnName(s, name) {
			return name
		}
	}
}

func uniqueColumnName(s *Sheet, name string) string {
	if !sheetHasColumnName(s, name) {
		return name
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s_%d", name, index)
		if !sheetHasColumnName(s, candidate) {
			return candidate
		}
	}
}

func sheetHasColumnName(s *Sheet, name string) bool {
	for _, col := range s.Columns {
		if col.Name == name {
			return true
		}
	}
	return false
}
