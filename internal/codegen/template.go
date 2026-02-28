package codegen

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/thellimist/clihub/internal/closure"
)

var mainTemplate = template.Must(template.New("main.go").Funcs(template.FuncMap{
	"quoteSlice":       quoteSlice,
	"cobraFlag":        cobraFlagType,
	"defaultLit":       defaultValueLiteral,
	"varName":          toVarName,
	"funcName":         toFuncName,
	"quote":            quoteStr,
	"hasEnumDesc":      hasEnumDesc,
	"closureMerged":    closureMergedJSON,
	"closureIsHidden":  closureIsHidden,
	"closureHasParams": closureHasParams,
	"closureSkipFlag":  closureSkipFlag,
	"closureDefault":   closureDefaultLiteral,
}).Parse(mainTemplateSource))

var goModTemplate = template.Must(template.New("go.mod").Parse(goModTemplateSource))

func quoteStr(s string) string {
	return fmt.Sprintf("%q", s)
}

func quoteSlice(ss []string) string {
	if len(ss) == 0 {
		return "[]string(nil)"
	}
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = fmt.Sprintf("%q", s)
	}
	return "[]string{" + strings.Join(parts, ", ") + "}"
}

func cobraFlagType(goType string) string {
	switch goType {
	case "int":
		return "IntVar"
	case "float64":
		return "Float64Var"
	case "bool":
		return "BoolVar"
	case "[]string":
		return "StringSliceVar"
	case "[]int":
		return "IntSliceVar"
	default:
		return "StringVar"
	}
}

func defaultValueLiteral(goType string, defaultValue any) string {
	if defaultValue != nil {
		switch goType {
		case "string":
			if s, ok := defaultValue.(string); ok {
				return fmt.Sprintf("%q", s)
			}
		case "int":
			if f, ok := defaultValue.(float64); ok {
				return fmt.Sprintf("%d", int(f))
			}
		case "float64":
			if f, ok := defaultValue.(float64); ok {
				return fmt.Sprintf("%g", f)
			}
		case "bool":
			if b, ok := defaultValue.(bool); ok {
				return fmt.Sprintf("%t", b)
			}
		}
	}
	switch goType {
	case "int":
		return "0"
	case "float64":
		return "0"
	case "bool":
		return "false"
	case "[]string":
		return "nil"
	case "[]int":
		return "nil"
	default:
		return `""`
	}
}

func toVarName(flagName string) string {
	out := make([]byte, 0, len(flagName))
	upper := false
	for _, c := range flagName {
		if c == '-' {
			upper = true
			continue
		}
		if upper {
			if c >= 'a' && c <= 'z' {
				c = c - 32
			}
			upper = false
		}
		out = append(out, byte(c))
	}
	return "flag" + strings.ToUpper(string(out[:1])) + string(out[1:])
}

func toFuncName(toolName string) string {
	out := make([]byte, 0, len(toolName))
	upper := true
	for _, c := range toolName {
		if c == '-' || c == '_' {
			upper = true
			continue
		}
		if upper {
			if c >= 'a' && c <= 'z' {
				c = c - 32
			}
			upper = false
		}
		out = append(out, byte(c))
	}
	return string(out)
}

func hasEnumDesc(desc string, enums []string) string {
	if len(enums) == 0 {
		return desc
	}
	return desc + " (" + strings.Join(enums, "|") + ")"
}

// closureMergedJSON returns the merged closure params (global + tool-specific)
// as a Go-embeddable JSON string literal. Returns empty string if no params.
func closureMergedJSON(cfg *closure.Config, toolName string) string {
	if cfg == nil {
		return ""
	}
	merged := cfg.Merge(toolName)
	if len(merged) == 0 {
		return ""
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%q", string(data))
}

// closureIsHidden returns true if the closure config uses hidden mode.
func closureIsHidden(cfg *closure.Config) bool {
	return cfg != nil && cfg.Mode == closure.ModeHidden
}

// closureHasParams returns true if the closure config has any params for a tool.
func closureHasParams(cfg *closure.Config, toolName string) bool {
	if cfg == nil {
		return false
	}
	return len(cfg.Merge(toolName)) > 0
}

// closureSkipFlag returns true if this flag should be skipped in hidden mode
// (i.e., the param is closure-injected and mode is hidden).
func closureSkipFlag(cfg *closure.Config, toolName, propertyName string) bool {
	if cfg == nil || cfg.Mode != closure.ModeHidden {
		return false
	}
	merged := cfg.Merge(toolName)
	_, ok := merged[propertyName]
	return ok
}

// closureDefaultLiteral returns the Go literal for a closure default value
// given the flag's Go type. Used in "default" mode to override flag defaults.
func closureDefaultLiteral(cfg *closure.Config, toolName, propertyName, goType string) string {
	if cfg == nil || cfg.Mode != closure.ModeDefault {
		return ""
	}
	merged := cfg.Merge(toolName)
	val, ok := merged[propertyName]
	if !ok {
		return ""
	}
	return defaultValueLiteral(goType, val)
}
