package template

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/Masterminds/sprig/v3"
	"github.com/PastureStack/compose-cli/template/funcs"
)

var ErrCatalogTemplate = errors.New("CatalogLaunchTemplateUnsupported")

const MaxCatalogTemplateBytes = 1024 * 1024

type catalogWriter struct{ bytes.Buffer }

func (w *catalogWriter) WriteString(input string) (int, error) { return w.Write([]byte(input)) }

func (w *catalogWriter) Write(input []byte) (int, error) {
	if w.Len()+len(input) > MaxCatalogTemplateBytes {
		return 0, ErrCatalogTemplate
	}
	return w.Buffer.Write(input)
}

func catalogPrintf(format string, args ...interface{}) (string, error) {
	if len(format) > 256 || len(args) > 32 {
		return "", ErrCatalogTemplate
	}
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i < len(format) && format[i] == '%' {
			continue
		}
		for i < len(format) && strings.ContainsRune("#0+- ", rune(format[i])) {
			i++
		}
		for part := 0; part < 2; part++ {
			start := i
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				i++
			}
			if i > start {
				width, err := strconv.Atoi(format[start:i])
				if err != nil || width > 4096 {
					return "", ErrCatalogTemplate
				}
			}
			if part == 0 && i < len(format) && format[i] == '.' {
				i++
				continue
			}
			break
		}
		if i >= len(format) || !strings.ContainsRune("vsqdxXbOocUeEfFgGtT", rune(format[i])) {
			return "", ErrCatalogTemplate
		}
	}
	result := fmt.Sprintf(format, args...)
	if len(result) > MaxCatalogTemplateBytes {
		return "", ErrCatalogTemplate
	}
	return result, nil
}

func catalogIndent(spaces int, input string) (string, error) {
	if spaces < 0 || spaces > 128 || len(input)+(strings.Count(input, "\n")+1)*spaces > MaxCatalogTemplateBytes {
		return "", ErrCatalogTemplate
	}
	padding := strings.Repeat(" ", spaces)
	return padding + strings.ReplaceAll(input, "\n", "\n"+padding), nil
}

func catalogJoin(separator string, value interface{}) (string, error) {
	items := reflect.ValueOf(value)
	if !items.IsValid() || (items.Kind() != reflect.Array && items.Kind() != reflect.Slice) || items.Len() > 10000 {
		return "", ErrCatalogTemplate
	}
	length := 0
	result := make([]string, 0, items.Len())
	for i := 0; i < items.Len(); i++ {
		text := fmt.Sprint(items.Index(i).Interface())
		length += len(text)
		if i > 0 {
			length += len(separator)
		}
		if length > MaxCatalogTemplateBytes {
			return "", ErrCatalogTemplate
		}
		result = append(result, text)
	}
	return strings.Join(result, separator), nil
}

func catalogReplace(old, replacement, input string) (string, error) {
	count := strings.Count(input, old)
	if len(input)+count*(len(replacement)-len(old)) > MaxCatalogTemplateBytes {
		return "", ErrCatalogTemplate
	}
	return strings.ReplaceAll(input, old, replacement), nil
}

// ApplyCatalog is the opt-in, deterministic subset for a reviewed typed launch.
// The ordinary Apply owner remains unchanged. No host environment, DNS, random,
// crypto key generation, filesystem or caller-supplied function is available.
func ApplyCatalog(contents []byte, stack StackInfo, environment EnvironmentInfo, variables map[string]string) ([]byte, error) {
	if len(contents) > MaxCatalogTemplateBytes || len(variables) > 100 {
		return nil, ErrCatalogTemplate
	}
	trimmed := strings.TrimSpace(string(contents))
	if strings.HasPrefix(trimmed, "#notemplating") || strings.HasPrefix(trimmed, "# notemplating") {
		return append([]byte(nil), contents...), nil
	}
	all := sprig.HermeticTxtFuncMap()
	allowed := template.FuncMap{}
	for _, name := range strings.Fields("default empty coalesce ternary quote squote trim trimAll trimSuffix trimPrefix upper lower title untitle contains hasPrefix hasSuffix splitList first last rest initial reverse uniq without compact list dict get hasKey pick omit add add1 sub div mod mul max min int int64 float64 atoi toString toStrings b64enc b64dec sha256sum") {
		if function, ok := all[name]; ok {
			allowed[name] = function
		}
	}
	allowed["splitPreserveQuotes"] = funcs.SplitPreserveQuotes
	allowed["printf"] = catalogPrintf
	allowed["indent"] = catalogIndent
	allowed["nindent"] = func(spaces int, input string) (string, error) {
		result, err := catalogIndent(spaces, input)
		return "\n" + result, err
	}
	allowed["join"] = catalogJoin
	allowed["replace"] = catalogReplace
	allowed["print"] = func(args ...interface{}) (string, error) {
		return catalogPrintf(strings.Repeat("%v", len(args)), args...)
	}
	allowed["println"] = func(args ...interface{}) (string, error) {
		return catalogPrintf(strings.Repeat("%v ", len(args))+"\n", args...)
	}
	t, err := template.New("catalog").Option("missingkey=error").Funcs(allowed).Parse(string(contents))
	if err != nil {
		return nil, ErrCatalogTemplate
	}
	nodes := 0
	for _, defined := range t.Templates() {
		if !catalogTree(defined.Tree.Root, &nodes) {
			return nil, ErrCatalogTemplate
		}
	}
	writer := &catalogWriter{}
	if err = t.Execute(writer, map[string]interface{}{"Values": variables, "Release": stack, "Stack": stack, "Environment": environment}); err != nil {
		return nil, ErrCatalogTemplate
	}
	// The normal executor must use this exact approved rendering, not run a
	// second evaluation with its ordinary template function set.
	result := append([]byte("# notemplating\n"), writer.Bytes()...)
	if len(result) > MaxCatalogTemplateBytes {
		return nil, ErrCatalogTemplate
	}
	return result, nil
}

func catalogTree(node parse.Node, count *int) bool {
	if node == nil {
		return true
	}
	*count++
	if *count > 10000 {
		return false
	}
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return true
		}
		for _, child := range n.Nodes {
			if !catalogTree(child, count) {
				return false
			}
		}
	case *parse.TemplateNode:
		return false
	case *parse.IfNode:
		return catalogTree(n.List, count) && catalogTree(n.ElseList, count)
	case *parse.RangeNode:
		return catalogTree(n.List, count) && catalogTree(n.ElseList, count)
	case *parse.WithNode:
		return catalogTree(n.List, count) && catalogTree(n.ElseList, count)
	}
	return true
}
