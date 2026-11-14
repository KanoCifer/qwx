// Package endpoints 从源码扫出端点清单，供文档生成与漂移校验使用。
//
// 端点常量与缓存键族各有一个真源：源码里的常量声明。文档里的表是从
// 源码扫出来生成的，改了源码忘了改文档，漂移测试会失败。
package endpoints

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// tick 是 Markdown 表格里的反引号。
const tick = "\x60"

// Constant 是一个端点常量。
type Constant struct {
	// Const 是常量的引用名，如 "weather.EndpointCurrent"。
	Const string
	// Value 是常量的值，如 "weather/v1/current"。
	//
	// 值发布后不再变更：qwx.TTLPolicy 按值匹配。
	Value string
	// Package 是所属端点包的目录，如 "v1/weather"。
	Package string
	// Methods 是使用该常量的端点方法名，按字典序。
	Methods []string
}

// Family 是一个缓存键族常量。
type Family struct {
	// Const 是常量的引用名，如 "cachekey.FamilyWeather"。
	Const string
	// Prefix 是前缀根，如 "qwx:v1:"。
	//
	// 前缀与包名不必对齐（潮汐族是 "qwx:tide:" 而不是 "qwx:ocean:"），
	// 这是已发布键字面量的约定。
	Prefix string
	// Packages 是使用该族的端点包目录，按字典序。
	//
	// 空表示没有端点包引用它。
	Packages []string
}

// Index 是一次扫描的结果。
type Index struct {
	// Constants 是全部端点常量，按包目录与常量名排序。
	Constants []Constant
	// Families 是全部缓存键族，按常量名排序。
	Families []Family
}

// Scan 扫描模块根目录下的端点包与缓存键族。
func Scan(root string) (*Index, error) {
	pkgs, err := endpointPackages(root)
	if err != nil {
		return nil, err
	}
	families, err := scanFamilies(root)
	if err != nil {
		return nil, err
	}
	idx := &Index{}
	for _, pkg := range pkgs {
		consts, uses, fams, goName, err := scanPackage(root, pkg)
		if err != nil {
			return nil, err
		}
		for i := range consts {
			consts[i].Methods = uses[consts[i].Const]
			sort.Strings(consts[i].Methods)
			consts[i].Const = goName + "." + consts[i].Const
		}
		idx.Constants = append(idx.Constants, consts...)
		for name := range fams {
			fam, ok := families[name]
			if !ok {
				// 引用了不存在的族常量：不在族表里，不生成条目。
				continue
			}
			fam.Packages = append(fam.Packages, pkg)
			families[name] = fam
		}
	}
	sort.Slice(idx.Constants, func(i, j int) bool {
		if idx.Constants[i].Package != idx.Constants[j].Package {
			return idx.Constants[i].Package < idx.Constants[j].Package
		}
		return idx.Constants[i].Const < idx.Constants[j].Const
	})
	out := make([]Family, 0, len(families))
	for _, f := range families {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Const < out[j].Const })
	idx.Families = out
	return idx, nil
}

// endpointPackages 列出模块根下的端点包目录。
//
// 顶层目录里直接有 .go 文件的是端点包（geo、console）；只含子目录的是
// 代际命名空间（v1、v7），进一层再看。
func endpointPackages(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		switch e.Name() {
		case "internal", "quick", "examples", "live", "rediscache", "docs", "assets", ".git", ".github":
			continue
		}
		dir := filepath.Join(root, e.Name())
		if hasGoFiles(dir) && !hasSubPackages(dir) {
			out = append(out, e.Name())
			continue
		}
		sub, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, s := range sub {
			if s.IsDir() && hasGoFiles(filepath.Join(dir, s.Name())) {
				out = append(out, e.Name()+"/"+s.Name())
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// hasSubPackages 报告目录里是否有含 Go 文件的子目录。
//
// v1 既有自己的 Go 文件（跨端点族共用的基础类型），又装着端点包子目录：
// 它是命名空间，不是端点包。
func hasSubPackages(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && hasGoFiles(filepath.Join(dir, e.Name())) {
			return true
		}
	}
	return false
}

// hasGoFiles 报告目录里是否有非测试的 Go 文件。
func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// scanPackage 扫一个端点包：端点常量、它们被哪些方法使用、引用了哪些键族、
// 包的 Go 名。
func scanPackage(root, pkg string) ([]Constant, map[string][]string, map[string]Family, string, error) {
	dir := filepath.Join(root, pkg)
	fset := token.NewFileSet()
	parsed, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, nil, nil, "", err
	}
	var consts []Constant
	uses := map[string][]string{}
	fams := map[string]Family{}
	goName := ""
	for name, p := range parsed {
		if goName == "" {
			goName = name
		}
		for _, f := range p.Files {
			for _, decl := range f.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					if d.Tok != token.CONST {
						continue
					}
					for _, spec := range d.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						val, ok := endpointValue(vs)
						if !ok {
							continue
						}
						for _, name := range vs.Names {
							consts = append(consts, Constant{Const: name.Name, Value: val, Package: pkg})
						}
					}
				case *ast.FuncDecl:
					if d.Recv == nil {
						continue
					}
					collectRequestEndpoints(d, uses)
				}
			}
			collectFamilyRefs(f, fams)
		}
	}
	return consts, uses, fams, goName, nil
}

// endpointValue 判断一个常量声明是不是端点常量，是则取出它的值。
//
// 两种写法都要认：显式标注类型的
//
//	EndpointCurrent qwx.Endpoint = "weather/v1/current"
//
// 与直接转换的
//
//	EndpointCurrent = qwx.Endpoint("weather/v1/current")
func endpointValue(vs *ast.ValueSpec) (string, bool) {
	if vs.Type == nil && !isEndpointCall(vs.Values) {
		return "", false
	}
	if vs.Type != nil && !isEndpointType(vs.Type) {
		return "", false
	}
	if len(vs.Values) != 1 {
		return "", false
	}
	switch v := vs.Values[0].(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false
		}
		return unquote(v.Value), true
	case *ast.CallExpr:
		if !isEndpointType(v.Fun) {
			return "", false
		}
		if len(v.Args) != 1 {
			return "", false
		}
		lit, ok := v.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return "", false
		}
		return unquote(lit.Value), true
	}
	return "", false
}

// isEndpointCall 判断常量声明的值是不是 qwx.Endpoint(...) 转换。
//
// 没有显式类型标注时，只认这种写法 —— 否则同包里别的字符串常量
// （如台风流域代码 "NP"）会被误当成端点常量。
func isEndpointCall(values []ast.Expr) bool {
	if len(values) != 1 {
		return false
	}
	call, ok := values[0].(*ast.CallExpr)
	return ok && isEndpointType(call.Fun)
}

// isEndpointType 判断表达式是不是 qwx.Endpoint 类型。
func isEndpointType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "qwx" && sel.Sel.Name == "Endpoint"
}

// collectRequestEndpoints 收集函数体里 qwx.Request 字面量上的 Endpoint 字段。
func collectRequestEndpoints(fd *ast.FuncDecl, uses map[string][]string) {
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := cl.Type.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "qwx" || sel.Sel.Name != "Request" {
			return true
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Endpoint" {
				continue
			}
			val, ok := kv.Value.(*ast.Ident)
			if !ok {
				continue
			}
			uses[val.Name] = append(uses[val.Name], fd.Name.Name)
		}
		return true
	})
}

// collectFamilyRefs 收集文件里引用的 cachekey.FamilyX 常量。
func collectFamilyRefs(f *ast.File, fams map[string]Family) {
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "cachekey" {
			return true
		}
		if !strings.HasPrefix(sel.Sel.Name, "Family") {
			return true
		}
		fams[sel.Sel.Name] = Family{Const: "cachekey." + sel.Sel.Name}
		return true
	})
}

// scanFamilies 扫 internal/cachekey 里的族常量。
func scanFamilies(root string) (map[string]Family, error) {
	dir := filepath.Join(root, "internal", "cachekey")
	fset := token.NewFileSet()
	parsed, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}
	fams := map[string]Family{}
	for _, p := range parsed {
		for _, f := range p.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.CONST {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Values) != 1 {
						continue
					}
					if vs.Type != nil {
						id, ok := vs.Type.(*ast.Ident)
						if !ok || id.Name != "Family" {
							continue
						}
					}
					lit, ok := vs.Values[0].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					for _, name := range vs.Names {
						fams[name.Name] = Family{
							Const:  "cachekey." + name.Name,
							Prefix: unquote(lit.Value),
						}
					}
				}
			}
		}
	}
	return fams, nil
}

// unquote 去掉 Go 字符串字面量的引号。
func unquote(s string) string {
	if v, err := strconv.Unquote(s); err == nil {
		return v
	}
	return strings.Trim(s, "\"")
}

// ConstantsTable 渲染端点常量表。
func (idx *Index) ConstantsTable() string {
	var b strings.Builder
	b.WriteString("| 端点常量 | 值 | 包 | 端点 |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, c := range idx.Constants {
		methods := strings.Join(c.Methods, "、")
		if methods == "" {
			methods = "—"
		}
		fmt.Fprintf(&b, "| %s%s%s | %s%s%s | %s%s%s | %s |\n",
			tick, c.Const, tick, tick, c.Value, tick, tick, c.Package, tick, methods)
	}
	return b.String()
}

// FamiliesTable 渲染缓存键族表。
func (idx *Index) FamiliesTable() string {
	var b strings.Builder
	b.WriteString("| 族常量 | 前缀根 | 包 |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, f := range idx.Families {
		pkg := strings.Join(f.Packages, "、")
		if pkg == "" {
			pkg = "—"
		}
		fmt.Fprintf(&b, "| %s%s%s | %s%s%s | %s%s%s |\n",
			tick, f.Const, tick, tick, f.Prefix, tick, tick, pkg, tick)
	}
	return b.String()
}
